//go:build integration

package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ExchangeList(t *testing.T) {
	c := newIntegrationClient(t)

	t.Run("all", func(t *testing.T) {
		result, err := c.ExchangeList(-1)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEmpty(t, result.GetExchanges(), "testnet should have at least one exchange")

		ex := result.GetExchanges()[0]
		assert.Greater(t, ex.GetExchangeId(), int64(0), "exchange should have a positive ID")
		assert.NotEmpty(t, ex.GetFirstTokenId(), "exchange should have a first token")
		assert.NotEmpty(t, ex.GetSecondTokenId(), "exchange should have a second token")
	})

	t.Run("paginated", func(t *testing.T) {
		result, err := c.ExchangeList(0, 5)
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.LessOrEqual(t, len(result.GetExchanges()), 5, "paginated result should respect limit")
	})
}

func TestIntegration_ExchangeByID(t *testing.T) {
	c := newIntegrationClient(t)

	list, err := c.ExchangeList(-1)
	require.NoError(t, err)
	require.NotEmpty(t, list.GetExchanges(), "need at least one exchange")

	id := list.GetExchanges()[0].GetExchangeId()
	exchange, err := c.ExchangeByID(id)
	require.NoError(t, err)
	require.NotNil(t, exchange)
	assert.Equal(t, id, exchange.GetExchangeId(), "returned exchange ID should match requested")
	assert.NotEmpty(t, exchange.GetCreatorAddress(), "exchange should have a creator")
}

func TestIntegration_ExchangeInject(t *testing.T) {
	c := newIntegrationClient(t)

	// Not the exchange creator — expected error.
	_, err := c.ExchangeInject(nileTestAccountAddress, 1, "_", 100000000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_ExchangeWithdraw(t *testing.T) {
	c := newIntegrationClient(t)

	// Not the exchange creator — expected error.
	_, err := c.ExchangeWithdraw(nileTestAccountAddress, 1, "_", 100000000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_ExchangeTrade(t *testing.T) {
	c := newIntegrationClient(t)

	// Creates a trade transaction — validates the SDK builds the correct contract.
	tx, err := c.ExchangeTrade(nileTestAccountAddress, 1, "_", 100, 1)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "ExchangeTrade should produce a transaction ID")
}

func TestIntegration_ExchangeCreate(t *testing.T) {
	c := newIntegrationClient(t)

	// Account doesn't have enough balance for exchange creation fee.
	_, err := c.ExchangeCreate(
		nileTestAccountAddress,
		"_",       // TRX
		100000000, // 100 TRX
		"1000001",
		100,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}
