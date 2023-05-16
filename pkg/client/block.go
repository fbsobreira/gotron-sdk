package client

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc"
)

// GetNowBlock return TIP block
func (g *GrpcClient) GetNowBlock() (*api.BlockExtention, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	result, err := g.Client.GetNowBlock2(ctx, new(api.EmptyMessage))

	if err != nil {
		return nil, fmt.Errorf("Get block now: %v", err)
	}

	return result, nil
}

// GetBlockByNum block from number
func (g *GrpcClient) GetBlockByNum(num int64) (*api.BlockExtention, error) {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	ctx, cancel := g.getContext()
	defer cancel()

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	result, err := g.Client.GetBlockByNum2(ctx, numMessage, maxSizeOption)

	if err != nil {
		return nil, fmt.Errorf("Get block by num: %v", err)

	}
	return result, nil
}

// GetBlockInfoByNum block from number
func (g *GrpcClient) GetBlockInfoByNum(num int64) (*api.TransactionInfoList, error) {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	ctx, cancel := g.getContext()
	defer cancel()

	result, err := g.Client.GetTransactionInfoByBlockNum(ctx, numMessage, maxSizeOption)

	if err != nil {
		return nil, fmt.Errorf("Get block info by num: %v", err)

	}
	return result, nil
}

// GetBlockByID block from hash
func (g *GrpcClient) GetBlockByID(id string) (*core.Block, error) {
	blockID := new(api.BytesMessage)
	var err error

	blockID.Value, err = common.FromHex(id)
	if err != nil {
		return nil, fmt.Errorf("get block by id: %v", err)
	}

	ctx, cancel := g.getContext()
	defer cancel()

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	return g.Client.GetBlockById(ctx, blockID, maxSizeOption)
}

// GetBlockByLimitNext return list of block start/end
func (g *GrpcClient) GetBlockByLimitNext(start, end int64) (*api.BlockListExtention, error) {
	blockLimit := new(api.BlockLimit)
	blockLimit.StartNum = start
	blockLimit.EndNum = end

	ctx, cancel := g.getContext()
	defer cancel()

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	return g.Client.GetBlockByLimitNext2(ctx, blockLimit, maxSizeOption)
}

// GetBlockByLatestNum return block list till num
func (g *GrpcClient) GetBlockByLatestNum(num int64) (*api.BlockListExtention, error) {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	ctx, cancel := g.getContext()
	defer cancel()

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	return g.Client.GetBlockByLatestNum2(ctx, numMessage, maxSizeOption)
}
