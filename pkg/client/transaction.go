package client

import (
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// GetTransactionSignWeight queries transaction sign weight
func (g *GrpcClient) GetTransactionSignWeight(tx *core.Transaction) (*api.TransactionSignWeight, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	result, err := g.Client.GetTransactionSignWeight(ctx, tx)
	if err != nil {
		return nil, err
	}
	return result, nil
}
