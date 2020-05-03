package client

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
)

// ListWitnesses return all witnesses
func (g *GrpcClient) ListWitnesses() (*api.WitnessList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	return g.Client.ListWitnesses(ctx, new(api.EmptyMessage))
}

// CreateWitness upgrade account to network witness
func (g *GrpcClient) CreateWitness(from, urlStr string) (*api.TransactionExtention, error) {
	var err error

	contract := &core.WitnessCreateContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	contract.Url = []byte(urlStr)

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	tx, err := g.Client.CreateWitness2(ctx, contract)
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

// UpdateWitness change URL info
func (g *GrpcClient) UpdateWitness(from, urlStr string) (*api.TransactionExtention, error) {
	var err error

	contract := &core.WitnessUpdateContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	contract.UpdateUrl = []byte(urlStr)

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	tx, err := g.Client.UpdateWitness2(ctx, contract)
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

// VoteWitnessAccount change account vote
func (g *GrpcClient) VoteWitnessAccount(from string,
	witnessMap map[string]string) (*api.TransactionExtention, error) {
	var err error

	contract := &core.VoteWitnessContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	for key, value := range witnessMap {

		if witnessAddress, err := common.DecodeCheck(key); err == nil {
			if voteCount, err := strconv.ParseInt(value, 64, 10); err == nil {
				vote := &core.VoteWitnessContract_Vote{
					VoteAddress: witnessAddress,
					VoteCount:   voteCount,
				}
				contract.Votes = append(contract.Votes, vote)
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	tx, err := g.Client.VoteWitnessAccount2(ctx, contract)
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
