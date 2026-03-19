package contract

import (
	"context"
	"errors"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// newTestTxExt returns a minimal TransactionExtention for testing.
func newTestTxExt() *api.TransactionExtention {
	return &api.TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{
					{
						Type:      core.Transaction_Contract_TriggerSmartContract,
						Parameter: &anypb.Any{Value: []byte("test")},
					},
				},
			},
		},
		Txid:   []byte("dummytxid"),
		Result: &api.Return{Result: true},
	}
}

// mockClient implements the Client interface for testing.
type mockClient struct {
	triggerConstantContractCtxFunc         func(ctx context.Context, from, contractAddress, method, jsonString string, opts ...client.ConstantCallOption) (*api.TransactionExtention, error)
	triggerContractCtxFunc                 func(ctx context.Context, from, contractAddress, method, jsonString string, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error)
	triggerConstantContractWithDataCtxFunc func(ctx context.Context, from, contractAddress string, data []byte, opts ...client.ConstantCallOption) (*api.TransactionExtention, error)
	triggerContractWithDataCtxFunc         func(ctx context.Context, from, contractAddress string, data []byte, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error)
	estimateEnergyCtxFunc                  func(ctx context.Context, from, contractAddress, method, jsonString string, tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error)
	estimateEnergyWithDataCtxFunc          func(ctx context.Context, from, contractAddress string, data []byte, tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error)
	broadcastCtxFunc                       func(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	getTransactionInfoByIDCtxFunc          func(ctx context.Context, id string) (*core.TransactionInfo, error)
}

func (m *mockClient) TriggerConstantContractCtx(ctx context.Context, from, contractAddress, method, jsonString string, opts ...client.ConstantCallOption) (*api.TransactionExtention, error) {
	if m.triggerConstantContractCtxFunc != nil {
		return m.triggerConstantContractCtxFunc(ctx, from, contractAddress, method, jsonString, opts...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) TriggerContractCtx(ctx context.Context, from, contractAddress, method, jsonString string, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	if m.triggerContractCtxFunc != nil {
		return m.triggerContractCtxFunc(ctx, from, contractAddress, method, jsonString, feeLimit, tAmount, tTokenID, tTokenAmount)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) TriggerConstantContractWithDataCtx(ctx context.Context, from, contractAddress string, data []byte, opts ...client.ConstantCallOption) (*api.TransactionExtention, error) {
	if m.triggerConstantContractWithDataCtxFunc != nil {
		return m.triggerConstantContractWithDataCtxFunc(ctx, from, contractAddress, data, opts...)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) TriggerContractWithDataCtx(ctx context.Context, from, contractAddress string, data []byte, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	if m.triggerContractWithDataCtxFunc != nil {
		return m.triggerContractWithDataCtxFunc(ctx, from, contractAddress, data, feeLimit, tAmount, tTokenID, tTokenAmount)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) EstimateEnergyCtx(ctx context.Context, from, contractAddress, method, jsonString string, tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error) {
	if m.estimateEnergyCtxFunc != nil {
		return m.estimateEnergyCtxFunc(ctx, from, contractAddress, method, jsonString, tAmount, tTokenID, tTokenAmount)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) EstimateEnergyWithDataCtx(ctx context.Context, from, contractAddress string, data []byte, tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error) {
	if m.estimateEnergyWithDataCtxFunc != nil {
		return m.estimateEnergyWithDataCtxFunc(ctx, from, contractAddress, data, tAmount, tTokenID, tTokenAmount)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) BroadcastCtx(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	if m.broadcastCtxFunc != nil {
		return m.broadcastCtxFunc(ctx, tx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClient) GetTransactionInfoByIDCtx(ctx context.Context, id string) (*core.TransactionInfo, error) {
	if m.getTransactionInfoByIDCtxFunc != nil {
		return m.getTransactionInfoByIDCtxFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func TestMethodChaining(t *testing.T) {
	mc := &mockClient{}
	call := New(mc, "TContractAddr").
		Method("transfer(address,uint256)").
		From("TFromAddr").
		Params(`[{"address":"TToAddr"},{"uint256":"1000"}]`).
		Apply(WithFeeLimit(100000000), WithCallValue(500))

	assert.Equal(t, "TContractAddr", call.contractAddress)
	assert.Equal(t, "transfer(address,uint256)", call.method)
	assert.Equal(t, "TFromAddr", call.from)
	assert.Equal(t, `[{"address":"TToAddr"},{"uint256":"1000"}]`, call.jsonParams)
	assert.Equal(t, int64(100000000), call.cfg.feeLimit)
	assert.Equal(t, int64(500), call.cfg.callValue)
}

func TestCallReturnsResults(t *testing.T) {
	expectedResult := [][]byte{{0x01, 0x02, 0x03}}

	mc := &mockClient{
		triggerConstantContractCtxFunc: func(_ context.Context, from, contractAddress, method, jsonString string, _ ...client.ConstantCallOption) (*api.TransactionExtention, error) {
			assert.Equal(t, zeroAddress, from) // no From set, should use zero
			assert.Equal(t, "TContract", contractAddress)
			assert.Equal(t, "balanceOf(address)", method)
			return &api.TransactionExtention{
				ConstantResult: expectedResult,
				EnergyUsed:     42,
			}, nil
		},
	}

	result, err := New(mc, "TContract").
		Method("balanceOf(address)").
		Params(`[{"address":"TOwner"}]`).
		Call(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result.RawResults)
	assert.Equal(t, int64(42), result.EnergyUsed)
}

func TestCallWithData(t *testing.T) {
	inputData := []byte{0x70, 0xa0, 0x82, 0x31}
	expectedResult := [][]byte{{0xAA, 0xBB}}

	mc := &mockClient{
		triggerConstantContractWithDataCtxFunc: func(_ context.Context, from, contractAddress string, data []byte, _ ...client.ConstantCallOption) (*api.TransactionExtention, error) {
			assert.Equal(t, inputData, data)
			assert.Equal(t, "TContract", contractAddress)
			return &api.TransactionExtention{
				ConstantResult: expectedResult,
			}, nil
		},
	}

	result, err := New(mc, "TContract").
		WithData(inputData).
		Call(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expectedResult, result.RawResults)
}

func TestBuildReturnsTransaction(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, from, contractAddress, method, jsonString string, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "TFrom", from)
			assert.Equal(t, "TContract", contractAddress)
			assert.Equal(t, "transfer(address,uint256)", method)
			assert.Equal(t, int64(50000000), feeLimit)
			assert.Equal(t, int64(1000), tAmount)
			return &api.TransactionExtention{
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{
						FeeLimit: feeLimit,
					},
				},
			}, nil
		},
	}

	tx, err := New(mc, "TContract").
		Method("transfer(address,uint256)").
		From("TFrom").
		Params(`[{"address":"TTo"},{"uint256":"100"}]`).
		Apply(WithFeeLimit(50000000), WithCallValue(1000)).
		Build(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, int64(50000000), tx.Transaction.RawData.FeeLimit)
}

func TestBuildWithDataPath(t *testing.T) {
	inputData := []byte{0x01, 0x02, 0x03, 0x04}

	mc := &mockClient{
		triggerContractWithDataCtxFunc: func(_ context.Context, from, contractAddress string, data []byte, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
			assert.Equal(t, inputData, data)
			assert.Equal(t, int64(10000000), feeLimit)
			return &api.TransactionExtention{
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{
						FeeLimit: feeLimit,
					},
				},
			}, nil
		},
	}

	tx, err := New(mc, "TContract").
		From("TFrom").
		WithData(inputData).
		Apply(WithFeeLimit(10000000)).
		Build(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, tx)
}

func TestBuildRequiresFrom(t *testing.T) {
	mc := &mockClient{}

	_, err := New(mc, "TContract").
		Method("transfer(address,uint256)").
		Build(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "From address is required")
}

func TestOptionsApplied(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(100000000), feeLimit)
			assert.Equal(t, int64(500), tAmount)
			assert.Equal(t, "1000001", tTokenID)
			assert.Equal(t, int64(200), tTokenAmount)
			return &api.TransactionExtention{
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{},
				},
			}, nil
		},
	}

	_, err := New(mc, "TContract").
		Method("test()").
		From("TFrom").
		Apply(
			WithFeeLimit(100000000),
			WithCallValue(500),
			WithTokenValue("1000001", 200),
		).
		Build(context.Background())

	require.NoError(t, err)
}

func TestPermissionIDApplied(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{
						Contract: []*core.Transaction_Contract{
							{
								PermissionId: 0,
							},
						},
					},
				},
			}, nil
		},
	}

	tx, err := New(mc, "TContract").
		Method("test()").
		From("TFrom").
		Apply(WithPermissionID(2)).
		Build(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int32(2), tx.Transaction.RawData.Contract[0].PermissionId)
}

