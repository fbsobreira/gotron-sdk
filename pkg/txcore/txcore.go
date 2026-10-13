// Package txcore provides shared transaction signing, broadcasting,
// and confirmation logic used by both txbuilder and contract packages.
package txcore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/txresult"
	"google.golang.org/protobuf/proto"
)

// DefaultPollInterval is the default interval between confirmation checks.
const DefaultPollInterval = 2 * time.Second

// Broadcaster abstracts the gRPC calls needed for sending and confirming
// transactions.
type Broadcaster interface {
	BroadcastCtx(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	GetTransactionInfoByIDCtx(ctx context.Context, id string) (*core.TransactionInfo, error)
}

// ErrPQSchemeDisabled is returned by Send when the signer uses a post-quantum
// signature scheme that the connected node has not enabled.
//
// Experimental: TIP-899 post-quantum support is a draft and may change.
var ErrPQSchemeDisabled = errors.New("post-quantum signature scheme disabled on node")

// Receipt is an alias for the shared receipt type.
type Receipt = txresult.Receipt

// TransactionID computes the hex-encoded SHA-256 hash of the marshalled
// RawData, which is the canonical TRON transaction ID.
func TransactionID(tx *core.Transaction) (string, error) {
	if tx == nil || tx.RawData == nil {
		return "", fmt.Errorf("invalid transaction: missing raw data")
	}
	rawData, err := proto.Marshal(tx.RawData)
	if err != nil {
		return "", fmt.Errorf("marshal raw data: %w", err)
	}
	h := sha256.Sum256(rawData)
	return common.BytesToHexString(h[:]), nil
}

// Send signs and broadcasts a transaction, returning a Receipt.
//
// Note: When result.Code != 0 (broadcast rejected), Send returns a nil Go
// error but sets receipt.Error. Callers MUST check receipt.Error in addition
// to the returned error.
//
// Experimental: if s is a PQSchemeSigner (as signer.NewPQSigner returns) and
// b is a PQProber (as *client.GrpcClient is), Send asks the node whether the
// scheme is enabled before signing. If it is not, Send returns an error
// wrapping ErrPQSchemeDisabled without signing or broadcasting. If either
// interface is missing, no probe is made and Send behaves as it does for any
// other signer; the node still rejects a disabled scheme at broadcast. A
// Broadcaster that wraps *client.GrpcClient must implement PQProber itself
// (usually by forwarding to the client) to keep the early check.
func Send(ctx context.Context, b Broadcaster, s signer.Signer, tx *core.Transaction) (*Receipt, error) {
	if err := checkPQScheme(ctx, b, s); err != nil {
		return nil, err
	}
	signed, err := s.Sign(tx)
	if err != nil {
		return nil, fmt.Errorf("signing transaction: %w", err)
	}
	txID, err := TransactionID(signed)
	if err != nil {
		return nil, fmt.Errorf("computing tx ID: %w", err)
	}
	receipt := &Receipt{TxID: txID}
	result, err := b.BroadcastCtx(ctx, signed)
	if err != nil {
		return receipt, fmt.Errorf("broadcasting transaction: %w", err)
	}
	if result == nil {
		return receipt, fmt.Errorf("broadcasting transaction: empty response")
	}
	if result.Code != 0 {
		receipt.Error = string(result.GetMessage())
	}
	return receipt, nil
}

// PQSchemeSigner is implemented by signers that sign with a TIP-899
// post-quantum key, such as the one signer.NewPQSigner returns.
//
// Experimental: TIP-899 is a draft and is live only on the Nile testnet.
type PQSchemeSigner interface {
	PQScheme() core.PQScheme
}

// PQProber is implemented by broadcasters that can ask the node whether a
// TIP-899 post-quantum scheme is enabled, such as *client.GrpcClient.
//
// Experimental: TIP-899 is a draft and is live only on the Nile testnet.
type PQProber interface {
	PQSchemeAllowedCtx(ctx context.Context, scheme core.PQScheme) (bool, error)
}

// checkPQScheme probes the node for post-quantum capability when both the
// signer and the broadcaster opt in. It is a no-op otherwise.
func checkPQScheme(ctx context.Context, b Broadcaster, s signer.Signer) error {
	pqSigner, ok := s.(PQSchemeSigner)
	if !ok {
		return nil
	}
	prober, ok := b.(PQProber)
	if !ok {
		return nil
	}
	return probePQScheme(ctx, prober, pqSigner.PQScheme())
}

// probePQScheme asks p whether scheme is enabled on the node. It returns an
// error wrapping ErrPQSchemeDisabled if it is not.
func probePQScheme(ctx context.Context, p PQProber, scheme core.PQScheme) error {
	allowed, err := p.PQSchemeAllowedCtx(ctx, scheme)
	if err != nil {
		return fmt.Errorf("checking post-quantum capability: %w", err)
	}
	if !allowed {
		return fmt.Errorf("%w: %s", ErrPQSchemeDisabled, scheme)
	}
	return nil
}

// CheckPQSchemes asks p whether each distinct post-quantum scheme on
// tx.PqAuthSig, then each pending scheme, is enabled on the node, and returns
// an error wrapping ErrPQSchemeDisabled for one that is not. Before signing,
// pass the scheme you will sign with as pending; after signing outside Send,
// call it before BroadcastCtx. A pending scheme this SDK does not know wraps
// pqc.ErrUnsupportedScheme. With nothing to probe, p may be nil.
//
// Experimental: TIP-899 is a draft and is live only on the Nile testnet.
func CheckPQSchemes(ctx context.Context, p PQProber, tx *core.Transaction, pending ...core.PQScheme) error {
	if tx == nil {
		return fmt.Errorf("checking post-quantum capability: nil transaction")
	}
	for i, scheme := range pending {
		if _, err := pqc.MaxAuthSigSize(scheme); err != nil {
			return fmt.Errorf("checking post-quantum capability: pending[%d]: %w", i, err)
		}
	}
	schemes := make([]core.PQScheme, 0, len(tx.GetPqAuthSig())+len(pending))
	for _, sig := range tx.GetPqAuthSig() {
		schemes = append(schemes, sig.GetScheme())
	}
	schemes = append(schemes, pending...)

	seen := make(map[core.PQScheme]bool)
	for _, scheme := range schemes {
		if seen[scheme] {
			continue
		}
		seen[scheme] = true
		if p == nil {
			return fmt.Errorf("checking post-quantum capability: nil prober")
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("checking post-quantum capability: %w", err)
		}
		if err := probePQScheme(ctx, p, scheme); err != nil {
			return err
		}
	}
	return nil
}

// SendAndConfirm sends a transaction and polls until confirmed or the context
// is cancelled.
func SendAndConfirm(ctx context.Context, b Broadcaster, s signer.Signer, tx *core.Transaction, pollInterval time.Duration) (*Receipt, error) {
	receipt, err := Send(ctx, b, s, tx)
	if err != nil {
		return receipt, err
	}
	if receipt.Error != "" {
		return receipt, nil
	}
	if pollInterval <= 0 {
		pollInterval = DefaultPollInterval
	}
	return WaitForConfirmation(ctx, b, receipt, pollInterval)
}

// WaitForConfirmation polls for transaction confirmation.
func WaitForConfirmation(ctx context.Context, b Broadcaster, receipt *Receipt, pollInterval time.Duration) (*Receipt, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return receipt, fmt.Errorf("waiting for confirmation: %w", ctx.Err())
		case <-ticker.C:
			info, infoErr := b.GetTransactionInfoByIDCtx(ctx, receipt.TxID)
			if infoErr != nil {
				if strings.Contains(infoErr.Error(), "not found") {
					continue
				}
				return receipt, fmt.Errorf("checking confirmation: %w", infoErr)
			}
			if info == nil || info.GetBlockNumber() == 0 {
				continue
			}
			receipt.Confirmed = true
			receipt.BlockNumber = info.GetBlockNumber()
			receipt.Fee = info.GetFee()
			if info.GetReceipt() != nil {
				receipt.EnergyUsed = info.GetReceipt().GetEnergyUsageTotal()
				receipt.BandwidthUsed = info.GetReceipt().GetNetUsage()
			}
			if results := info.GetContractResult(); len(results) > 0 {
				receipt.Result = results[0]
			}
			if info.GetResult() != core.TransactionInfo_SUCESS {
				receipt.Error = string(info.GetResMessage())
			}
			return receipt, nil
		}
	}
}

