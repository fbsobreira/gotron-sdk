package client

import (
	"context"
	"log"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"go.uber.org/zap"
)

// GetNowBlock return TIP block
func (g *GrpcClient) GetNowBlock() (*core.Block, error) {
	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	result, err := g.Client.GetNowBlock(ctx, new(api.EmptyMessage))

	if err != nil {
		zap.L().Error("Get block now", zap.Error(err))
		return nil, err
	}

	return result, nil
}

// GetBlockByNum block from number
func (g *GrpcClient) GetBlockByNum(num int64) *core.Block {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	result, err := g.Client.GetBlockByNum(ctx, numMessage)

	if err != nil {
		log.Fatalf("get block by num error: %v", err)
	}

	return result
}

// GetBlockByID block from hash
func (g *GrpcClient) GetBlockByID(id string) (*core.Block, error) {
	blockID := new(api.BytesMessage)
	var err error

	blockID.Value, err = common.Decode(id)

	if err != nil {
		log.Fatalf("get block by id error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	return g.Client.GetBlockById(ctx, blockID)
}

// GetBlockByLimitNext return list of block start/end
func (g *GrpcClient) GetBlockByLimitNext(start, end int64) (*api.BlockList, error) {
	blockLimit := new(api.BlockLimit)
	blockLimit.StartNum = start
	blockLimit.EndNum = end

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	return g.Client.GetBlockByLimitNext(ctx, blockLimit)

}

// GetBlockByLatestNum return block list till num
func (g *GrpcClient) GetBlockByLatestNum(num int64) (*api.BlockList, error) {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	return g.Client.GetBlockByLatestNum(ctx, numMessage)

}
