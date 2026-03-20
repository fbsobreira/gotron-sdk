package txcore

import (
	"context"
	"crypto/sha256"
	"errors"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// mockBroadcaster implements the Broadcaster interface for testing.
type mockBroadcaster struct {
	broadcastFn          func(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	getTransactionInfoFn func(ctx context.Context, id string) (*core.TransactionInfo, error)
}

func (m *mockBroadcaster) BroadcastCtx(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	if m.broadcastFn != nil {
		return m.broadcastFn(ctx, tx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockBroadcaster) GetTransactionInfoByIDCtx(ctx context.Context, id string) (*core.TransactionInfo, error) {
	if m.getTransactionInfoFn != nil {
		return m.getTransactionInfoFn(ctx, id)
	}
	return nil, errors.New("not implemented")
}

// mockSigner implements signer.Signer for testing.
type mockSigner struct {
	addr address.Address
}

func (s *mockSigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	tx.Signature = append(tx.Signature, []byte("fakesig"))
	return tx, nil
}

func (s *mockSigner) Address() address.Address {
	return s.addr
}

// newDummyTx returns a minimal Transaction for testing.
func newDummyTx() *core.Transaction {
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type:      core.Transaction_Contract_TransferContract,
					Parameter: &anypb.Any{Value: []byte("test")},
				},
			},
		},
	}
}

// newDummyTxExt returns a minimal TransactionExtention for testing.
func newDummyTxExt() *api.TransactionExtention {
	return &api.TransactionExtention{
		Transaction: newDummyTx(),
		Txid:        []byte("dummytxid"),
		Result:      &api.Return{Result: true},
	}
}

func TestTransactionID(t *testing.T) {
	tx := newDummyTx()
	id, err := TransactionID(tx)
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	// Verify it matches manual computation.
	raw, err := proto.Marshal(tx.RawData)
	require.NoError(t, err)
	h := sha256.Sum256(raw)
	expected := common.BytesToHexString(h[:])
	assert.Equal(t, expected, id)
}

func TestTransactionID_NilTransaction(t *testing.T) {
	_, err := TransactionID(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing raw data")
}

func TestTransactionID_NilRawData(t *testing.T) {
	_, err := TransactionID(&core.Transaction{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing raw data")
}

func TestSend_Success(t *testing.T) {
	broadcastCalled := false
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, tx *core.Transaction) (*api.Return, error) {
			broadcastCalled = true
			assert.NotEmpty(t, tx.Signature)
			return &api.Return{Result: true, Code: 0}, nil
		},
	}

	receipt, err := Send(context.Background(), b, &mockSigner{}, newDummyTx())
	require.NoError(t, err)
	assert.True(t, broadcastCalled)
	assert.NotEmpty(t, receipt.TxID)
	assert.Empty(t, receipt.Error)
}

func TestSend_BroadcastRejection(t *testing.T) {
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{
				Result:  false,
				Code:    api.Return_CONTRACT_VALIDATE_ERROR,
				Message: []byte("bad contract"),
			}, nil
		},
	}

	receipt, err := Send(context.Background(), b, &mockSigner{}, newDummyTx())
	require.NoError(t, err)
	assert.Equal(t, "bad contract", receipt.Error)
}

func TestSend_BroadcastNetworkError(t *testing.T) {
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return nil, errors.New("network timeout")
		},
	}

	_, err := Send(context.Background(), b, &mockSigner{}, newDummyTx())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
}

func TestSend_EmptyResponse(t *testing.T) {
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return nil, nil
		},
	}

	_, err := Send(context.Background(), b, &mockSigner{}, newDummyTx())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

func TestSendAndConfirm_Success(t *testing.T) {
	callCount := 0
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoFn: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			callCount++
			if callCount < 2 {
				return &core.TransactionInfo{}, nil
			}
			return &core.TransactionInfo{
				BlockNumber: 12345,
				Fee:         100000,
				Receipt: &core.ResourceReceipt{
					EnergyUsageTotal: 50000,
					NetUsage:         300,
				},
				ContractResult: [][]byte{{0x01, 0x02}},
			}, nil
		},
	}

	receipt, err := SendAndConfirm(context.Background(), b, &mockSigner{}, newDummyTx(), 10*time.Millisecond)
	require.NoError(t, err)
	assert.True(t, receipt.Confirmed)
	assert.Equal(t, int64(12345), receipt.BlockNumber)
	assert.Equal(t, int64(100000), receipt.Fee)
	assert.Equal(t, int64(50000), receipt.EnergyUsed)
	assert.Equal(t, int64(300), receipt.BandwidthUsed)
	assert.Equal(t, []byte{0x01, 0x02}, receipt.Result)
}