// ApplyPermissionID sets the permission ID on all contracts in the
// transaction. Call it before signing: the permission ID is part of the
// signed digest, so changing it on a transaction that already carries
// signatures invalidates them and the node rejects it.
// transaction.Controller refuses that case (see WithPermissionID).
func ApplyPermissionID(tx *api.TransactionExtention, id int32) {
	if tx == nil || tx.Transaction == nil || tx.Transaction.RawData == nil {
		return
	}
	for _, c := range tx.Transaction.RawData.Contract {
		c.PermissionId = id
	}
}

// ApplyMemo sets the memo on the transaction.
func ApplyMemo(tx *api.TransactionExtention, memo string) {
	if tx == nil || tx.Transaction == nil || tx.Transaction.RawData == nil {
		return
	}
	tx.Transaction.RawData.Data = []byte(memo)
}

// RecomputeTxID recomputes the transaction ID after mutations.
func RecomputeTxID(tx *api.TransactionExtention) error {
	if tx == nil || tx.Transaction == nil || tx.Transaction.RawData == nil {
		return fmt.Errorf("invalid transaction: missing raw data")
	}
	raw, err := proto.Marshal(tx.Transaction.RawData)
	if err != nil {
		return fmt.Errorf("recomputing txid: %w", err)
	}
	h := sha256.Sum256(raw)
	tx.Txid = h[:]
	return nil
}
