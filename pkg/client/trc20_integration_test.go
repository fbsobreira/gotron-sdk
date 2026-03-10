//go:build integration

package client_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_TRC20GetDecimals(t *testing.T) {
	c := newIntegrationClient(t)

	decimals, err := c.TRC20GetDecimals(nileUSDTContract)
	require.NoError(t, err)
	require.NotNil(t, decimals)
	assert.Equal(t, int64(6), decimals.Int64(), "USDT should have 6 decimals")
}

func TestIntegration_TRC20GetName(t *testing.T) {
	c := newIntegrationClient(t)

	name, err := c.TRC20GetName(nileUSDTContract)
	require.NoError(t, err)
	assert.Equal(t, "Tether USD", name, "USDT contract name should be Tether USD")
}

func TestIntegration_TRC20GetSymbol(t *testing.T) {
	c := newIntegrationClient(t)

	symbol, err := c.TRC20GetSymbol(nileUSDTContract)
	require.NoError(t, err)
	assert.Equal(t, "USDT", symbol, "USDT contract symbol should be USDT")
}

func TestIntegration_TRC20ContractBalance(t *testing.T) {
	c := newIntegrationClient(t)

	balance, err := c.TRC20ContractBalance(nileTestAccountAddress, nileUSDTContract)
	require.NoError(t, err)
	require.NotNil(t, balance)
	assert.True(t, balance.Cmp(big.NewInt(0)) >= 0, "balance should be non-negative")
}

func TestIntegration_TRC20Send(t *testing.T) {
	c := newIntegrationClient(t)

	// Creates a transaction — validates the SDK builds the correct call data.
	tx, err := c.TRC20Send(
		nileTestAccountAddress,
		nileTestAddress2,
		nileUSDTContract,
		big.NewInt(1),
		10000000,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "TRC20Send should produce a transaction with ID")
	assert.NotNil(t, tx.GetTransaction(), "TRC20Send should produce a transaction body")
}

func TestIntegration_TRC20TransferFrom(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.TRC20TransferFrom(
		nileTestAccountAddress,
		nileTestAddress2,
		nileTestAddress,
		nileUSDTContract,
		big.NewInt(1),
		10000000,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "TRC20TransferFrom should produce a transaction with ID")
	assert.NotNil(t, tx.GetTransaction(), "TRC20TransferFrom should produce a transaction body")
}

func TestIntegration_TRC20Approve(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.TRC20Approve(
		nileTestAccountAddress,
		nileTestAddress2,
		nileUSDTContract,
		big.NewInt(1000),
		10000000,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "TRC20Approve should produce a transaction with ID")
	assert.NotNil(t, tx.GetTransaction(), "TRC20Approve should produce a transaction body")
}