func TestEstimateEnergy(t *testing.T) {
	mc := &mockClient{
		estimateEnergyCtxFunc: func(_ context.Context, from, contractAddress, _, _ string, _ int64, _ string, _ int64) (*api.EstimateEnergyMessage, error) {
			assert.Equal(t, "TFrom", from)
			assert.Equal(t, "TContract", contractAddress)
			return &api.EstimateEnergyMessage{
				EnergyRequired: 32000,
			}, nil
		},
	}

	energy, err := New(mc, "TContract").
		Method("transfer(address,uint256)").
		From("TFrom").
		Params(`[{"address":"TTo"},{"uint256":"100"}]`).
		EstimateEnergy(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(32000), energy)
}

func TestEstimateEnergyWithData(t *testing.T) {
	mc := &mockClient{
		estimateEnergyWithDataCtxFunc: func(_ context.Context, from, contractAddress string, data []byte, _ int64, _ string, _ int64) (*api.EstimateEnergyMessage, error) {
			assert.Equal(t, "TFrom", from)
			assert.Equal(t, []byte{0x01}, data)
			return &api.EstimateEnergyMessage{
				EnergyRequired: 45000,
			}, nil
		},
	}

	energy, err := New(mc, "TContract").
		From("TFrom").
		WithData([]byte{0x01}).
		EstimateEnergy(context.Background())

	require.NoError(t, err)
	assert.Equal(t, int64(45000), energy)
}

