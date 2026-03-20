// Package txbuilder provides a fluent builder for native TRON transactions
// (transfers, staking, voting, etc.) with Build / Send / SendAndConfirm
// terminal operations.
package txbuilder

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"google.golang.org/protobuf/proto"
)

// Builder is the entry point for constructing native TRON transactions.
// Create one with New, then call transaction methods (Transfer, FreezeV2, etc.)
// to get a Tx with terminal operations.
type Builder struct {
	client   Client
	defaults config
}

// New creates a Builder bound to the given client. Options set shared defaults
// that apply to every transaction produced by this builder.
func New(client Client, opts ...Option) *Builder {
	return &Builder{
		client:   client,
		defaults: applyOptions(opts),
	}
}

// Tx represents a single prepared transaction with terminal operations.
type Tx struct {
	client  Client
	cfg     config
	buildFn func(ctx context.Context) (*api.TransactionExtention, error)
}

// newTx creates a Tx that inherits the builder's defaults, with per-call
// options merged on top.
func (b *Builder) newTx(buildFn func(ctx context.Context) (*api.TransactionExtention, error), opts []Option) *Tx {
	cfg := b.defaults
	for _, o := range opts {
		o(&cfg)
	}
	return &Tx{
		client:  b.client,
		cfg:     cfg,
		buildFn: buildFn,
	}
}

// WithMemo attaches a memo to this transaction.
// Returns itself for chaining.
func (t *Tx) WithMemo(memo string) *Tx {
	t.cfg.memo = memo
	return t
}

// WithPermissionID sets the permission ID for multi-signature transactions.
// Returns itself for chaining.
func (t *Tx) WithPermissionID(id int32) *Tx {
	t.cfg.permissionID = &id
	return t
}

// Build creates the unsigned transaction, applying any configured options
// (permission ID, memo, etc.).
func (t *Tx) Build(ctx context.Context) (*api.TransactionExtention, error) {
	tx, err := t.buildFn(ctx)
	if err != nil {
		return nil, err
	}

	if tx.Transaction == nil || tx.Transaction.RawData == nil {
		return nil, fmt.Errorf("invalid transaction: missing raw data")
	}

	if t.cfg.permissionID != nil {
		for _, c := range tx.Transaction.RawData.Contract {
			c.PermissionId = *t.cfg.permissionID //nolint:staticcheck // proto generated field name
		}
	}

	if t.cfg.memo != "" {
		tx.Transaction.RawData.Data = []byte(t.cfg.memo)
	}

	// Recompute Txid after mutations so it matches the final RawData.
	if t.cfg.permissionID != nil || t.cfg.memo != "" {
		raw, err := proto.Marshal(tx.Transaction.RawData)
		if err != nil {
			return nil, fmt.Errorf("recomputing txid: %w", err)
		}
		h := sha256.Sum256(raw)
		tx.Txid = h[:]
	}

	return tx, nil
}

// Sign builds and signs the transaction without broadcasting. Returns the
// signed transaction ready for deferred broadcast or inspection.
func (t *Tx) Sign(ctx context.Context, s signer.Signer) (*core.Transaction, error) {
	ext, err := t.Build(ctx)
	if err != nil {
		return nil, err
	}
	return s.Sign(ext.Transaction)
}

// Decode builds the transaction and decodes the first contract parameter into
// human-readable fields (base58 addresses, TRX-formatted amounts). Useful for
// inspecting or displaying what a transaction does before signing.
func (t *Tx) Decode(ctx context.Context) (*transaction.ContractData, error) {
	ext, err := t.Build(ctx)
	if err != nil {
		return nil, err
	}
	return transaction.DecodeContractData(ext.Transaction)
}

// Send builds, signs, and broadcasts the transaction. It returns a Receipt
// populated from the broadcast response.
func (t *Tx) Send(ctx context.Context, s signer.Signer) (*Receipt, error) {
	ext, err := t.Build(ctx)
	if err != nil {
		return nil, err
	}

	signed, err := s.Sign(ext.Transaction)
	if err != nil {
		return nil, fmt.Errorf("signing transaction: %w", err)
	}

	txID, err := transactionID(signed)
	if err != nil {
		return nil, fmt.Errorf("computing tx ID: %w", err)
	}

	receipt := &Receipt{TxID: txID}

	result, err := t.client.BroadcastCtx(ctx, signed)
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

// SendAndConfirm is like Send but additionally polls GetTransactionInfoByID
// until the transaction is confirmed or the context is cancelled.
func (t *Tx) SendAndConfirm(ctx context.Context, s signer.Signer) (*Receipt, error) {
	receipt, err := t.Send(ctx, s)
	if err != nil {
		return receipt, err
	}
	if receipt.Error != "" {
		return receipt, nil
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return receipt, fmt.Errorf("waiting for confirmation: %w", ctx.Err())
		case <-ticker.C:
			info, infoErr := t.client.GetTransactionInfoByIDCtx(ctx, receipt.TxID)
			if infoErr != nil {
				// "not found" is transient — tx not indexed yet. Other errors are permanent.
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
			if info.GetResult() != 0 {
				receipt.Error = string(info.GetResMessage())
			}
			return receipt, nil
		}
	}
}

// transactionID computes the hex-encoded SHA-256 hash of the marshalled
// RawData, which is the canonical TRON transaction ID.
func transactionID(tx *core.Transaction) (string, error) {
	raw, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(raw)
	return common.BytesToHexString(h[:]), nil
}
