package client_test

import (
	"context"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreezeBalance(t *testing.T) {
	mock := &mockWalletServer{
		FreezeBalance2Func: func(_ context.Context, in *core.FreezeBalanceContract) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(1_000_000), in.FrozenBalance)
			assert.Equal(t, int64(3), in.FrozenDuration)
			assert.Equal(t, core.ResourceCode_BANDWIDTH, in.Resource)
			assert.NotEmpty(t, in.OwnerAddress)
			assert.Empty(t, in.ReceiverAddress) // no delegate
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.FreezeBalance(accountAddress, "", core.ResourceCode_BANDWIDTH, 1_000_000)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestFreezeBalance_WithDelegate(t *testing.T) {
	mock := &mockWalletServer{
		FreezeBalance2Func: func(_ context.Context, in *core.FreezeBalanceContract) (*api.TransactionExtention, error) {
			assert.NotEmpty(t, in.ReceiverAddress)
			assert.Equal(t, core.ResourceCode_ENERGY, in.Resource)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.FreezeBalance(accountAddress, accountAddressWitness, core.ResourceCode_ENERGY, 2_000_000)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestFreezeBalance_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.FreezeBalance("invalid", "", core.ResourceCode_BANDWIDTH, 1000)
	require.Error(t, err)
}

func TestFreezeBalance_InvalidDelegate(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.FreezeBalance(accountAddress, "invalid-delegate", core.ResourceCode_BANDWIDTH, 1000)
	require.Error(t, err)
}

func TestFreezeBalance_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		FreezeBalance2Func: func(_ context.Context, _ *core.FreezeBalanceContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)
	_, err := c.FreezeBalance(accountAddress, "", core.ResourceCode_BANDWIDTH, 1000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestUnfreezeBalance(t *testing.T) {
	mock := &mockWalletServer{
		UnfreezeBalance2Func: func(_ context.Context, in *core.UnfreezeBalanceContract) (*api.TransactionExtention, error) {
			assert.Equal(t, core.ResourceCode_ENERGY, in.Resource)
			assert.NotEmpty(t, in.OwnerAddress)
			assert.Empty(t, in.ReceiverAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.UnfreezeBalance(accountAddress, "", core.ResourceCode_ENERGY)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestUnfreezeBalance_WithDelegate(t *testing.T) {
	mock := &mockWalletServer{
		UnfreezeBalance2Func: func(_ context.Context, in *core.UnfreezeBalanceContract) (*api.TransactionExtention, error) {
			assert.NotEmpty(t, in.ReceiverAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.UnfreezeBalance(accountAddress, accountAddressWitness, core.ResourceCode_BANDWIDTH)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestUnfreezeBalance_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.UnfreezeBalance("invalid", "", core.ResourceCode_BANDWIDTH)
	require.Error(t, err)
}

func TestWithdrawExpireUnfreeze(t *testing.T) {
	mock := &mockWalletServer{
		WithdrawExpireUnfreezeFunc: func(_ context.Context, in *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
			assert.NotEmpty(t, in.OwnerAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.WithdrawExpireUnfreeze(accountAddress, 1700000000000)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestWithdrawExpireUnfreeze_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.WithdrawExpireUnfreeze("invalid", 1700000000000)
	require.Error(t, err)
}

func TestWithdrawExpireUnfreeze_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		WithdrawExpireUnfreezeFunc: func(_ context.Context, _ *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)
	_, err := c.WithdrawExpireUnfreeze(accountAddress, 1700000000000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}
