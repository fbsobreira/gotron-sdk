package client

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// ListWitnesses return all witnesses
func (g *GrpcClient) ListWitnesses() (*api.WitnessList, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.ListWitnesses(ctx, new(api.EmptyMessage))
}

// CreateWitness upgrade account to network witness
func (g *GrpcClient) CreateWitness(from, urlStr string) (*api.TransactionExtention, error) {
	var err error

	contract := &core.WitnessCreateContract{
		Url: []byte(urlStr),
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
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

	ctx, cancel := g.getContext()
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
	witnessMap map[string]int64) (*api.TransactionExtention, error) {
	var err error

	contract := &core.VoteWitnessContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	for key, value := range witnessMap {
		if witnessAddress, err := common.DecodeCheck(key); err == nil {
			vote := &core.VoteWitnessContract_Vote{
				VoteAddress: witnessAddress,
				VoteCount:   value,
			}
			contract.Votes = append(contract.Votes, vote)

		} else {
			return nil, err
		}
	}

	ctx, cancel := g.getContext()
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

// GetWitnessBrokerage from witness address
func (g *GrpcClient) GetWitnessBrokerage(witness string) (float64, error) {
	addr, err := common.DecodeCheck(witness)
	if err != nil {
		return 0, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	result, err := g.Client.GetBrokerageInfo(ctx, GetMessageBytes(addr))
	if err != nil {
		return 0, err
	}
	return float64(result.Num), nil
}

// UpdateBrokerage change SR comission fees
func (g *GrpcClient) UpdateBrokerage(from string, comission int32) (*api.TransactionExtention, error) {
	var err error

	contract := &core.UpdateBrokerageContract{
		Brokerage: comission,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.UpdateBrokerage(ctx, contract)
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
