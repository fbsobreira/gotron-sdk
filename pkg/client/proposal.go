package client

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// ProposalsList return all network proposals
func (g *GrpcClient) ProposalsList() (*api.ProposalList, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.ProposalsListCtx(ctx)
}

// ProposalsListCtx is the context-aware version of ProposalsList.
func (g *GrpcClient) ProposalsListCtx(ctx context.Context) (*api.ProposalList, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.ListProposals(ctx, new(api.EmptyMessage))
}

// ProposalCreate create proposal based on parameter list
func (g *GrpcClient) ProposalCreate(from string, parameters map[int64]int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.ProposalCreateCtx(ctx, from, parameters)
}

// ProposalCreateCtx is the context-aware version of ProposalCreate.
func (g *GrpcClient) ProposalCreateCtx(ctx context.Context, from string, parameters map[int64]int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	contract := &core.ProposalCreateContract{
		Parameters: parameters,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	tx, err := g.Client.ProposalCreate(ctx, contract)
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

// ProposalApprove change URL info
func (g *GrpcClient) ProposalApprove(from string, id int64, confirm bool) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.ProposalApproveCtx(ctx, from, id, confirm)
}

// ProposalApproveCtx is the context-aware version of ProposalApprove.
func (g *GrpcClient) ProposalApproveCtx(ctx context.Context, from string, id int64, confirm bool) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	contract := &core.ProposalApproveContract{
		ProposalId:    id,
		IsAddApproval: confirm,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	tx, err := g.Client.ProposalApprove(ctx, contract)
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

// ProposalWithdraw withdraws a proposal.
func (g *GrpcClient) ProposalWithdraw(from string, id int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.ProposalWithdrawCtx(ctx, from, id)
}

// ProposalWithdrawCtx is the context-aware version of ProposalWithdraw.
func (g *GrpcClient) ProposalWithdrawCtx(ctx context.Context, from string, id int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	var err error

	contract := &core.ProposalDeleteContract{
		ProposalId: id,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	tx, err := g.Client.ProposalDelete(ctx, contract)
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
