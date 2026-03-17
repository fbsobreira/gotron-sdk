package client

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// GetAccountResource from BASE58 address
func (g *GrpcClient) GetAccountResource(addr string) (*api.AccountResourceMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetAccountResourceCtx(ctx, addr)
}

// GetAccountResourceCtx is the context-aware version of GetAccountResource.
func (g *GrpcClient) GetAccountResourceCtx(ctx context.Context, addr string) (*api.AccountResourceMessage, error) {
	ctx = g.withAPIKey(ctx)

	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	return g.Client.GetAccountResource(ctx, account)
}

// GetDelegatedResources from BASE58 address
func (g *GrpcClient) GetDelegatedResources(address string) ([]*api.DelegatedResourceList, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetDelegatedResourcesCtx(ctx, address)
}

// GetDelegatedResourcesCtx is the context-aware version of GetDelegatedResources.
func (g *GrpcClient) GetDelegatedResourcesCtx(ctx context.Context, address string) ([]*api.DelegatedResourceList, error) {
	ctx = g.withAPIKey(ctx)

	addrBytes, err := common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ai, err := g.Client.GetDelegatedResourceAccountIndex(ctx, GetMessageBytes(addrBytes))
	if err != nil {
		return nil, err
	}
	result := make([]*api.DelegatedResourceList, len(ai.GetToAccounts()))
	for i, addrTo := range ai.GetToAccounts() {
		dm := &api.DelegatedResourceMessage{
			FromAddress: addrBytes,
			ToAddress:   addrTo,
		}
		resource, err := g.Client.GetDelegatedResource(ctx, dm)
		if err != nil {
			return nil, err

		}
		result[i] = resource
	}
	return result, nil
}

// GetDelegatedResourcesV2 from BASE58 address
func (g *GrpcClient) GetDelegatedResourcesV2(address string) ([]*api.DelegatedResourceList, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetDelegatedResourcesV2Ctx(ctx, address)
}

// GetDelegatedResourcesV2Ctx is the context-aware version of GetDelegatedResourcesV2.
func (g *GrpcClient) GetDelegatedResourcesV2Ctx(ctx context.Context, address string) ([]*api.DelegatedResourceList, error) {
	ctx = g.withAPIKey(ctx)

	addrBytes, err := common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ai, err := g.Client.GetDelegatedResourceAccountIndexV2(ctx, GetMessageBytes(addrBytes))
	if err != nil {
		return nil, err
	}

	result := make([]*api.DelegatedResourceList, len(ai.GetToAccounts()))
	for i, addrTo := range ai.GetToAccounts() {
		dm := &api.DelegatedResourceMessage{
			FromAddress: addrBytes,
			ToAddress:   addrTo,
		}
		resource, err := g.Client.GetDelegatedResourceV2(ctx, dm)
		if err != nil {
			return nil, err

		}
		result[i] = resource
	}
	return result, nil
}

// GetReceivedDelegatedResourcesV2 returns resources delegated to the given
// BASE58 address by other accounts (Stake 2.0).
func (g *GrpcClient) GetReceivedDelegatedResourcesV2(address string) ([]*api.DelegatedResourceList, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetReceivedDelegatedResourcesV2Ctx(ctx, address)
}

// GetReceivedDelegatedResourcesV2Ctx is the context-aware version of GetReceivedDelegatedResourcesV2.
func (g *GrpcClient) GetReceivedDelegatedResourcesV2Ctx(ctx context.Context, address string) ([]*api.DelegatedResourceList, error) {
	ctx = g.withAPIKey(ctx)

	addrBytes, err := common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ai, err := g.Client.GetDelegatedResourceAccountIndexV2(ctx, GetMessageBytes(addrBytes))
	if err != nil {
		return nil, err
	}

	result := make([]*api.DelegatedResourceList, len(ai.GetFromAccounts()))
	for i, addrFrom := range ai.GetFromAccounts() {
		dm := &api.DelegatedResourceMessage{
			FromAddress: addrFrom,
			ToAddress:   addrBytes,
		}
		resource, err := g.Client.GetDelegatedResourceV2(ctx, dm)
		if err != nil {
			return nil, err
		}
		result[i] = resource
	}
	return result, nil
}

// GetCanDelegatedMaxSize from BASE58 address
func (g *GrpcClient) GetCanDelegatedMaxSize(address string, resource int32) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetCanDelegatedMaxSizeCtx(ctx, address, resource)
}

// GetCanDelegatedMaxSizeCtx is the context-aware version of GetCanDelegatedMaxSize.
func (g *GrpcClient) GetCanDelegatedMaxSizeCtx(ctx context.Context, address string, resource int32) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	ctx = g.withAPIKey(ctx)

	addrBytes, err := common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	dm := &api.CanDelegatedMaxSizeRequestMessage{}

	dm.Type = resource
	dm.OwnerAddress = addrBytes

	response, err := g.Client.GetCanDelegatedMaxSize(ctx, dm)
	if err != nil {
		return nil, err

	}

	return response, nil
}

// DelegateResource from BASE58 address
func (g *GrpcClient) DelegateResource(from, to string, resource core.ResourceCode, delegateBalance int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.DelegateResourceCtx(ctx, from, to, resource, delegateBalance, lock, lockPeriod)
}

// DelegateResourceCtx is the context-aware version of DelegateResource.
func (g *GrpcClient) DelegateResourceCtx(ctx context.Context, from, to string, resource core.ResourceCode, delegateBalance int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	addrFromBytes, err := common.DecodeCheck(from)
	if err != nil {
		return nil, err
	}

	addrToBytes, err := common.DecodeCheck(to)
	if err != nil {
		return nil, err
	}

	contract := &core.DelegateResourceContract{}

	contract.Resource = resource
	contract.OwnerAddress = addrFromBytes
	contract.ReceiverAddress = addrToBytes
	contract.Balance = delegateBalance
	contract.Lock = lock
	contract.LockPeriod = lockPeriod

	response, err := g.Client.DelegateResource(ctx, contract)
	if err != nil {
		return nil, err

	}

	return response, nil
}

// UnDelegateResource from BASE58 address
func (g *GrpcClient) UnDelegateResource(owner, receiver string, resource core.ResourceCode, delegateBalance int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.UnDelegateResourceCtx(ctx, owner, receiver, resource, delegateBalance)
}

// UnDelegateResourceCtx is the context-aware version of UnDelegateResource.
func (g *GrpcClient) UnDelegateResourceCtx(ctx context.Context, owner, receiver string, resource core.ResourceCode, delegateBalance int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	addrOwnerBytes, err := common.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	addrReceiverBytes, err := common.DecodeCheck(receiver)
	if err != nil {
		return nil, err
	}

	contract := &core.UnDelegateResourceContract{}

	contract.Resource = resource
	contract.OwnerAddress = addrOwnerBytes
	contract.ReceiverAddress = addrReceiverBytes
	contract.Balance = delegateBalance

	response, err := g.Client.UnDelegateResource(ctx, contract)
	if err != nil {
		return nil, err

	}

	return response, nil
}
