//go:build integration

package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

func TestIntegration_GetReceivedDelegatedResourcesV2(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetReceivedDelegatedResourcesV2(nileTestAccountAddress)
	require.NoError(t, err)
	require.NotNil(t, result)
	// May be empty if no one has delegated to this account.
}

func TestIntegration_GetCanDelegatedMaxSize(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.GetCanDelegatedMaxSize(nileTestAccountAddress, int32(core.ResourceCode_BANDWIDTH))
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.GreaterOrEqual(t, result.GetMaxSize(), int64(0), "max delegatable size should be non-negative")
}

func TestIntegration_DelegateResource(t *testing.T) {
	c := newIntegrationClient(t)

	// Creates a delegate resource transaction — validates the SDK builds the correct contract.
	tx, err := c.DelegateResource(
		nileTestAccountAddress, nileTestAddress2,
		core.ResourceCode_BANDWIDTH, 1000000,
		false, 0,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "DelegateResource should produce a transaction ID")
	assert.NotNil(t, tx.GetTransaction().GetRawData(), "transaction should have raw data")
}

func TestIntegration_UnDelegateResource(t *testing.T) {
	c := newIntegrationClient(t)

	// No delegated resources — the node returns an empty transaction (no error, but no txid).
	tx, err := c.UnDelegateResource(
		nileTestAccountAddress, nileTestAddress2,
		core.ResourceCode_BANDWIDTH, 1000000,
	)
	require.NoError(t, err)
	require.NotNil(t, tx, "UnDelegateResource should return a non-nil response")
}
