// Package txbuilder provides a fluent builder for native TRON transactions
// (transfers, staking, voting, etc.) with Build / Send / SendAndConfirm
// terminal operations.
package txbuilder

import (
	"context"
	"sync/atomic"

	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/txcore"
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
//
// A Tx is single-use: its Build, Send, or SendAndConfirm method may only be
// called once. Calling any terminal a second time returns an error. This
// prevents accidentally broadcasting the same transaction twice or getting
// unexpected results from a stale builder state.
//
// To create multiple transactions of the same type, call the Builder method
// (e.g. Transfer, FreezeV2) again to obtain a fresh Tx.
type Tx struct {
	client  Client
	cfg     config
	buildFn func(ctx context.Context) (*api.TransactionExtention, error)
	used    atomic.Int32 // 0 = unused, 1 = already built
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
//
// Build may only be called once per Tx. Subsequent calls return
// ErrAlreadyBuilt. Because Send and SendAndConfirm call Build internally,
// calling any terminal method consumes the Tx.
func (t *Tx) Build(ctx context.Context) (*api.TransactionExtention, error) {
	if !t.used.CompareAndSwap(0, 1) {
		return nil, ErrAlreadyBuilt
	}

	tx, err := t.buildFn(ctx)
	if err != nil {
		return nil, err
	}

	if tx.Transaction == nil || tx.Transaction.RawData == nil {
		return nil, ErrMissingRawData
	}

	mutated := false
	if t.cfg.permissionID != nil {
		txcore.ApplyPermissionID(tx, *t.cfg.permissionID)
		mutated = true
	}

	if t.cfg.memo != "" {
		txcore.ApplyMemo(tx, t.cfg.memo)
		mutated = true
	}

	// Recompute Txid after mutations so it matches the final RawData.
	if mutated {
		if err := txcore.RecomputeTxID(tx); err != nil {
			return nil, err
		}
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
// populated from the broadcast response. Like Build, it may only be called
// once per Tx.
func (t *Tx) Send(ctx context.Context, s signer.Signer) (*Receipt, error) {
	ext, err := t.Build(ctx)
	if err != nil {
		return nil, err
	}
	return txcore.Send(ctx, t.client, s, ext.Transaction)
}

// SendAndConfirm is like Send but additionally polls GetTransactionInfoByID
// until the transaction is confirmed or the context is cancelled. Like Build,
// it may only be called once per Tx.
func (t *Tx) SendAndConfirm(ctx context.Context, s signer.Signer) (*Receipt, error) {
	ext, err := t.Build(ctx)
	if err != nil {
		return nil, err
	}
	return txcore.SendAndConfirm(ctx, t.client, s, ext.Transaction, t.cfg.pollInterval)
}
