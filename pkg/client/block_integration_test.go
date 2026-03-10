//go:build integration

package client_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetBlockByID(t *testing.T) {
	c := newIntegrationClient(t)

	// First get a known block to obtain its ID (hash).
	block, err := c.GetBlockByNum(1)
	require.NoError(t, err)
	require.NotNil(t, block)

	blockID := fmt.Sprintf("%x", block.GetBlockid())
	require.NotEmpty(t, blockID, "block 1 should have a block ID")

	result, err := c.GetBlockByID(blockID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.GetBlockHeader())
	assert.Equal(t, int64(1), result.GetBlockHeader().GetRawData().GetNumber(), "should return block number 1")
}

func TestIntegration_GetBlockByLatestNum(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetBlockByLatestNum(3)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.GetBlock(), "should return at least one block")
	assert.LessOrEqual(t, len(result.GetBlock()), 3, "should return at most 3 blocks")

	// Verify blocks are ordered with ascending block numbers.
	blocks := result.GetBlock()
	for i := 1; i < len(blocks); i++ {
		prev := blocks[i-1].GetBlockHeader().GetRawData().GetNumber()
		curr := blocks[i].GetBlockHeader().GetRawData().GetNumber()
		assert.Greater(t, curr, prev, "blocks should be in ascending order")
	}
}

func TestIntegration_GetBlockByLimitNext(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetBlockByLimitNext(1, 4)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.GetBlock(), "should return blocks in range")

	// First block should be block 1.
	firstBlock := result.GetBlock()[0]
	assert.Equal(t, int64(1), firstBlock.GetBlockHeader().GetRawData().GetNumber(), "first block should be number 1")

	// Should return blocks 1, 2, 3 (end is exclusive in TRON).
	assert.LessOrEqual(t, len(result.GetBlock()), 3, "should return at most 3 blocks for range [1,4)")
}
