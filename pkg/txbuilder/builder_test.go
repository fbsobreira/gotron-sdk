package txbuilder

import (
	"context"
	"fmt"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"
)

// mockClient implements the Client interface for testing.
type mockClient struct {
	transferFn               func(ctx context.Context, from, to string, amount int64) (*api.TransactionExtention, error)
	broadcastFn              func(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	getTransactionInfoFn     func(ctx context.Context, id string) (*core.TransactionInfo, error)
	freezeV2Fn               func(ctx context.Context, from string, resource core.ResourceCode, amount int64) (*api.TransactionExtention, error)
	unfreezeV2Fn             func(ctx context.Context, from string, resource core.ResourceCode, amount int64) (*api.TransactionExtention, error)
	delegateResourceFn       func(ctx context.Context, from, to string, resource core.ResourceCode, amount int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error)
	unDelegateResourceFn     func(ctx context.Context, from, to string, resource core.ResourceCode, amount int64) (*api.TransactionExtention, error)
	voteWitnessAccountFn     func(ctx context.Context, from string, votes map[string]int64) (*api.TransactionExtention, error)
	withdrawExpireUnfreezeFn func(ctx context.Context, from string, timestamp int64) (*api.TransactionExtention, error)
}

func (m *mockClient) TransferCtx(ctx context.Context, from, to string, amount int64) (*api.TransactionExtention, error) {
	if m.transferFn != nil {
		return m.transferFn(ctx, from, to, amount)
	}
	return nil, fmt.Errorf("TransferCtx not implemented")
}

func (m *mockClient) BroadcastCtx(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	if m.broadcastFn != nil {
		return m.broadcastFn(ctx, tx)
	}
	return nil, fmt.Errorf("BroadcastCtx not implemented")
}

func (m *mockClient) GetTransactionInfoByIDCtx(ctx context.Context, id string) (*core.TransactionInfo, error) {
	if m.getTransactionInfoFn != nil {
		return m.getTransactionInfoFn(ctx, id)
	}
	return nil, fmt.Errorf("GetTransactionInfoByIDCtx not implemented")
}

func (m *mockClient) FreezeBalanceV2Ctx(ctx context.Context, from string, resource core.ResourceCode, amount int64) (*api.TransactionExtention, error) {
	if m.freezeV2Fn != nil {
		return m.freezeV2Fn(ctx, from, resource, amount)
	}
	return nil, fmt.Errorf("FreezeBalanceV2Ctx not implemented")
}

func (m *mockClient) UnfreezeBalanceV2Ctx(ctx context.Context, from string, resource core.ResourceCode, amount int64) (*api.TransactionExtention, error) {
	if m.unfreezeV2Fn != nil {
		return m.unfreezeV2Fn(ctx, from, resource, amount)
	}
	return nil, fmt.Errorf("UnfreezeBalanceV2Ctx not implemented")
}

func (m *mockClient) DelegateResourceCtx(ctx context.Context, from, to string, resource core.ResourceCode, amount int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
	if m.delegateResourceFn != nil {
		return m.delegateResourceFn(ctx, from, to, resource, amount, lock, lockPeriod)
	}
	return nil, fmt.Errorf("DelegateResourceCtx not implemented")
}

func (m *mockClient) UnDelegateResourceCtx(ctx context.Context, from, to string, resource core.ResourceCode, amount int64) (*api.TransactionExtention, error) {
	if m.unDelegateResourceFn != nil {
		return m.unDelegateResourceFn(ctx, from, to, resource, amount)
	}
	return nil, fmt.Errorf("UnDelegateResourceCtx not implemented")
}

func (m *mockClient) VoteWitnessAccountCtx(ctx context.Context, from string, votes map[string]int64) (*api.TransactionExtention, error) {
	if m.voteWitnessAccountFn != nil {
		return m.voteWitnessAccountFn(ctx, from, votes)
	}
	return nil, fmt.Errorf("VoteWitnessAccountCtx not implemented")
}

func (m *mockClient) WithdrawExpireUnfreezeCtx(ctx context.Context, from string, timestamp int64) (*api.TransactionExtention, error) {
	if m.withdrawExpireUnfreezeFn != nil {
		return m.withdrawExpireUnfreezeFn(ctx, from, timestamp)
	}
	return nil, fmt.Errorf("WithdrawExpireUnfreezeCtx not implemented")
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

// newDummyTxExt returns a minimal TransactionExtention for testing.
func newDummyTxExt() *api.TransactionExtention {
	return &api.TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{
					{
						Type:         core.Transaction_Contract_TransferContract,
						Parameter:    &anypb.Any{Value: []byte("test")},
						PermissionId: 0,
					},
				},
			},
		},
		Txid:   []byte("dummytxid"),
		Result: &api.Return{Result: true},
	}
}

