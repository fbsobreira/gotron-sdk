package txbuilder

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// Client is the subset of GrpcClient that transaction builders need.
// Accepting an interface avoids import cycles and simplifies testing.
type Client interface {
	TransferCtx(ctx context.Context, from, toAddress string, amount int64) (*api.TransactionExtention, error)
	BroadcastCtx(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	GetTransactionInfoByIDCtx(ctx context.Context, id string) (*core.TransactionInfo, error)
	FreezeBalanceV2Ctx(ctx context.Context, from string, resource core.ResourceCode, frozenBalance int64) (*api.TransactionExtention, error)
	UnfreezeBalanceV2Ctx(ctx context.Context, from string, resource core.ResourceCode, unfreezeBalance int64) (*api.TransactionExtention, error)
	DelegateResourceCtx(ctx context.Context, from, to string, resource core.ResourceCode, delegateBalance int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error)
	UnDelegateResourceCtx(ctx context.Context, owner, receiver string, resource core.ResourceCode, delegateBalance int64) (*api.TransactionExtention, error)
	VoteWitnessAccountCtx(ctx context.Context, from string, witnessMap map[string]int64) (*api.TransactionExtention, error)
	WithdrawExpireUnfreezeCtx(ctx context.Context, from string, timestamp int64) (*api.TransactionExtention, error)
}
