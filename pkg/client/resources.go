package client

import (
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// GetAccountResource from BASE58 address
func (g *GrpcClient) GetAccountResource(addr string) (*api.AccountResourceMessage, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.GetAccountResource(ctx, account)
}

// GetDelegatedResources from BASE58 address
func (g *GrpcClient) GetDelegatedResources(address string) ([]*api.DelegatedResourceList, error) {
	addrBytes, err := common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}
	ctx, cancel := g.getContext()
	defer cancel()

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
	addrBytes, err := common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}
	ctx, cancel := g.getContext()
	defer cancel()

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

// GetCanDelegatedMaxSize from BASE58 address
func (g *GrpcClient) GetCanDelegatedMaxSize(address string, resource int32) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	addrBytes, err := common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}
	ctx, cancel := g.getContext()
	defer cancel()

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
	addrFromBytes, err := common.DecodeCheck(from)
	if err != nil {
		return nil, err
	}

	addrToBytes, err := common.DecodeCheck(to)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

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
func (g *GrpcClient) UnDelegateResource(owner, receiver string, resource core.ResourceCode, delegateBalance int64, lock bool) (*api.TransactionExtention, error) {
	addrOwnerBytes, err := common.DecodeCheck(owner)
	if err != nil {
		return nil, err
	}

	addrReceiverBytes, err := common.DecodeCheck(receiver)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

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