func TestTransfer_Build(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, from, to string, amount int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "TFromAddr", from)
			assert.Equal(t, "TToAddr", to)
			assert.Equal(t, int64(1000000), amount)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.Transfer("TFromAddr", "TToAddr", 1000000).Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
	require.NotNil(t, ext.Transaction)
}

func TestTransfer_BuildWithPermissionID(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.Transfer("TFrom", "TTo", 100, WithPermissionID(2)).Build(context.Background())
	require.NoError(t, err)

	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestTransfer_BuildWithMemo(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.Transfer("TFrom", "TTo", 100, WithMemo("hello tron")).Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("hello tron"), ext.Transaction.RawData.Data)
}

func TestTransfer_Send(t *testing.T) {
	broadcastCalled := false
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
		broadcastFn: func(_ context.Context, tx *core.Transaction) (*api.Return, error) {
			broadcastCalled = true
			assert.NotEmpty(t, tx.Signature, "transaction should be signed")
			return &api.Return{Result: true, Code: 0}, nil
		},
	}

	s := &mockSigner{}
	b := New(mc)
	receipt, err := b.Transfer("TFrom", "TTo", 100).Send(context.Background(), s)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.True(t, broadcastCalled)
	assert.NotEmpty(t, receipt.TxID)
	assert.Empty(t, receipt.Error)
}

func TestTransfer_Send_BroadcastError(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{
				Result:  false,
				Code:    api.Return_CONTRACT_VALIDATE_ERROR,
				Message: []byte("bad contract"),
			}, nil
		},
	}

	s := &mockSigner{}
	b := New(mc)
	receipt, err := b.Transfer("TFrom", "TTo", 100).Send(context.Background(), s)
	require.NoError(t, err)
	assert.Equal(t, "bad contract", receipt.Error)
}

