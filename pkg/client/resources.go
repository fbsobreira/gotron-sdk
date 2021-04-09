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
