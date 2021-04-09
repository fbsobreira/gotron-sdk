package client

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
)

// FreezeBalance from base58 address
func (g *GrpcClient) FreezeBalance(from, delegateTo string,
	resource core.ResourceCode, frozenBalance int64) (*api.TransactionExtention, error) {
	var err error

	contract := &core.FreezeBalanceContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	contract.FrozenBalance = frozenBalance
	contract.FrozenDuration = 3 // Tron Only allows 3 days freeze

	if len(delegateTo) > 0 {
		if contract.ReceiverAddress, err = common.DecodeCheck(delegateTo); err != nil {
			return nil, err
		}

	}
	contract.Resource = resource

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.FreezeBalance2(ctx, contract)
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

// UnfreezeBalance from base58 address
func (g *GrpcClient) UnfreezeBalance(from, delegateTo string, resource core.ResourceCode) (*api.TransactionExtention, error) {
	var err error

	contract := &core.UnfreezeBalanceContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	if len(delegateTo) > 0 {
		if contract.ReceiverAddress, err = common.DecodeCheck(delegateTo); err != nil {
			return nil, err
		}

	}
	contract.Resource = resource

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.UnfreezeBalance2(ctx, contract)
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
