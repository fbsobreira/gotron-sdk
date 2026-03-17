package client

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// FreezeBalance from base58 address
func (g *GrpcClient) FreezeBalance(from, delegateTo string,
	resource core.ResourceCode, frozenBalance int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.FreezeBalanceCtx(ctx, from, delegateTo, resource, frozenBalance)
}

// FreezeBalanceCtx is the context-aware version of FreezeBalance.
func (g *GrpcClient) FreezeBalanceCtx(ctx context.Context, from, delegateTo string,
	resource core.ResourceCode, frozenBalance int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

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

// FreezeBalanceV2 freezes balance from base58 address.
func (g *GrpcClient) FreezeBalanceV2(from string,
	resource core.ResourceCode, frozenBalance int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.FreezeBalanceV2Ctx(ctx, from, resource, frozenBalance)
}

// FreezeBalanceV2Ctx is the context-aware version of FreezeBalanceV2.
func (g *GrpcClient) FreezeBalanceV2Ctx(ctx context.Context, from string,
	resource core.ResourceCode, frozenBalance int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	if frozenBalance <= 0 {
		return nil, fmt.Errorf("freeze balance must be positive, got %d", frozenBalance)
	}

	var err error

	contract := &core.FreezeBalanceV2Contract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	contract.FrozenBalance = frozenBalance
	contract.Resource = resource

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
	ctx, cancel := g.newContext()
	defer cancel()
	return g.UnfreezeBalanceCtx(ctx, from, delegateTo, resource)
}

// UnfreezeBalanceCtx is the context-aware version of UnfreezeBalance.
func (g *GrpcClient) UnfreezeBalanceCtx(ctx context.Context, from, delegateTo string, resource core.ResourceCode) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

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

// UnfreezeBalanceV2 unfreezes balance from base58 address.
func (g *GrpcClient) UnfreezeBalanceV2(from string, resource core.ResourceCode, unfreezeBalance int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.UnfreezeBalanceV2Ctx(ctx, from, resource, unfreezeBalance)
}

// UnfreezeBalanceV2Ctx is the context-aware version of UnfreezeBalanceV2.
func (g *GrpcClient) UnfreezeBalanceV2Ctx(ctx context.Context, from string, resource core.ResourceCode, unfreezeBalance int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	if unfreezeBalance <= 0 {
		return nil, fmt.Errorf("unfreeze balance must be positive, got %d", unfreezeBalance)
	}

	var err error

	contract := &core.UnfreezeBalanceV2Contract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	contract.UnfreezeBalance = unfreezeBalance
	contract.Resource = resource

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
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetAvailableUnfreezeCountCtx(ctx, from)
}

// GetAvailableUnfreezeCountCtx is the context-aware version of GetAvailableUnfreezeCount.
func (g *GrpcClient) GetAvailableUnfreezeCountCtx(ctx context.Context, from string) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	contract := &api.GetAvailableUnfreezeCountRequestMessage{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	tx, err := g.Client.GetAvailableUnfreezeCount(ctx, contract)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// GetCanWithdrawUnfreezeAmount from base58 address
func (g *GrpcClient) GetCanWithdrawUnfreezeAmount(from string, timestamp int64) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetCanWithdrawUnfreezeAmountCtx(ctx, from, timestamp)
}

// GetCanWithdrawUnfreezeAmountCtx is the context-aware version of GetCanWithdrawUnfreezeAmount.
func (g *GrpcClient) GetCanWithdrawUnfreezeAmountCtx(ctx context.Context, from string, timestamp int64) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	contract := &api.CanWithdrawUnfreezeAmountRequestMessage{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	contract.Timestamp = timestamp

	tx, err := g.Client.GetCanWithdrawUnfreezeAmount(ctx, contract)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// WithdrawExpireUnfreeze from base58 address
func (g *GrpcClient) WithdrawExpireUnfreeze(from string, timestamp int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.WithdrawExpireUnfreezeCtx(ctx, from, timestamp)
}

// WithdrawExpireUnfreezeCtx is the context-aware version of WithdrawExpireUnfreeze.
func (g *GrpcClient) WithdrawExpireUnfreezeCtx(ctx context.Context, from string, timestamp int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	contract := &core.WithdrawExpireUnfreezeContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	tx, err := g.Client.WithdrawExpireUnfreeze(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	return tx, nil
}
