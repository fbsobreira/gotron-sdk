//go:build integration

package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Transfer(t *testing.T) {
	c := newIntegrationClient(t)

	// Creates a TRX transfer transaction (1 sun) — not broadcast.
	tx, err := c.Transfer(nileTestAccountAddress, nileTestAddress2, 1)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "Transfer should produce a transaction ID")
	assert.NotNil(t, tx.GetTransaction().GetRawData(), "transaction should have raw data")
}
