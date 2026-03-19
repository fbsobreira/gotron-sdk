package txbuilder

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

// Transfer creates a TRX transfer transaction.
func (b *Builder) Transfer(from, to string, amount int64, opts ...Option) *Tx {
	return b.newTx(func(ctx context.Context) (*api.TransactionExtention, error) {
		return b.client.TransferCtx(ctx, from, to, amount)
	}, opts)
}
