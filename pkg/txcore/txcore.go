// Package txcore provides shared transaction signing, broadcasting,
// and confirmation logic used by both txbuilder and contract packages.
package txcore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
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
func Send(ctx context.Context, b Broadcaster, s signer.Signer, tx *core.Transaction) (*Receipt, error) {
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
// transaction.
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
