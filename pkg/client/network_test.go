package client_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransfer(t *testing.T) {
	from := "TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b"
	to := "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"

	mock := &mockWalletServer{
		CreateTransaction2Func: func(_ context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(1_000_000), in.Amount)
			assert.NotEmpty(t, in.OwnerAddress)
			assert.NotEmpty(t, in.ToAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.Transfer(from, to, 1_000_000)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, tx.GetTxid())
}

func TestTransfer_InvalidFrom(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.Transfer("invalid", "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", 1000)
	require.Error(t, err)
}

func TestTransfer_InvalidTo(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.Transfer("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", "invalid", 1000)
	require.Error(t, err)
}

func TestTransfer_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		CreateTransaction2Func: func(_ context.Context, _ *core.TransferContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)
	_, err := c.Transfer("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", 1000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestTransfer_ResultCodeError(t *testing.T) {
	mock := &mockWalletServer{
		CreateTransaction2Func: func(_ context.Context, _ *core.TransferContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Txid:        []byte{0x01},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
				Result: &api.Return{
					Code:    api.Return_BANDWITH_ERROR,
					Message: []byte("not enough bandwidth"),
				},
			}, nil
		},
	}
	c := newMockClient(t, mock)
	_, err := c.Transfer("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", 1000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not enough bandwidth")
}

func TestBroadcast(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}

	c := newMockClient(t, mock)
	result, err := c.Broadcast(&core.Transaction{RawData: &core.TransactionRaw{}})
	require.NoError(t, err)
	assert.True(t, result.GetResult())
}

func TestBroadcast_ResultFalse(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: false, Code: api.Return_SUCCESS, Message: []byte("duplicate")}, nil
		},
	}

	c := newMockClient(t, mock)
	_, err := c.Broadcast(&core.Transaction{RawData: &core.TransactionRaw{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestBroadcast_NonSuccessCode(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_DUP_TRANSACTION_ERROR, Message: []byte("dup tx")}, nil
		},
	}

	c := newMockClient(t, mock)
	_, err := c.Broadcast(&core.Transaction{RawData: &core.TransactionRaw{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "dup tx")
}

func TestBroadcast_RPCError(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return nil, fmt.Errorf("network timeout")
		},
	}

	c := newMockClient(t, mock)
	_, err := c.Broadcast(&core.Transaction{RawData: &core.TransactionRaw{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
}

func TestGetTransactionByID(t *testing.T) {
	txID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	txIDBytes, _ := hex.DecodeString(txID)

	mock := &mockWalletServer{
		GetTransactionByIdFunc: func(_ context.Context, in *api.BytesMessage) (*core.Transaction, error) {
			return &core.Transaction{
				RawData: &core.TransactionRaw{RefBlockBytes: txIDBytes[:2]},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.GetTransactionByID(txID)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestGetTransactionByID_NotFound(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionByIdFunc: func(_ context.Context, _ *api.BytesMessage) (*core.Transaction, error) {
			return &core.Transaction{}, nil // zero-size
		},
	}

	c := newMockClient(t, mock)
	_, err := c.GetTransactionByID("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetTransactionByID_InvalidHex(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetTransactionByID("not-hex")
	require.Error(t, err)
}

func TestGetTransactionInfoByID(t *testing.T) {
	txID := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	txIDBytes, _ := hex.DecodeString(txID)

	mock := &mockWalletServer{
		GetTransactionInfoByIdFunc: func(_ context.Context, _ *api.BytesMessage) (*core.TransactionInfo, error) {
			return &core.TransactionInfo{Id: txIDBytes, Fee: 100000}, nil
		},
	}

	c := newMockClient(t, mock)
	info, err := c.GetTransactionInfoByID(txID)
	require.NoError(t, err)
	assert.Equal(t, int64(100000), info.Fee)
}

func TestGetTransactionInfoByID_NotFound(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionInfoByIdFunc: func(_ context.Context, _ *api.BytesMessage) (*core.TransactionInfo, error) {
			return &core.TransactionInfo{Id: []byte{0x00}}, nil // different ID
		},
	}

	c := newMockClient(t, mock)
	_, err := c.GetTransactionInfoByID("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetNowBlock(t *testing.T) {
	mock := &mockWalletServer{
		GetNowBlock2Func: func(_ context.Context, _ *api.EmptyMessage) (*api.BlockExtention, error) {
			return &api.BlockExtention{
				BlockHeader: &core.BlockHeader{
					RawData: &core.BlockHeaderRaw{Number: 12345},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	block, err := c.GetNowBlock()
	require.NoError(t, err)
	assert.Equal(t, int64(12345), block.BlockHeader.RawData.Number)
}

func TestGetBlockInfoByNum(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionInfoByBlockNumFunc: func(_ context.Context, in *api.NumberMessage) (*api.TransactionInfoList, error) {
			assert.Equal(t, int64(100), in.Num)
			return &api.TransactionInfoList{
				TransactionInfo: []*core.TransactionInfo{
					{Fee: 50000},
					{Fee: 60000},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	info, err := c.GetBlockInfoByNum(100)
	require.NoError(t, err)
	assert.Len(t, info.TransactionInfo, 2)
}

func TestListNodes(t *testing.T) {
	mock := &mockWalletServer{
		ListNodesFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.NodeList, error) {
			return &api.NodeList{
				Nodes: []*api.Node{
					{Address: &api.Address{Host: []byte("node1.tron.io"), Port: 50051}},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	nodes, err := c.ListNodes()
	require.NoError(t, err)
	require.Len(t, nodes.Nodes, 1)
	assert.Equal(t, "node1.tron.io", string(nodes.Nodes[0].Address.Host))
}

func TestGetNextMaintenanceTime(t *testing.T) {
	mock := &mockWalletServer{
		GetNextMaintenanceTimeFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			return &api.NumberMessage{Num: 1700000000000}, nil
		},
	}

	c := newMockClient(t, mock)
	result, err := c.GetNextMaintenanceTime()
	require.NoError(t, err)
	assert.Equal(t, int64(1700000000000), result.Num)
}

func TestTotalTransaction(t *testing.T) {
	mock := &mockWalletServer{
		TotalTransactionFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			return &api.NumberMessage{Num: 99_000_000}, nil
		},
	}

	c := newMockClient(t, mock)
	result, err := c.TotalTransaction()
	require.NoError(t, err)
	assert.Equal(t, int64(99_000_000), result.Num)
}

func TestGetNodeInfo(t *testing.T) {
	mock := &mockWalletServer{
		GetNodeInfoFunc: func(_ context.Context, _ *api.EmptyMessage) (*core.NodeInfo, error) {
			return &core.NodeInfo{
				BeginSyncNum:        1000,
				CurrentConnectCount: 25,
			}, nil
		},
	}

	c := newMockClient(t, mock)
	info, err := c.GetNodeInfo()
	require.NoError(t, err)
	assert.Equal(t, int64(1000), info.BeginSyncNum)
	assert.Equal(t, int32(25), info.CurrentConnectCount)
}

func TestGetWitnessBrokerage(t *testing.T) {
	mock := &mockWalletServer{
		GetBrokerageInfoFunc: func(_ context.Context, _ *api.BytesMessage) (*api.NumberMessage, error) {
			return &api.NumberMessage{Num: 20}, nil
		},
	}

	c := newMockClient(t, mock)
	brokerage, err := c.GetWitnessBrokerage(accountAddressWitness)
	require.NoError(t, err)
	assert.Equal(t, float64(20), brokerage)
}

func TestGetWitnessBrokerage_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetWitnessBrokerage("invalid")
	require.Error(t, err)
}

func TestListWitnesses(t *testing.T) {
	mock := &mockWalletServer{
		ListWitnessesFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.WitnessList, error) {
			return &api.WitnessList{
				Witnesses: []*core.Witness{
					{VoteCount: 1000, TotalProduced: 500},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	list, err := c.ListWitnesses()
	require.NoError(t, err)
	require.Len(t, list.Witnesses, 1)
	assert.Equal(t, int64(1000), list.Witnesses[0].VoteCount)
}

func TestVoteWitnessAccount(t *testing.T) {
	mock := &mockWalletServer{
		VoteWitnessAccount2Func: func(_ context.Context, in *core.VoteWitnessContract) (*api.TransactionExtention, error) {
			assert.Len(t, in.Votes, 1)
			assert.Equal(t, int64(100), in.Votes[0].VoteCount)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	votes := map[string]int64{
		accountAddressWitness: 100,
	}
	tx, err := c.VoteWitnessAccount(accountAddress, votes)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestVoteWitnessAccount_InvalidVoter(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.VoteWitnessAccount("invalid", map[string]int64{accountAddressWitness: 100})
	require.Error(t, err)
}

func TestVoteWitnessAccount_InvalidWitness(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.VoteWitnessAccount(accountAddress, map[string]int64{"invalid": 100})
	require.Error(t, err)
}

func TestCreateWitness(t *testing.T) {
	mock := &mockWalletServer{
		CreateWitness2Func: func(_ context.Context, in *core.WitnessCreateContract) (*api.TransactionExtention, error) {
			assert.Equal(t, "https://example.com", string(in.Url))
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.CreateWitness(accountAddress, "https://example.com")
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestGetTransactionSignWeight(t *testing.T) {
	mock := &mockWalletServer{
		GetTransactionSignWeightFunc: func(_ context.Context, _ *core.Transaction) (*api.TransactionSignWeight, error) {
			return &api.TransactionSignWeight{
				Result: &api.TransactionSignWeight_Result{
					Code: api.TransactionSignWeight_Result_ENOUGH_PERMISSION,
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	weight, err := c.GetTransactionSignWeight(&core.Transaction{RawData: &core.TransactionRaw{}})
	require.NoError(t, err)
	assert.Equal(t, api.TransactionSignWeight_Result_ENOUGH_PERMISSION, weight.Result.Code)
}
