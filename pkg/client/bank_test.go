package client_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreezeBalanceV2_ZeroAmount(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.FreezeBalanceV2(accountAddress, core.ResourceCode_BANDWIDTH, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "freeze balance must be positive")
}

func TestFreezeBalanceV2_NegativeAmount(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.FreezeBalanceV2(accountAddress, core.ResourceCode_BANDWIDTH, -100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "freeze balance must be positive")
}

func TestUnfreezeBalanceV2_ZeroAmount(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.UnfreezeBalanceV2(accountAddress, core.ResourceCode_ENERGY, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unfreeze balance must be positive")
}

func TestUnfreezeBalanceV2_NegativeAmount(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.UnfreezeBalanceV2(accountAddress, core.ResourceCode_ENERGY, -500)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unfreeze balance must be positive")
}
