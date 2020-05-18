package client

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
)

// ExchangeCreate from two tokens (TRC10/TRX) only
func (g *GrpcClient) ExchangeCreate(
	from string,
	tokenID1 string,
	amountToken1 int64,
	tokenID2 string,
	amountToken2 int64,
) (*api.TransactionExtention, error) {
	var err error

	contract := &core.ExchangeCreateContract{
		FirstTokenId:       []byte(tokenID1),
		FirstTokenBalance:  amountToken1,
		SecondTokenId:      []byte(tokenID2),
		SecondTokenBalance: amountToken2,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	tx, err := g.Client.ExchangeCreate(ctx, contract)
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
