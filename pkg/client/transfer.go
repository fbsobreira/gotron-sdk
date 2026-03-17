package client

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// Transfer from to base58 address
func (g *GrpcClient) Transfer(from, toAddress string, amount int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TransferCtx(ctx, from, toAddress, amount)
}

// TransferCtx is the context-aware version of Transfer.
func (g *GrpcClient) TransferCtx(ctx context.Context, from, toAddress string, amount int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)
	var err error

	contract := &core.TransferContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	if contract.ToAddress, err = common.DecodeCheck(toAddress); err != nil {
		return nil, err
	}
	contract.Amount = amount

	tx, err := g.Client.CreateTransaction2(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}
