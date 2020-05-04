package client

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// TriggerConstantContract and return tx result
func (g *GrpcClient) TriggerConstantContract(ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	return g.Client.TriggerConstantContract(ctx, ct)
}
