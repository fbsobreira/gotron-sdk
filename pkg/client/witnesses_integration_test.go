//go:build integration

package client_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_CreateWitness(t *testing.T) {
	c := newIntegrationClient(t)

	// Account doesn't have enough TRX to become a witness.
	_, err := c.CreateWitness(nileTestAccountAddress, "https://example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_UpdateWitness(t *testing.T) {
	c := newIntegrationClient(t)

	// Account is not a witness.
	_, err := c.UpdateWitness(nileTestAccountAddress, "https://example.com/updated")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_UpdateBrokerage(t *testing.T) {
	c := newIntegrationClient(t)

	// Account is not a witness.
	_, err := c.UpdateBrokerage(nileTestAccountAddress, 20)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_GetWitnessBrokerage(t *testing.T) {
	c := newIntegrationClient(t)

	witnesses, err := c.ListWitnesses()
	require.NoError(t, err)
	require.NotEmpty(t, witnesses.GetWitnesses(), "need at least one witness")

	witnessAddr := addressBytesToBase58(t, witnesses.GetWitnesses()[0].GetAddress())

	brokerage, err := c.GetWitnessBrokerage(witnessAddr)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, brokerage, float64(0), "brokerage should be >= 0")
	assert.LessOrEqual(t, brokerage, float64(100), "brokerage should be <= 100")
}

func TestIntegration_VoteWitnessAccount(t *testing.T) {
	c := newIntegrationClient(t)

	witnesses, err := c.ListWitnesses()
	require.NoError(t, err)
	require.NotEmpty(t, witnesses.GetWitnesses(), "need at least one witness")

	witnessAddr := addressBytesToBase58(t, witnesses.GetWitnesses()[0].GetAddress())

	// Should produce a valid unsigned transaction.
	tx, err := c.VoteWitnessAccount(nileTestAccountAddress, map[string]int64{
		witnessAddr: 1,
	})
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "VoteWitnessAccount should produce a transaction ID")
}

// addressBytesToBase58 converts raw address bytes to a base58 string.
func addressBytesToBase58(t *testing.T, addr []byte) string {
	t.Helper()
	require.NotEmpty(t, addr, "address bytes must not be empty")
	return address.Address(addr).String()
}