func TestSendAndConfirm_ContextCancelled(t *testing.T) {
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoFn: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			return &core.TransactionInfo{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := SendAndConfirm(ctx, b, &mockSigner{}, newDummyTx(), 10*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestSendAndConfirm_BroadcastError(t *testing.T) {
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{
				Result:  false,
				Code:    api.Return_CONTRACT_VALIDATE_ERROR,
				Message: []byte("validation failed"),
			}, nil
		},
	}

	receipt, err := SendAndConfirm(context.Background(), b, &mockSigner{}, newDummyTx(), 10*time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, "validation failed", receipt.Error)
	assert.False(t, receipt.Confirmed)
}

func TestSendAndConfirm_FailedTransaction(t *testing.T) {
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoFn: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			return &core.TransactionInfo{
				BlockNumber: 99999,
				Result:      core.TransactionInfo_FAILED,
				ResMessage:  []byte("REVERT opcode"),
			}, nil
		},
	}

	receipt, err := SendAndConfirm(context.Background(), b, &mockSigner{}, newDummyTx(), 10*time.Millisecond)
	require.NoError(t, err)
	assert.True(t, receipt.Confirmed)
	assert.Equal(t, "REVERT opcode", receipt.Error)
}

func TestSendAndConfirm_DefaultPollInterval(t *testing.T) {
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoFn: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			return &core.TransactionInfo{
				BlockNumber: 12345,
			}, nil
		},
	}

	// Passing 0 should use DefaultPollInterval (test still works, just slow).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	receipt, err := SendAndConfirm(ctx, b, &mockSigner{}, newDummyTx(), 0)
	require.NoError(t, err)
	assert.True(t, receipt.Confirmed)
}

func TestSendAndConfirm_NotFoundRetries(t *testing.T) {
	callCount := 0
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoFn: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			callCount++
			if callCount < 3 {
				return nil, errors.New("transaction not found")
			}
			return &core.TransactionInfo{BlockNumber: 100}, nil
		},
	}

	receipt, err := SendAndConfirm(context.Background(), b, &mockSigner{}, newDummyTx(), 10*time.Millisecond)
	require.NoError(t, err)
	assert.True(t, receipt.Confirmed)
	assert.Equal(t, 3, callCount)
}

func TestSendAndConfirm_PermanentError(t *testing.T) {
	b := &mockBroadcaster{
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoFn: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			return nil, errors.New("database error")
		},
	}

	_, err := SendAndConfirm(context.Background(), b, &mockSigner{}, newDummyTx(), 10*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

func TestApplyPermissionID(t *testing.T) {
	tx := newDummyTxExt()
	ApplyPermissionID(tx, 2)
	for _, c := range tx.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestApplyPermissionID_Nil(t *testing.T) {
	// Should not panic on nil inputs.
	ApplyPermissionID(nil, 2)
	ApplyPermissionID(&api.TransactionExtention{}, 2)
	ApplyPermissionID(&api.TransactionExtention{Transaction: &core.Transaction{}}, 2)
}

func TestApplyMemo(t *testing.T) {
	tx := newDummyTxExt()
	ApplyMemo(tx, "hello tron")
	assert.Equal(t, []byte("hello tron"), tx.Transaction.RawData.Data)
}

func TestApplyMemo_Nil(t *testing.T) {
	// Should not panic on nil inputs.
	ApplyMemo(nil, "test")
	ApplyMemo(&api.TransactionExtention{}, "test")
	ApplyMemo(&api.TransactionExtention{Transaction: &core.Transaction{}}, "test")
}

func TestRecomputeTxID(t *testing.T) {
	tx := newDummyTxExt()
	originalTxid := make([]byte, len(tx.Txid))
	copy(originalTxid, tx.Txid)

	// Mutate and recompute.
	tx.Transaction.RawData.Data = []byte("memo")
	err := RecomputeTxID(tx)
	require.NoError(t, err)
	assert.NotEqual(t, originalTxid, tx.Txid)

	// Verify it matches manual computation.
	raw, err := proto.Marshal(tx.Transaction.RawData)
	require.NoError(t, err)
	h := sha256.Sum256(raw)
	assert.Equal(t, h[:], tx.Txid)
}

func TestRecomputeTxID_Nil(t *testing.T) {
	err := RecomputeTxID(nil)
	assert.Error(t, err)

	err = RecomputeTxID(&api.TransactionExtention{})
	assert.Error(t, err)

	err = RecomputeTxID(&api.TransactionExtention{Transaction: &core.Transaction{}})
	assert.Error(t, err)
}

func TestWaitForConfirmation_NilInfo(t *testing.T) {
	callCount := 0
	b := &mockBroadcaster{
		getTransactionInfoFn: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			callCount++
			if callCount < 2 {
				return nil, nil
			}
			return &core.TransactionInfo{BlockNumber: 100}, nil
		},
	}

	receipt := &Receipt{TxID: "abc123"}
	result, err := WaitForConfirmation(context.Background(), b, receipt, 10*time.Millisecond)
	require.NoError(t, err)
	assert.True(t, result.Confirmed)
}
