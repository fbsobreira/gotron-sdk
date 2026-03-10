//go:build integration

package client_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

func TestIntegration_FreezeBalance(t *testing.T) {
	c := newIntegrationClient(t)

	// Freeze v2 is enabled on Nile — v1 freeze should be rejected.
	_, err := c.FreezeBalance(nileTestAccountAddress, "", core.ResourceCode_BANDWIDTH, 1000000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "freeze v2 is open, old freeze is closed")
}

func TestIntegration_UnfreezeBalance(t *testing.T) {
	c := newIntegrationClient(t)

	// No v1 frozen balance — expected error.
	_, err := c.UnfreezeBalance(nileTestAccountAddress, "", core.ResourceCode_BANDWIDTH)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_WithdrawExpireUnfreeze(t *testing.T) {
	c := newIntegrationClient(t)

	// May succeed (returning empty tx) or fail depending on account state.
	tx, err := c.WithdrawExpireUnfreeze(nileTestAccountAddress, time.Now().UnixMilli())
	if err == nil {
		require.NotNil(t, tx)
	}
	// No expired unfreeze is acceptable.
}

func TestIntegration_FreezeBalanceV2(t *testing.T) {
	c := newIntegrationClient(t)

	// Should produce a valid unsigned transaction (freeze v2 is active on Nile).
	tx, err := c.FreezeBalanceV2(nileTestAccountAddress, core.ResourceCode_BANDWIDTH, 1000000)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "FreezeBalanceV2 should produce a transaction ID")
}

func TestIntegration_UnfreezeBalanceV2(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.UnfreezeBalanceV2(nileTestAccountAddress, core.ResourceCode_BANDWIDTH, 1000000)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "UnfreezeBalanceV2 should produce a transaction ID")
}

func TestIntegration_GetAvailableUnfreezeCount(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetAvailableUnfreezeCount(nileTestAccountAddress)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, result.GetCount(), int64(0), "unfreeze count should be non-negative")
}

func TestIntegration_GetCanWithdrawUnfreezeAmount(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetCanWithdrawUnfreezeAmount(nileTestAccountAddress, time.Now().UnixMilli())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, result.GetAmount(), int64(0), "withdrawable amount should be non-negative")
}
