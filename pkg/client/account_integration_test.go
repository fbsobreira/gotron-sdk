//go:build integration

package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_GetAccount(t *testing.T) {
	c := newIntegrationClient(t)

	acc, err := c.GetAccount(nileTestAccountAddress)
	require.NoError(t, err)
	require.NotNil(t, acc)
	assert.NotEmpty(t, acc.Address, "account address should be populated")
	assert.GreaterOrEqual(t, acc.Balance, int64(0), "balance should be non-negative")
}

func TestIntegration_GetAccount_WitnessAddress(t *testing.T) {
	c := newIntegrationClient(t)

	acc, err := c.GetAccount(nileTestWitnessAddress)
	require.NoError(t, err)
	require.NotNil(t, acc)
	assert.NotEmpty(t, acc.Address)
}

func TestIntegration_GetAccountDetailed(t *testing.T) {
	c := newIntegrationClient(t)

	acc, err := c.GetAccountDetailed(nileTestAccountAddress)
	require.NoError(t, err)
	require.NotNil(t, acc)
	assert.Equal(t, nileTestAccountAddress, acc.Address, "returned address should match request")
	assert.GreaterOrEqual(t, acc.Balance, int64(0))
	assert.GreaterOrEqual(t, acc.BWTotal, int64(0))
	assert.GreaterOrEqual(t, acc.EnergyTotal, int64(0))
	assert.GreaterOrEqual(t, acc.TronPower, int64(0))
	assert.GreaterOrEqual(t, acc.MaxCanDelegateBandwidth, int64(0))
	assert.GreaterOrEqual(t, acc.MaxCanDelegateEnergy, int64(0))
}

func TestIntegration_GetAccountDetailed_WitnessAccount(t *testing.T) {
	c := newIntegrationClient(t)

	acc, err := c.GetAccountDetailed(nileTestWitnessAddress)
	require.NoError(t, err)
	require.NotNil(t, acc)
	assert.Equal(t, nileTestWitnessAddress, acc.Address)
	assert.GreaterOrEqual(t, acc.Rewards, int64(0), "witness should have non-negative rewards")
	assert.GreaterOrEqual(t, acc.Allowance, int64(0))
}

func TestIntegration_GetAccountResource(t *testing.T) {
	c := newIntegrationClient(t)

	res, err := c.GetAccountResource(nileTestAccountAddress)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.GreaterOrEqual(t, res.FreeNetLimit, int64(0), "free net limit should be non-negative")
	assert.GreaterOrEqual(t, res.EnergyLimit, int64(0))
}

func TestIntegration_GetAccountNet(t *testing.T) {
	c := newIntegrationClient(t)

	net, err := c.GetAccountNet(nileTestAccountAddress)
	require.NoError(t, err)
	require.NotNil(t, net)
	assert.GreaterOrEqual(t, net.FreeNetLimit, int64(0))
}

func TestIntegration_GetRewardsInfo(t *testing.T) {
	c := newIntegrationClient(t)

	rewards, err := c.GetRewardsInfo(nileTestAccountAddress)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, rewards, int64(0), "rewards should be non-negative")
}

func TestIntegration_CreateAccount(t *testing.T) {
	c := newIntegrationClient(t)

	// Existing account: the node returns a contract validation error.
	_, err := c.CreateAccount(nileTestAccountAddress, nileTestAddress2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_UpdateAccount(t *testing.T) {
	c := newIntegrationClient(t)

	// Should produce a valid unsigned transaction.
	tx, err := c.UpdateAccount(nileTestAccountAddress, "integration-test")
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "transaction should have an ID")
}

func TestIntegration_WithdrawBalance(t *testing.T) {
	c := newIntegrationClient(t)

	// Non-witness account: the node returns a contract validation error.
	_, err := c.WithdrawBalance(nileTestAccountAddress)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_UpdateAccountPermission(t *testing.T) {
	c := newIntegrationClient(t)

	owner := map[string]interface{}{
		"threshold": int64(1),
		"keys": map[string]int64{
			nileTestAccountAddress: 1,
		},
	}
	actives := []map[string]interface{}{
		{
			"name":      "active",
			"threshold": int64(1),
			"operations": map[string]bool{
				"TransferContract": true,
			},
			"keys": map[string]int64{
				nileTestAccountAddress: 1,
			},
		},
	}

	tx, err := c.UpdateAccountPermission(nileTestAccountAddress, owner, nil, actives)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "permission update should produce a transaction")
}
