package contract

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// Client is the subset of GrpcClient that the contract call builder needs.
// Accepting an interface avoids tight coupling and simplifies testing.
type Client interface {
	TriggerConstantContractCtx(ctx context.Context, from, contractAddress, method, jsonString string, opts ...client.ConstantCallOption) (*api.TransactionExtention, error)
	TriggerContractCtx(ctx context.Context, from, contractAddress, method, jsonString string, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error)
	TriggerConstantContractWithDataCtx(ctx context.Context, from, contractAddress string, data []byte, opts ...client.ConstantCallOption) (*api.TransactionExtention, error)
	TriggerContractWithDataCtx(ctx context.Context, from, contractAddress string, data []byte, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error)
	EstimateEnergyCtx(ctx context.Context, from, contractAddress, method, jsonString string, tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error)
	BroadcastCtx(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	GetTransactionInfoByIDCtx(ctx context.Context, id string) (*core.TransactionInfo, error)
}
