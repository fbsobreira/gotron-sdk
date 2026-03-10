//go:build integration

package client_test

import (
	"fmt"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetNodeInfo(t *testing.T) {
	c := newIntegrationClient(t)

	info, err := c.GetNodeInfo()
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.NotEmpty(t, info.GetConfigNodeInfo().GetCodeVersion(), "node should report a code version")
	assert.Greater(t, info.GetBeginSyncNum(), int64(0), "sync number should be positive")
}

func TestIntegration_ListNodes(t *testing.T) {
	c := newIntegrationClient(t)

	nodes, err := c.ListNodes()
	require.NoError(t, err)
	require.NotNil(t, nodes)
	// ListNodes may return an empty list on some nodes, so just check no error.
}

func TestIntegration_ListWitnesses(t *testing.T) {
	c := newIntegrationClient(t)

	witnesses, err := c.ListWitnesses()
	require.NoError(t, err)
	require.NotNil(t, witnesses)
	require.NotEmpty(t, witnesses.GetWitnesses(), "network must have at least one witness")

	w := witnesses.GetWitnesses()[0]
	assert.NotEmpty(t, w.GetAddress(), "witness should have an address")
	assert.NotEmpty(t, w.GetUrl(), "witness should have a URL")
	assert.Greater(t, w.GetTotalProduced(), int64(0), "active witness should have produced blocks")
}

func TestIntegration_GetNextMaintenanceTime(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetNextMaintenanceTime()
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, result.GetNum(), int64(0), "maintenance time should be a positive timestamp")
}

func TestIntegration_GetNowBlock(t *testing.T) {
	c := newIntegrationClient(t)

	block, err := c.GetNowBlock()
	require.NoError(t, err)
	require.NotNil(t, block)
	require.NotNil(t, block.GetBlockHeader())

	header := block.GetBlockHeader().GetRawData()
	assert.Greater(t, header.GetNumber(), int64(0), "block number should be positive")
	assert.Greater(t, header.GetTimestamp(), int64(0), "block timestamp should be positive")
	assert.NotEmpty(t, header.GetWitnessAddress(), "block should have a witness address")
}

func TestIntegration_GetBlockByNum(t *testing.T) {
	c := newIntegrationClient(t)

	block, err := c.GetBlockByNum(1)
	require.NoError(t, err)
	require.NotNil(t, block)
	require.NotNil(t, block.GetBlockHeader())

	header := block.GetBlockHeader().GetRawData()
	assert.Equal(t, int64(1), header.GetNumber(), "should return block number 1")
	assert.Greater(t, header.GetTimestamp(), int64(0))
}

func TestIntegration_GetBlockInfoByNum(t *testing.T) {
	c := newIntegrationClient(t)

	info, err := c.GetBlockInfoByNum(1)
	require.NoError(t, err)
	require.NotNil(t, info)
	// Block 1 may or may not have transactions, just verify no error.
}

func TestIntegration_GetTransactionByID(t *testing.T) {
	c := newIntegrationClient(t)

	txIDHex := findConfirmedTransactionID(t, c)

	tx, err := c.GetTransactionByID(txIDHex)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotNil(t, tx.GetRawData(), "transaction should have raw data")
	assert.NotEmpty(t, tx.GetRawData().GetContract(), "transaction should have at least one contract")
}

func TestIntegration_GetTransactionInfoByID(t *testing.T) {
	c := newIntegrationClient(t)

	txIDHex := findConfirmedTransactionID(t, c)

	info, err := c.GetTransactionInfoByID(txIDHex)
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, txIDHex, fmt.Sprintf("%x", info.GetId()), "transaction info ID should match requested")
}

// findConfirmedTransactionID looks back from the tip to find a confirmed block
// with at least one transaction, avoiding flakiness from querying the tip block
// whose receipts may not be indexed yet.
func findConfirmedTransactionID(t *testing.T, c *client.GrpcClient) string {
	t.Helper()

	tip, err := c.GetNowBlock()
	require.NoError(t, err)

	tipNum := tip.GetBlockHeader().GetRawData().GetNumber()

	// Scan backwards from 10 blocks behind the tip.
	for offset := int64(10); offset <= 50; offset++ {
		block, err := c.GetBlockByNum(tipNum - offset)
		require.NoError(t, err)

		if len(block.GetTransactions()) > 0 {
			return fmt.Sprintf("%x", block.GetTransactions()[0].GetTxid())
		}
	}

	t.Skip("no transactions found in recent confirmed blocks")
	return ""
}

func TestIntegration_TotalTransaction(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.TotalTransaction()
	require.NoError(t, err)
	require.NotNil(t, result)
	// TotalTransaction is deprecated on some nodes, may return 0.
}

func TestIntegration_GetDelegatedResources(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetDelegatedResources(nileTestAccountAddress)
	require.NoError(t, err)
	require.NotNil(t, result)
	// May be empty if account has no delegations.
}

func TestIntegration_GetDelegatedResourcesV2(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetDelegatedResourcesV2(nileTestAccountAddress)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestIntegration_GetEnergyPrices(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetEnergyPrices()
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.GetPrices(), "energy prices should not be empty")
}

func TestIntegration_GetBandwidthPrices(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetBandwidthPrices()
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.GetPrices(), "bandwidth prices should not be empty")
}

func TestIntegration_GetMemoFee(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetMemoFee()
	require.NoError(t, err)
	require.NotNil(t, result)
	// Memo fee may be empty on some networks, but the call should succeed.
}

func TestIntegration_GetTransactionSignWeight(t *testing.T) {
	c := newIntegrationClient(t)

	// Create a real transaction to check its sign weight.
	tx, err := c.Transfer(nileTestAccountAddress, nileTestAddress2, 1)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, tx.GetTransaction())

	weight, err := c.GetTransactionSignWeight(tx.GetTransaction())
	require.NoError(t, err)
	require.NotNil(t, weight)
	assert.NotNil(t, weight.GetPermission(), "sign weight should include permission info")
	assert.Greater(t, weight.GetPermission().GetThreshold(), int64(0), "permission threshold should be positive")
}