func TestFreezeV2_Build(t *testing.T) {
	mc := &mockClient{
		freezeV2Fn: func(_ context.Context, from string, resource core.ResourceCode, amount int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "TOwner", from)
			assert.Equal(t, core.ResourceCode_BANDWIDTH, resource)
			assert.Equal(t, int64(5000000), amount)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.FreezeV2("TOwner", 5000000, core.ResourceCode_BANDWIDTH).Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestDelegateResource_WithLock(t *testing.T) {
	mc := &mockClient{
		delegateResourceFn: func(_ context.Context, from, to string, resource core.ResourceCode, amount int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "TOwner", from)
			assert.Equal(t, "TReceiver", to)
			assert.True(t, lock)
			assert.Equal(t, int64(86400), lockPeriod)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.DelegateResource("TOwner", "TReceiver", core.ResourceCode_ENERGY, 1000000).
		Lock(86400).
		Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestDelegateResource_WithoutLock(t *testing.T) {
	mc := &mockClient{
		delegateResourceFn: func(_ context.Context, _, _ string, _ core.ResourceCode, _ int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
			assert.False(t, lock)
			assert.Equal(t, int64(0), lockPeriod)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.DelegateResource("TOwner", "TReceiver", core.ResourceCode_ENERGY, 1000000).
		Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestVoteWitness_FluentVotes(t *testing.T) {
	mc := &mockClient{
		voteWitnessAccountFn: func(_ context.Context, from string, v map[string]int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "TVoter", from)
			assert.Equal(t, map[string]int64{"TWitness1": 100, "TWitness2": 200}, v)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.VoteWitness("TVoter").
		Vote("TWitness1", 100).
		Vote("TWitness2", 200).
		Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestVoteWitness_VotesMap(t *testing.T) {
	votes := map[string]int64{"TWitness1": 100, "TWitness2": 200}
	mc := &mockClient{
		voteWitnessAccountFn: func(_ context.Context, from string, v map[string]int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "TVoter", from)
			assert.Equal(t, votes, v)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.VoteWitness("TVoter").
		Votes(votes).
		Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestVoteWitness_MixedVotes(t *testing.T) {
	mc := &mockClient{
		voteWitnessAccountFn: func(_ context.Context, _ string, v map[string]int64) (*api.TransactionExtention, error) {
			assert.Len(t, v, 3)
			assert.Equal(t, int64(100), v["TW1"])
			assert.Equal(t, int64(200), v["TW2"])
			assert.Equal(t, int64(300), v["TW3"])
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.VoteWitness("TVoter").
		Vote("TW1", 100).
		Votes(map[string]int64{"TW2": 200, "TW3": 300}).
		Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestBuild_Error(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return nil, fmt.Errorf("rpc error")
		},
	}

	b := New(mc)
	ext, err := b.Transfer("TFrom", "TTo", 100).Build(context.Background())
	assert.Error(t, err)
	assert.Nil(t, ext)
	assert.Contains(t, err.Error(), "rpc error")
}

func TestBuilder_SharedDefaults(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc, WithPermissionID(2))
	ext, err := b.Transfer("TFrom", "TTo", 100).Build(context.Background())
	require.NoError(t, err)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestBuilder_PerCallOverridesDefaults(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc, WithPermissionID(2), WithMemo("default"))
	ext, err := b.Transfer("TFrom", "TTo", 100, WithMemo("override")).Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("override"), ext.Transaction.RawData.Data)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestUnfreezeV2_Build(t *testing.T) {
	mc := &mockClient{
		unfreezeV2Fn: func(_ context.Context, from string, resource core.ResourceCode, amount int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "TOwner", from)
			assert.Equal(t, core.ResourceCode_ENERGY, resource)
			assert.Equal(t, int64(3000000), amount)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.UnfreezeV2("TOwner", 3000000, core.ResourceCode_ENERGY).Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestUnDelegateResource_Build(t *testing.T) {
	mc := &mockClient{
		unDelegateResourceFn: func(_ context.Context, from, to string, resource core.ResourceCode, amount int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "TOwner", from)
			assert.Equal(t, "TReceiver", to)
			assert.Equal(t, core.ResourceCode_BANDWIDTH, resource)
			assert.Equal(t, int64(2000000), amount)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.UnDelegateResource("TOwner", "TReceiver", core.ResourceCode_BANDWIDTH, 2000000).Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestVoteWitness_Send(t *testing.T) {
	broadcastCalled := false
	mc := &mockClient{
		voteWitnessAccountFn: func(_ context.Context, _ string, v map[string]int64) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(500), v["TWitness1"])
			return newDummyTxExt(), nil
		},
		broadcastFn: func(_ context.Context, tx *core.Transaction) (*api.Return, error) {
			broadcastCalled = true
			assert.NotEmpty(t, tx.Signature)
			return &api.Return{Result: true, Code: 0}, nil
		},
	}

	s := &mockSigner{}
	b := New(mc)
	receipt, err := b.VoteWitness("TVoter").
		Vote("TWitness1", 500).
		Send(context.Background(), s)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.True(t, broadcastCalled)
	assert.NotEmpty(t, receipt.TxID)
}

func TestDelegateResource_Send(t *testing.T) {
	broadcastCalled := false
	mc := &mockClient{
		delegateResourceFn: func(_ context.Context, _, _ string, _ core.ResourceCode, _ int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
			assert.True(t, lock)
			assert.Equal(t, int64(172800), lockPeriod)
			return newDummyTxExt(), nil
		},
		broadcastFn: func(_ context.Context, tx *core.Transaction) (*api.Return, error) {
			broadcastCalled = true
			return &api.Return{Result: true, Code: 0}, nil
		},
	}

	s := &mockSigner{}
	b := New(mc)
	receipt, err := b.DelegateResource("TOwner", "TReceiver", core.ResourceCode_ENERGY, 1000000).
		Lock(172800).
		Send(context.Background(), s)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.True(t, broadcastCalled)
}

func TestSendAndConfirm_Success(t *testing.T) {
	callCount := 0
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoFn: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			callCount++
			if callCount < 2 {
				return &core.TransactionInfo{}, nil // block 0 = not confirmed yet
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

	s := &mockSigner{}
	b := New(mc)
	receipt, err := b.Transfer("TFrom", "TTo", 100).SendAndConfirm(context.Background(), s)
	require.NoError(t, err)
	assert.True(t, receipt.Confirmed)
	assert.Equal(t, int64(12345), receipt.BlockNumber)
	assert.Equal(t, int64(100000), receipt.Fee)
	assert.Equal(t, int64(50000), receipt.EnergyUsed)
	assert.Equal(t, int64(300), receipt.BandwidthUsed)
	assert.Equal(t, []byte{0x01, 0x02}, receipt.Result)
}

func TestSendAndConfirm_ContextCancelled(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoFn: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			return &core.TransactionInfo{}, nil // never confirms
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately so the next ticker fires after context is done.
	cancel()

	s := &mockSigner{}
	b := New(mc)
	_, err := b.Transfer("TFrom", "TTo", 100).SendAndConfirm(ctx, s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestSendAndConfirm_BroadcastError(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{
				Result:  false,
				Code:    api.Return_CONTRACT_VALIDATE_ERROR,
				Message: []byte("validation failed"),
			}, nil
		},
	}

	s := &mockSigner{}
	b := New(mc)
	receipt, err := b.Transfer("TFrom", "TTo", 100).SendAndConfirm(context.Background(), s)
	require.NoError(t, err)
	// Should return early with error in receipt, not poll for confirmation.
	assert.Equal(t, "validation failed", receipt.Error)
	assert.False(t, receipt.Confirmed)
}

func TestSendAndConfirm_FailedTransaction(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
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

	s := &mockSigner{}
	b := New(mc)
	receipt, err := b.Transfer("TFrom", "TTo", 100).SendAndConfirm(context.Background(), s)
	require.NoError(t, err)
	assert.True(t, receipt.Confirmed)
	assert.Equal(t, "REVERT opcode", receipt.Error)
}

func TestVoteWitness_EmptyVotes(t *testing.T) {
	mc := &mockClient{
		voteWitnessAccountFn: func(_ context.Context, _ string, v map[string]int64) (*api.TransactionExtention, error) {
			assert.Empty(t, v)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.VoteWitness("TVoter").Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestVoteWitness_OverwriteVote(t *testing.T) {
	mc := &mockClient{
		voteWitnessAccountFn: func(_ context.Context, _ string, v map[string]int64) (*api.TransactionExtention, error) {
			assert.Len(t, v, 1)
			assert.Equal(t, int64(300), v["TW1"]) // second call overwrites first
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.VoteWitness("TVoter").
		Vote("TW1", 100).
		Vote("TW1", 300).
		Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestVoteWitness_WithMemo(t *testing.T) {
	mc := &mockClient{
		voteWitnessAccountFn: func(_ context.Context, _ string, _ map[string]int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.VoteWitness("TVoter", WithMemo("my vote")).
		Vote("TW1", 100).
		Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("my vote"), ext.Transaction.RawData.Data)
}

func TestDelegateResource_WithPermissionID(t *testing.T) {
	mc := &mockClient{
		delegateResourceFn: func(_ context.Context, _, _ string, _ core.ResourceCode, _ int64, _ bool, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.DelegateResource("TOwner", "TReceiver", core.ResourceCode_ENERGY, 1000000, WithPermissionID(2)).
		Lock(86400).
		Build(context.Background())
	require.NoError(t, err)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestTransfer_BuildError(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	s := &mockSigner{}
	b := New(mc)
	_, err := b.Transfer("TFrom", "TTo", 100).Send(context.Background(), s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestTransfer_BroadcastNetworkError(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
		broadcastFn: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return nil, fmt.Errorf("network timeout")
		},
	}

	s := &mockSigner{}
	b := New(mc)
	_, err := b.Transfer("TFrom", "TTo", 100).Send(context.Background(), s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
}

// --- Fluent WithMemo / WithPermissionID tests ---

func TestTransfer_FluentWithMemo(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.Transfer("TFrom", "TTo", 100).
		WithMemo("fluent memo").
		Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("fluent memo"), ext.Transaction.RawData.Data)
}

func TestTransfer_FluentWithPermissionID(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.Transfer("TFrom", "TTo", 100).
		WithPermissionID(3).
		Build(context.Background())
	require.NoError(t, err)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(3), c.PermissionId)
	}
}

func TestTransfer_FluentChainBoth(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.Transfer("TFrom", "TTo", 100).
		WithMemo("payment").
		WithPermissionID(2).
		Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("payment"), ext.Transaction.RawData.Data)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestTransfer_FluentOverridesOption(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.Transfer("TFrom", "TTo", 100, WithMemo("from option")).
		WithMemo("from fluent").
		Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("from fluent"), ext.Transaction.RawData.Data)
}

func TestDelegateResource_FluentChain(t *testing.T) {
	mc := &mockClient{
		delegateResourceFn: func(_ context.Context, _, _ string, _ core.ResourceCode, _ int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
			assert.True(t, lock)
			assert.Equal(t, int64(86400), lockPeriod)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.DelegateResource("TOwner", "TReceiver", core.ResourceCode_ENERGY, 1000000).
		Lock(86400).
		WithMemo("delegate with lock").
		WithPermissionID(2).
		Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("delegate with lock"), ext.Transaction.RawData.Data)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestDelegateResource_FluentAnyOrder(t *testing.T) {
	mc := &mockClient{
		delegateResourceFn: func(_ context.Context, _, _ string, _ core.ResourceCode, _ int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error) {
			assert.True(t, lock)
			assert.Equal(t, int64(172800), lockPeriod)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	// WithMemo before Lock — order shouldn't matter
	ext, err := b.DelegateResource("TOwner", "TReceiver", core.ResourceCode_ENERGY, 1000000).
		WithMemo("memo first").
		Lock(172800).
		Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("memo first"), ext.Transaction.RawData.Data)
}

func TestVoteWitness_FluentChain(t *testing.T) {
	mc := &mockClient{
		voteWitnessAccountFn: func(_ context.Context, _ string, v map[string]int64) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(100), v["TW1"])
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.VoteWitness("TVoter").
		Vote("TW1", 100).
		WithMemo("my votes").
		WithPermissionID(2).
		Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("my votes"), ext.Transaction.RawData.Data)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestVoteWitness_FluentAnyOrder(t *testing.T) {
	mc := &mockClient{
		voteWitnessAccountFn: func(_ context.Context, _ string, v map[string]int64) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(500), v["TW1"])
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	// WithPermissionID before Vote — order shouldn't matter
	ext, err := b.VoteWitness("TVoter").
		WithPermissionID(3).
		Vote("TW1", 500).
		Build(context.Background())
	require.NoError(t, err)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(3), c.PermissionId)
	}
}

// --- WithdrawExpireUnfreeze tests ---

func TestWithdrawExpireUnfreeze_Build(t *testing.T) {
	mc := &mockClient{
		withdrawExpireUnfreezeFn: func(_ context.Context, from string, timestamp int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "TOwner", from)
			assert.Equal(t, int64(1700000000), timestamp)
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.WithdrawExpireUnfreeze("TOwner", 1700000000).Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestWithdrawExpireUnfreeze_WithMemoAndPermission(t *testing.T) {
	mc := &mockClient{
		withdrawExpireUnfreezeFn: func(_ context.Context, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.WithdrawExpireUnfreeze("TOwner", 1700000000).
		WithMemo("withdraw").
		WithPermissionID(2).
		Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("withdraw"), ext.Transaction.RawData.Data)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestWithdrawExpireUnfreeze_Send(t *testing.T) {
	broadcastCalled := false
	mc := &mockClient{
		withdrawExpireUnfreezeFn: func(_ context.Context, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
		broadcastFn: func(_ context.Context, tx *core.Transaction) (*api.Return, error) {
			broadcastCalled = true
			assert.NotEmpty(t, tx.Signature)
			return &api.Return{Result: true, Code: 0}, nil
		},
	}

	s := &mockSigner{}
	b := New(mc)
	receipt, err := b.WithdrawExpireUnfreeze("TOwner", 1700000000).Send(context.Background(), s)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	assert.True(t, broadcastCalled)
	assert.NotEmpty(t, receipt.TxID)
}

func TestWithdrawExpireUnfreeze_WithOption(t *testing.T) {
	mc := &mockClient{
		withdrawExpireUnfreezeFn: func(_ context.Context, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc)
	ext, err := b.WithdrawExpireUnfreeze("TOwner", 1700000000, WithMemo("via option")).
		Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("via option"), ext.Transaction.RawData.Data)
}

func TestBuilder_DefaultsDoNotMutate(t *testing.T) {
	mc := &mockClient{
		transferFn: func(_ context.Context, _, _ string, _ int64) (*api.TransactionExtention, error) {
			return newDummyTxExt(), nil
		},
	}

	b := New(mc, WithMemo("default"))

	// First call overrides memo
	ext1, err := b.Transfer("TFrom", "TTo", 100, WithMemo("first")).Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("first"), ext1.Transaction.RawData.Data)

	// Second call should still get the original default, not "first"
	ext2, err := b.Transfer("TFrom", "TTo", 200).Build(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []byte("default"), ext2.Transaction.RawData.Data)
}
