package client

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// ListWitnesses return all witnesses
func (g *GrpcClient) ListWitnesses() (*api.WitnessList, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.ListWitnessesCtx(ctx)
}

// ListWitnessesCtx is the context-aware version of ListWitnesses.
func (g *GrpcClient) ListWitnessesCtx(ctx context.Context) (*api.WitnessList, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.ListWitnesses(ctx, new(api.EmptyMessage))
}

// ListWitnessesPaginated returns a paginated list of current witnesses
func (g *GrpcClient) ListWitnessesPaginated(page int64, limit ...int) (*api.WitnessList, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.ListWitnessesPaginatedCtx(ctx, page, limit...)
}

// ListWitnessesPaginatedCtx is the context-aware version of ListWitnessesPaginated.
func (g *GrpcClient) ListWitnessesPaginatedCtx(ctx context.Context, page int64, limit ...int) (*api.WitnessList, error) {
	ctx = g.withAPIKey(ctx)

	useLimit := int64(10)
	if len(limit) == 1 {
		useLimit = int64(limit[0])
	}
	return g.Client.GetPaginatedNowWitnessList(ctx, GetPaginatedMessage(page*useLimit, useLimit))
}

// CreateWitness upgrade account to network witness
func (g *GrpcClient) CreateWitness(from, urlStr string) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.CreateWitnessCtx(ctx, from, urlStr)
}

// CreateWitnessCtx is the context-aware version of CreateWitness.
func (g *GrpcClient) CreateWitnessCtx(ctx context.Context, from, urlStr string) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	contract := &core.WitnessCreateContract{
		Url: []byte(urlStr),
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

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
	ctx, cancel := g.newContext()
	defer cancel()
	return g.UpdateWitnessCtx(ctx, from, urlStr)
}

// UpdateWitnessCtx is the context-aware version of UpdateWitness.
func (g *GrpcClient) UpdateWitnessCtx(ctx context.Context, from, urlStr string) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	contract := &core.WitnessUpdateContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	contract.UpdateUrl = []byte(urlStr)

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
	ctx, cancel := g.newContext()
	defer cancel()
	return g.VoteWitnessAccountCtx(ctx, from, witnessMap)
}

// VoteWitnessAccountCtx is the context-aware version of VoteWitnessAccount.
func (g *GrpcClient) VoteWitnessAccountCtx(ctx context.Context, from string,
	witnessMap map[string]int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

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
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetWitnessBrokerageCtx(ctx, witness)
}

// GetWitnessBrokerageCtx is the context-aware version of GetWitnessBrokerage.
func (g *GrpcClient) GetWitnessBrokerageCtx(ctx context.Context, witness string) (float64, error) {
	ctx = g.withAPIKey(ctx)

	addr, err := common.DecodeCheck(witness)
	if err != nil {
		return 0, err
	}

	result, err := g.Client.GetBrokerageInfo(ctx, GetMessageBytes(addr))
	if err != nil {
		return 0, err
	}
	return float64(result.Num), nil
}

// UpdateBrokerage change SR commission fees
func (g *GrpcClient) UpdateBrokerage(from string, commission int32) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.UpdateBrokerageCtx(ctx, from, commission)
}

// UpdateBrokerageCtx is the context-aware version of UpdateBrokerage.
func (g *GrpcClient) UpdateBrokerageCtx(ctx context.Context, from string, commission int32) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	contract := &core.UpdateBrokerageContract{
		Brokerage: commission,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

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
