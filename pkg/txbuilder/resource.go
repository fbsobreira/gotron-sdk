package txbuilder

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// DelegateTx is a delegation transaction builder with a fluent Lock method.
// It embeds *Tx so all terminal operations (Build, Send, SendAndConfirm) are
// available directly.
type DelegateTx struct {
	*Tx
	lock       bool
	lockPeriod int64
}

// Lock enables the delegation lock with the given period (in blocks).
// Returns itself for chaining.
func (d *DelegateTx) Lock(period int64) *DelegateTx {
	d.lock = true
	d.lockPeriod = period
	return d
}

// FreezeV2 creates a Stake 2.0 freeze transaction.
func (b *Builder) FreezeV2(from string, amount int64, resource core.ResourceCode, opts ...Option) *Tx {
	return b.newTx(func(ctx context.Context) (*api.TransactionExtention, error) {
		return b.client.FreezeBalanceV2Ctx(ctx, from, resource, amount)
	}, opts)
}

// UnfreezeV2 creates a Stake 2.0 unfreeze transaction.
func (b *Builder) UnfreezeV2(from string, amount int64, resource core.ResourceCode, opts ...Option) *Tx {
	return b.newTx(func(ctx context.Context) (*api.TransactionExtention, error) {
		return b.client.UnfreezeBalanceV2Ctx(ctx, from, resource, amount)
	}, opts)
}

// DelegateResource creates a resource delegation transaction.
// Use .Lock(period) on the returned DelegateTx to enable delegation locking.
func (b *Builder) DelegateResource(from, to string, resource core.ResourceCode, amount int64, opts ...Option) *DelegateTx {
	dt := &DelegateTx{}
	dt.Tx = b.newTx(func(ctx context.Context) (*api.TransactionExtention, error) {
		return b.client.DelegateResourceCtx(ctx, from, to, resource, amount, dt.lock, dt.lockPeriod)
	}, opts)
	return dt
}

// UnDelegateResource creates a resource un-delegation transaction.
func (b *Builder) UnDelegateResource(from, to string, resource core.ResourceCode, amount int64, opts ...Option) *Tx {
	return b.newTx(func(ctx context.Context) (*api.TransactionExtention, error) {
		return b.client.UnDelegateResourceCtx(ctx, from, to, resource, amount)
	}, opts)
}
