package client

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
)

// GetDelegatedResource from BASE58 address
func (g *GrpcClient) GetDelegatedResources(address string) ([]*api.DelegatedResourceList, error) {
	addrBytes, err := common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
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