func TestEstimateEnergyRequiresFrom(t *testing.T) {
	mc := &mockClient{}

	_, err := New(mc, "TContract").
		Method("test()").
		EstimateEnergy(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "From address is required")
}

func TestWithABI(t *testing.T) {
	mc := &mockClient{}
	call := New(mc, "TContract").WithABI(`[{"name":"test"}]`)
	assert.Equal(t, `[{"name":"test"}]`, call.abiJSON)
}

func TestCallWithCallValue(t *testing.T) {
	mc := &mockClient{
		triggerConstantContractCtxFunc: func(_ context.Context, _, _, _, _ string, opts ...client.ConstantCallOption) (*api.TransactionExtention, error) {
			// Verify that an option was passed (callValue > 0 triggers opts)
			assert.Len(t, opts, 1)
			return &api.TransactionExtention{
				ConstantResult: [][]byte{{0x01}},
			}, nil
		},
	}

	_, err := New(mc, "TContract").
		Method("deposit()").
		Apply(WithCallValue(1000000)).
		Call(context.Background())

	require.NoError(t, err)
}

// mockSigner implements signer.Signer for testing.
type mockSigner struct{}

func (s *mockSigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	tx.Signature = append(tx.Signature, []byte("fakesig"))
	return tx, nil
}

func (s *mockSigner) Address() address.Address { return nil }

func TestSend(t *testing.T) {
	broadcastCalled := false
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			return newTestTxExt(), nil
		},
		broadcastCtxFunc: func(_ context.Context, tx *core.Transaction) (*api.Return, error) {
			broadcastCalled = true
			assert.NotEmpty(t, tx.Signature)
			return &api.Return{Result: true, Code: 0}, nil
		},
	}

	receipt, err := New(mc, "TContract").
		Method("transfer(address,uint256)").
		From("TFrom").
		Params(`[{"address":"TTo"},{"uint256":"1000"}]`).
		Send(context.Background(), &mockSigner{})
	require.NoError(t, err)
	assert.True(t, broadcastCalled)
	assert.NotEmpty(t, receipt.TxID)
	assert.Empty(t, receipt.Error)
}

func TestSend_BroadcastRejection(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			return newTestTxExt(), nil
		},
		broadcastCtxFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{
				Result:  false,
				Code:    api.Return_CONTRACT_VALIDATE_ERROR,
				Message: []byte("bad contract"),
			}, nil
		},
	}

	receipt, err := New(mc, "TContract").
		Method("transfer(address,uint256)").
		From("TFrom").
		Params(`[{"address":"TTo"},{"uint256":"1000"}]`).
		Send(context.Background(), &mockSigner{})
	require.NoError(t, err)
	assert.Equal(t, "bad contract", receipt.Error)
}

func TestSend_BroadcastNetworkError(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			return newTestTxExt(), nil
		},
		broadcastCtxFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return nil, errors.New("network timeout")
		},
	}

	_, err := New(mc, "TContract").
		Method("test()").
		From("TFrom").
		Send(context.Background(), &mockSigner{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
}

func TestSend_DeferredError(t *testing.T) {
	mc := &mockClient{}

	_, err := New(mc, "TContract").
		SetError(errors.New("invalid address")).
		Send(context.Background(), &mockSigner{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")
}

func TestSendAndConfirm(t *testing.T) {
	callCount := 0
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			return newTestTxExt(), nil
		},
		broadcastCtxFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoByIDCtxFunc: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			callCount++
			if callCount < 2 {
				return &core.TransactionInfo{}, nil
			}
			return &core.TransactionInfo{
				BlockNumber: 99999,
				Fee:         50000,
				Receipt: &core.ResourceReceipt{
					EnergyUsageTotal: 30000,
					NetUsage:         200,
				},
				ContractResult: [][]byte{{0xab}},
			}, nil
		},
	}

	receipt, err := New(mc, "TContract").
		Method("test()").
		From("TFrom").
		SendAndConfirm(context.Background(), &mockSigner{})
	require.NoError(t, err)
	assert.True(t, receipt.Confirmed)
	assert.Equal(t, int64(99999), receipt.BlockNumber)
	assert.Equal(t, int64(50000), receipt.Fee)
	assert.Equal(t, int64(30000), receipt.EnergyUsed)
	assert.Equal(t, int64(200), receipt.BandwidthUsed)
	assert.Equal(t, []byte{0xab}, receipt.Result)
}

