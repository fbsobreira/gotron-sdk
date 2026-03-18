package client_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestGetTransactionFromPending(t *testing.T) {
	txID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	txIDBytes, _ := hex.DecodeString(txID)

	mock := &mockWalletServer{
		GetTransactionFromPendingFunc: func(_ context.Context, in *api.BytesMessage) (*core.Transaction, error) {
			assert.Equal(t, txIDBytes, in.Value)
			return &core.Transaction{
				RawData: &core.TransactionRaw{RefBlockBytes: txIDBytes[:2]},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.GetTransactionFromPending(txID)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestGetTransactionFromPending_NotFound(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionFromPendingFunc: func(_ context.Context, _ *api.BytesMessage) (*core.Transaction, error) {
			return &core.Transaction{}, nil // zero-size
		},
	}

	c := newMockClient(t, mock)
	_, err := c.GetTransactionFromPending("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetTransactionFromPending_InvalidHex(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetTransactionFromPending("not-hex")
	require.Error(t, err)
}

func TestGetTransactionFromPending_RPCError(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionFromPendingFunc: func(_ context.Context, _ *api.BytesMessage) (*core.Transaction, error) {
			return nil, fmt.Errorf("rpc unavailable")
		},
	}

	c := newMockClient(t, mock)
	_, err := c.GetTransactionFromPending("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rpc unavailable")
}

func TestGetTransactionListFromPending(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionListFromPendingFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.TransactionIdList, error) {
			return &api.TransactionIdList{
				TxId: []string{
					"abc123",
					"def456",
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	list, err := c.GetTransactionListFromPending()
	require.NoError(t, err)
	require.Len(t, list.TxId, 2)
	assert.Equal(t, "abc123", list.TxId[0])
}

func TestGetTransactionListFromPending_Empty(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionListFromPendingFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.TransactionIdList, error) {
			return &api.TransactionIdList{}, nil
		},
	}

	c := newMockClient(t, mock)
	list, err := c.GetTransactionListFromPending()
	require.NoError(t, err)
	assert.Empty(t, list.TxId)
}

func TestGetTransactionListFromPending_RPCError(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionListFromPendingFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.TransactionIdList, error) {
			return nil, fmt.Errorf("network timeout")
		},
	}

	c := newMockClient(t, mock)
	_, err := c.GetTransactionListFromPending()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
}

func TestGetPendingSize(t *testing.T) {
	mock := &mockWalletServer{
		GetPendingSizeFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			return &api.NumberMessage{Num: 42}, nil
		},
	}

	c := newMockClient(t, mock)
	result, err := c.GetPendingSize()
	require.NoError(t, err)
	assert.Equal(t, int64(42), result.Num)
}

func TestGetPendingSize_Zero(t *testing.T) {
	mock := &mockWalletServer{
		GetPendingSizeFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			return &api.NumberMessage{Num: 0}, nil
		},
	}

	c := newMockClient(t, mock)
	result, err := c.GetPendingSize()
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Num)
}

func TestGetPendingSize_RPCError(t *testing.T) {
	mock := &mockWalletServer{
		GetPendingSizeFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	c := newMockClient(t, mock)
	_, err := c.GetPendingSize()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestIsTransactionPending_Found(t *testing.T) {
	txID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	mock := &mockWalletServer{
		GetTransactionFromPendingFunc: func(_ context.Context, _ *api.BytesMessage) (*core.Transaction, error) {
			return &core.Transaction{
				RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	pending, err := c.IsTransactionPending(txID)
	require.NoError(t, err)
	assert.True(t, pending)
}

func TestIsTransactionPending_NotFound(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionFromPendingFunc: func(_ context.Context, _ *api.BytesMessage) (*core.Transaction, error) {
			return &core.Transaction{}, nil // zero-size → triggers not-found error
		},
	}

	c := newMockClient(t, mock)
	pending, err := c.IsTransactionPending("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	require.NoError(t, err)
	assert.False(t, pending)
}

// makePendingTx builds a minimal Transaction with a TransferContract for the given owner address.
func makePendingTx(t *testing.T, ownerAddr string) *core.Transaction {
	t.Helper()
	addrBytes, err := common.DecodeCheck(ownerAddr)
	require.NoError(t, err)

	tc := &core.TransferContract{OwnerAddress: addrBytes, Amount: 1_000_000}
	param, err := anypb.New(tc)
	require.NoError(t, err)

	return &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type:      core.Transaction_Contract_TransferContract,
					Parameter: param,
				},
			},
		},
	}
}

func TestGetPendingTransactionsByAddress(t *testing.T) {
	ownerAddr := "TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b"
	otherAddr := "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"

	txOwner := makePendingTx(t, ownerAddr)
	txOther := makePendingTx(t, otherAddr)

	txMap := map[string]*core.Transaction{
		"aa": txOwner,
		"bb": txOther,
		"cc": txOwner,
	}

	mock := &mockWalletServer{
		GetTransactionListFromPendingFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.TransactionIdList, error) {
			return &api.TransactionIdList{TxId: []string{"aa", "bb", "cc"}}, nil
		},
		GetTransactionFromPendingFunc: func(_ context.Context, in *api.BytesMessage) (*core.Transaction, error) {
			id := hex.EncodeToString(in.Value)
			if tx, ok := txMap[id]; ok {
				return tx, nil
			}
			return &core.Transaction{}, nil
		},
	}

	c := newMockClient(t, mock)
	txs, err := c.GetPendingTransactionsByAddress(ownerAddr)
	require.NoError(t, err)
	assert.Len(t, txs, 2)
}

func TestGetPendingTransactionsByAddress_Empty(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionListFromPendingFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.TransactionIdList, error) {
			return &api.TransactionIdList{}, nil
		},
	}

	c := newMockClient(t, mock)
	txs, err := c.GetPendingTransactionsByAddress("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b")
	require.NoError(t, err)
	assert.Empty(t, txs)
}

func TestGetPendingTransactionsByAddress_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetPendingTransactionsByAddress("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")
}

func TestGetPendingTransactionsByAddress_ListError(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionListFromPendingFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.TransactionIdList, error) {
			return nil, fmt.Errorf("rpc error")
		},
	}

	c := newMockClient(t, mock)
	_, err := c.GetPendingTransactionsByAddress("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b")
	require.Error(t, err)
}
