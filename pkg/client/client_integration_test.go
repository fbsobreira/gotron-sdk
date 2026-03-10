//go:build integration

package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_Reconnect(t *testing.T) {
	c := newIntegrationClient(t)

	// Verify the client works before reconnect.
	_, err := c.GetNodeInfo()
	require.NoError(t, err)

	// Reconnect to the same endpoint.
	err = c.Reconnect("")
	require.NoError(t, err)

	// Verify the client still works after reconnect.
	info, err := c.GetNodeInfo()
	require.NoError(t, err)
	assert.NotEmpty(t, info.GetConfigNodeInfo().GetCodeVersion(), "should work after reconnect")
}

func TestIntegration_Reconnect_DifferentEndpoint(t *testing.T) {
	c := newIntegrationClient(t)

	// Reconnect using an explicit endpoint URL.
	endpoint := getEndpoint()
	err := c.Reconnect(endpoint)
	require.NoError(t, err)

	info, err := c.GetNodeInfo()
	require.NoError(t, err)
	assert.NotEmpty(t, info.GetConfigNodeInfo().GetCodeVersion(), "should work after reconnect to explicit endpoint")
}
