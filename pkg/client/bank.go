package client

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
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

// FreezeBalance from base58 address
func (g *GrpcClient) FreezeBalanceV2(from string,
	resource core.ResourceCode, frozenBalance int64) (*api.TransactionExtention, error) {
	var err error

	contract := &core.FreezeBalanceV2Contract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	contract.FrozenBalance = frozenBalance
	contract.Resource = resource

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.FreezeBalanceV2(ctx, contract)
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

// UnfreezeBalance from base58 address
func (g *GrpcClient) UnfreezeBalanceV2(from string, resource core.ResourceCode, unfreezeBalance int64) (*api.TransactionExtention, error) {
	var err error

	contract := &core.UnfreezeBalanceV2Contract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	contract.UnfreezeBalance = unfreezeBalance
	contract.Resource = resource

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.UnfreezeBalanceV2(ctx, contract)
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

// GetAvailableUnfreezeCount from base58 address
func (g *GrpcClient) GetAvailableUnfreezeCount(from string) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	var err error

	contract := &api.GetAvailableUnfreezeCountRequestMessage{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.GetAvailableUnfreezeCount(ctx, contract)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// GetCanWithdrawUnfreezeAmount from base58 address
func (g *GrpcClient) GetCanWithdrawUnfreezeAmount(from string, timestamp int64) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	var err error

	contract := &api.CanWithdrawUnfreezeAmountRequestMessage{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	contract.Timestamp = timestamp

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.GetCanWithdrawUnfreezeAmount(ctx, contract)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// WithdrawExpireUnfreeze from base58 address
func (g *GrpcClient) WithdrawExpireUnfreeze(from string, timestamp int64) (*api.TransactionExtention, error) {
	var err error

	contract := &core.WithdrawExpireUnfreezeContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.WithdrawExpireUnfreeze(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	return tx, nil
}
