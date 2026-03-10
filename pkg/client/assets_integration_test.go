//go:build integration

package client_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetAssetIssueByAccount(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetAssetIssueByAccount(nileTestAccountAddress)
	require.NoError(t, err)
	require.NotNil(t, result)
	// The result may be empty if the account has no TRC10 assets.
}

func TestIntegration_GetAssetIssueByName(t *testing.T) {
	c := newIntegrationClient(t)

	// First get a real TRC10 name from the asset list.
	list, err := c.GetAssetIssueList(0, 1)
	require.NoError(t, err)
	require.NotEmpty(t, list.GetAssetIssue(), "Nile testnet should have at least one TRC10 asset")

	assetName := string(list.GetAssetIssue()[0].GetName())
	require.NotEmpty(t, assetName)

	result, err := c.GetAssetIssueByName(assetName)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, assetName, string(result.GetName()), "returned asset name should match query")
	assert.Greater(t, result.GetTotalSupply(), int64(0))
}

func TestIntegration_GetAssetIssueByID(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetAssetIssueByID("1000001")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.GetName(), "asset name should be populated")
	assert.Greater(t, result.GetTotalSupply(), int64(0), "total supply should be positive")
}

func TestIntegration_GetAssetIssueList(t *testing.T) {
	c := newIntegrationClient(t)

	t.Run("all", func(t *testing.T) {
		result, err := c.GetAssetIssueList(-1)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.GetAssetIssue(), "should have at least one TRC10 token on testnet")
	})

	t.Run("paginated", func(t *testing.T) {
		result, err := c.GetAssetIssueList(0, 5)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.LessOrEqual(t, len(result.GetAssetIssue()), 5, "paginated result should respect limit")
	})
}

func TestIntegration_AssetIssue(t *testing.T) {
	c := newIntegrationClient(t)

	now := time.Now()
	start := now.Add(24 * time.Hour).UnixMilli()
	end := now.Add(48 * time.Hour).UnixMilli()

	// Account doesn't have enough balance — expected contract validation error.
	_, err := c.AssetIssue(
		nileTestAccountAddress,
		"IntegrationTestToken",
		"Integration test token",
		"ITT",
		"https://example.com",
		6,
		1000000,
		start, end,
		10000, 10000,
		1, 1, 0,
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_UpdateAssetIssue(t *testing.T) {
	c := newIntegrationClient(t)

	// Account has not issued any asset — expected contract validation error.
	_, err := c.UpdateAssetIssue(
		nileTestAccountAddress,
		"updated description",
		"https://example.com/updated",
		10000, 10000,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_TransferAsset(t *testing.T) {
	c := newIntegrationClient(t)

	// Account has no TRC10 balance — expected contract validation error.
	_, err := c.TransferAsset(
		nileTestAccountAddress, nileTestAddress2,
		"1000001", 1,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_ParticipateAssetIssue(t *testing.T) {
	c := newIntegrationClient(t)

	// Target account didn't issue the token — expected contract validation error.
	_, err := c.ParticipateAssetIssue(
		nileTestAccountAddress, nileTestAddress2,
		"1000001", 1,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_UnfreezeAsset(t *testing.T) {
	c := newIntegrationClient(t)

	// No frozen supply — expected contract validation error.
	_, err := c.UnfreezeAsset(nileTestAccountAddress)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}
