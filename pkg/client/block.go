package client

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc"
)

// GetNowBlock return TIP block
func (g *GrpcClient) GetNowBlock() (*api.BlockExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetNowBlockCtx(ctx)
}

// GetNowBlockCtx is the context-aware version of GetNowBlock.
func (g *GrpcClient) GetNowBlockCtx(ctx context.Context) (*api.BlockExtention, error) {
	ctx = g.withAPIKey(ctx)
	result, err := g.Client.GetNowBlock2(ctx, new(api.EmptyMessage))

	if err != nil {
		return nil, fmt.Errorf("Get block now: %w", err)
	}

	return result, nil
}

// GetBlockByNum block from number
func (g *GrpcClient) GetBlockByNum(num int64) (*api.BlockExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetBlockByNumCtx(ctx, num)
}

// GetBlockByNumCtx is the context-aware version of GetBlockByNum.
func (g *GrpcClient) GetBlockByNumCtx(ctx context.Context, num int64) (*api.BlockExtention, error) {
	ctx = g.withAPIKey(ctx)
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	result, err := g.Client.GetBlockByNum2(ctx, numMessage, maxSizeOption)

	if err != nil {
		return nil, fmt.Errorf("Get block by num: %w", err)

	}
	return result, nil
}

// GetBlockInfoByNum block from number
func (g *GrpcClient) GetBlockInfoByNum(num int64) (*api.TransactionInfoList, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetBlockInfoByNumCtx(ctx, num)
}

// GetBlockInfoByNumCtx is the context-aware version of GetBlockInfoByNum.
func (g *GrpcClient) GetBlockInfoByNumCtx(ctx context.Context, num int64) (*api.TransactionInfoList, error) {
	ctx = g.withAPIKey(ctx)
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)

	result, err := g.Client.GetTransactionInfoByBlockNum(ctx, numMessage, maxSizeOption)

	if err != nil {
		return nil, fmt.Errorf("Get block info by num: %w", err)

	}
	return result, nil
}

// GetBlockByID block from hash
func (g *GrpcClient) GetBlockByID(id string) (*core.Block, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetBlockByIDCtx(ctx, id)
}

// GetBlockByIDCtx is the context-aware version of GetBlockByID.
func (g *GrpcClient) GetBlockByIDCtx(ctx context.Context, id string) (*core.Block, error) {
	ctx = g.withAPIKey(ctx)
	blockID := new(api.BytesMessage)
	var err error

	blockID.Value, err = common.FromHex(id)
	if err != nil {
		return nil, fmt.Errorf("get block by id: %v", err)
	}

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	return g.Client.GetBlockById(ctx, blockID, maxSizeOption)
}

// GetBlockByLimitNext return list of block start/end
func (g *GrpcClient) GetBlockByLimitNext(start, end int64) (*api.BlockListExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetBlockByLimitNextCtx(ctx, start, end)
}

// GetBlockByLimitNextCtx is the context-aware version of GetBlockByLimitNext.
func (g *GrpcClient) GetBlockByLimitNextCtx(ctx context.Context, start, end int64) (*api.BlockListExtention, error) {
	ctx = g.withAPIKey(ctx)
	blockLimit := new(api.BlockLimit)
	blockLimit.StartNum = start
	blockLimit.EndNum = end

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	return g.Client.GetBlockByLimitNext2(ctx, blockLimit, maxSizeOption)
}

// GetBlockByLatestNum return block list till num
func (g *GrpcClient) GetBlockByLatestNum(num int64) (*api.BlockListExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetBlockByLatestNumCtx(ctx, num)
}

// GetBlockByLatestNumCtx is the context-aware version of GetBlockByLatestNum.
func (g *GrpcClient) GetBlockByLatestNumCtx(ctx context.Context, num int64) (*api.BlockListExtention, error) {
	ctx = g.withAPIKey(ctx)
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	return g.Client.GetBlockByLatestNum2(ctx, numMessage, maxSizeOption)
}