func TestSendAndConfirm_ContextCancelled(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			return newTestTxExt(), nil
		},
		broadcastCtxFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: 0}, nil
		},
		getTransactionInfoByIDCtxFunc: func(_ context.Context, _ string) (*core.TransactionInfo, error) {
			return &core.TransactionInfo{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New(mc, "TContract").
		Method("test()").
		From("TFrom").
		SendAndConfirm(ctx, &mockSigner{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// --- Fluent method tests ---

func TestFluentWithPermissionID(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			return newTestTxExt(), nil
		},
	}

	ext, err := New(mc, "TContract").
		Method("test()").
		From("TFrom").
		WithPermissionID(2).
		Build(context.Background())
	require.NoError(t, err)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestFluentWithFeeLimit(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, feeLimit, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(100_000_000), feeLimit)
			return newTestTxExt(), nil
		},
	}

	ext, err := New(mc, "TContract").
		Method("transfer(address,uint256)").
		From("TFrom").
		WithFeeLimit(100_000_000).
		Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestFluentWithCallValue(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, callValue int64, _ string, _ int64) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(1_000_000), callValue)
			return newTestTxExt(), nil
		},
	}

	ext, err := New(mc, "TContract").
		Method("deposit()").
		From("TFrom").
		WithCallValue(1_000_000).
		Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestFluentWithTokenValue(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, _ int64, tokenID string, tokenAmount int64) (*api.TransactionExtention, error) {
			assert.Equal(t, "1000001", tokenID)
			assert.Equal(t, int64(500), tokenAmount)
			return newTestTxExt(), nil
		},
	}

	ext, err := New(mc, "TContract").
		Method("deposit()").
		From("TFrom").
		WithTokenValue("1000001", 500).
		Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, ext)
}

func TestFluentChainAll(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, feeLimit, callValue int64, tokenID string, tokenAmount int64) (*api.TransactionExtention, error) {
			assert.Equal(t, int64(50_000_000), feeLimit)
			assert.Equal(t, int64(1_000_000), callValue)
			assert.Equal(t, "1000001", tokenID)
			assert.Equal(t, int64(100), tokenAmount)
			return newTestTxExt(), nil
		},
	}

	ext, err := New(mc, "TContract").
		Method("deposit()").
		From("TFrom").
		WithFeeLimit(50_000_000).
		WithCallValue(1_000_000).
		WithTokenValue("1000001", 100).
		WithPermissionID(2).
		Build(context.Background())
	require.NoError(t, err)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(2), c.PermissionId)
	}
}

func TestFluentEqualsOption(t *testing.T) {
	mc := &mockClient{
		triggerContractCtxFunc: func(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			return newTestTxExt(), nil
		},
	}

	// Build with functional options
	extOpt, err := New(mc, "TContract").
		Method("test()").
		From("TFrom").
		Apply(WithFeeLimit(50_000_000), WithPermissionID(2)).
		Build(context.Background())
	require.NoError(t, err)

	// Build with fluent methods
	extFluent, err := New(mc, "TContract").
		Method("test()").
		From("TFrom").
		WithFeeLimit(50_000_000).
		WithPermissionID(2).
		Build(context.Background())
	require.NoError(t, err)

	// Serialized RawData must be identical
	rawOpt, err := proto.Marshal(extOpt.Transaction.RawData)
	require.NoError(t, err)
	rawFluent, err := proto.Marshal(extFluent.Transaction.RawData)
	require.NoError(t, err)
	assert.Equal(t, rawOpt, rawFluent,
		"fluent and option APIs must produce identical raw transaction data")
}

func TestFluentWithData_PermissionAndFeeLimit(t *testing.T) {
	mc := &mockClient{
		triggerContractWithDataCtxFunc: func(_ context.Context, _, _ string, data []byte, feeLimit, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
			assert.Equal(t, []byte{0xa9, 0x05, 0x9c, 0xbb}, data[:4]) // transfer selector
			assert.Equal(t, int64(100_000_000), feeLimit)
			return newTestTxExt(), nil
		},
	}

	ext, err := New(mc, "TContract").
		From("TFrom").
		WithData([]byte{0xa9, 0x05, 0x9c, 0xbb, 0x00}).
		WithFeeLimit(100_000_000).
		WithPermissionID(3).
		Build(context.Background())
	require.NoError(t, err)
	for _, c := range ext.Transaction.RawData.Contract {
		assert.Equal(t, int32(3), c.PermissionId)
	}
}
