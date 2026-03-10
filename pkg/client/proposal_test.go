package client_test

import (
	"context"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProposalsList(t *testing.T) {
	mock := &mockWalletServer{
		ListProposalsFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.ProposalList, error) {
			return &api.ProposalList{
				Proposals: []*core.Proposal{
					{ProposalId: 1},
					{ProposalId: 2},
					{ProposalId: 3},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	list, err := c.ProposalsList()
	require.NoError(t, err)
	assert.Len(t, list.Proposals, 3)
}

func TestProposalCreate(t *testing.T) {
	params := map[int64]int64{0: 100000, 1: 2}

	mock := &mockWalletServer{
		ProposalCreateFunc: func(_ context.Context, in *core.ProposalCreateContract) (*api.TransactionExtention, error) {
			assert.Equal(t, params, in.Parameters)
			assert.NotEmpty(t, in.OwnerAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.ProposalCreate(accountAddress, params)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestProposalCreate_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.ProposalCreate("invalid-address", map[int64]int64{0: 1})
	require.Error(t, err)
}

func TestProposalCreate_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		ProposalCreateFunc: func(_ context.Context, _ *core.ProposalCreateContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}

	c := newMockClient(t, mock)
	_, err := c.ProposalCreate(accountAddress, map[int64]int64{0: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestProposalApprove(t *testing.T) {
	mock := &mockWalletServer{
		ProposalApproveFunc: func(_ context.Context, in *core.ProposalApproveContract) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(5), in.ProposalId)
			assert.True(t, in.IsAddApproval)
			assert.NotEmpty(t, in.OwnerAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.ProposalApprove(accountAddress, 5, true)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestProposalApprove_Disapprove(t *testing.T) {
	mock := &mockWalletServer{
		ProposalApproveFunc: func(_ context.Context, in *core.ProposalApproveContract) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(7), in.ProposalId)
			assert.False(t, in.IsAddApproval)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.ProposalApprove(accountAddress, 7, false)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestProposalApprove_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.ProposalApprove("invalid-address", 1, true)
	require.Error(t, err)
}

func TestProposalWithdraw(t *testing.T) {
	mock := &mockWalletServer{
		ProposalDeleteFunc: func(_ context.Context, in *core.ProposalDeleteContract) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(10), in.ProposalId)
			assert.NotEmpty(t, in.OwnerAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.ProposalWithdraw(accountAddress, 10)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestProposalWithdraw_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.ProposalWithdraw("invalid-address", 1)
	require.Error(t, err)
}
