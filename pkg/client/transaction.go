package client

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// GetTransactionSignWeight queries transaction sign weight
func (g *GrpcClient) GetTransactionSignWeight(tx *core.Transaction) (*api.TransactionSignWeight, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetTransactionSignWeightCtx(ctx, tx)
}

// GetTransactionSignWeightCtx is the context-aware version of GetTransactionSignWeight.
func (g *GrpcClient) GetTransactionSignWeightCtx(ctx context.Context, tx *core.Transaction) (*api.TransactionSignWeight, error) {
	ctx = g.withAPIKey(ctx)

	result, err := g.Client.GetTransactionSignWeight(ctx, tx)
	if err != nil {
		return nil, err
	}
	return result, nil
}
