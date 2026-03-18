package trc20

import (
	"context"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements contract.Client for testing.
type mockClient struct {
	constantResult [][]byte
}

func (m *mockClient) TriggerConstantContractCtx(_ context.Context, _, _, _, _ string, _ ...client.ConstantCallOption) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{
		ConstantResult: m.constantResult,
		Result:         &api.Return{Result: true},
	}, nil
}

func (m *mockClient) TriggerContractCtx(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{{}},
			},
		},
		Result: &api.Return{Result: true},
	}, nil
}

func (m *mockClient) TriggerConstantContractWithDataCtx(_ context.Context, _, _ string, _ []byte, _ ...client.ConstantCallOption) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{
		ConstantResult: m.constantResult,
		Result:         &api.Return{Result: true},
	}, nil
}

func (m *mockClient) TriggerContractWithDataCtx(_ context.Context, _, _ string, _ []byte, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{{}},
			},
		},
		Result: &api.Return{Result: true},
	}, nil
}

func (m *mockClient) EstimateEnergyCtx(_ context.Context, _, _, _, _ string, _ int64, _ string, _ int64) (*api.EstimateEnergyMessage, error) {
	return &api.EstimateEnergyMessage{
		Result:         &api.Return{Result: true},
		EnergyRequired: 100000,
	}, nil
}

func (m *mockClient) BroadcastCtx(_ context.Context, _ *core.Transaction) (*api.Return, error) {
	return &api.Return{Result: true}, nil
}

func (m *mockClient) GetTransactionInfoByIDCtx(_ context.Context, _ string) (*core.TransactionInfo, error) {
	return &core.TransactionInfo{}, nil
}

// abiEncodeString encodes a string the way Solidity does (offset + length + data).
func abiEncodeString(s string) []byte {
	buf := make([]byte, 96)
	// offset = 32
	buf[31] = 0x20
	// length
	big.NewInt(int64(len(s))).FillBytes(buf[32:64])
	// data
	copy(buf[64:], []byte(s))
	return buf
}

// abiEncodeUint256 encodes a uint256 as 32 bytes.
func abiEncodeUint256(n *big.Int) []byte {
	buf := make([]byte, 32)
	b := n.Bytes()
	copy(buf[32-len(b):], b)
	return buf
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		name    string
		input   [][]byte
		want    string
		wantErr bool
	}{
		{
			name:  "standard ABI encoding",
			input: [][]byte{abiEncodeString("Tether USD")},
			want:  "Tether USD",
		},
		{
			name: "fixed 32-byte UTF-8",
			input: func() [][]byte {
				b := make([]byte, 32)
				copy(b, "USDT")
				return [][]byte{b}
			}(),
			want: "USDT",
		},
		{
			name:    "empty result",
			input:   [][]byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeUint256(t *testing.T) {
	input := abiEncodeUint256(big.NewInt(6))
	got, err := decodeUint256([][]byte{input})
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(6), got)
}

func TestFormatBalance(t *testing.T) {
	tests := []struct {
		name     string
		raw      *big.Int
		decimals uint8
		want     string
	}{
		{"zero", big.NewInt(0), 6, "0"},
		{"1 USDT", big.NewInt(1_000_000), 6, "1"},
		{"1000.50 USDT", big.NewInt(1_000_500_000), 6, "1,000.5"},
		{"0.000001", big.NewInt(1), 6, "0.000001"},
		{"large", new(big.Int).Mul(big.NewInt(1_000_000), big.NewInt(1_000_000_000)), 6, "1,000,000,000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatBalance(tt.raw, tt.decimals))
		})
	}
}

func TestTokenName(t *testing.T) {
	mc := &mockClient{
		constantResult: [][]byte{abiEncodeString("Tether USD")},
	}
	token := New(mc, "TContractAddr")

	name, err := token.Name(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Tether USD", name)
}

func TestTokenDecimals(t *testing.T) {
	mc := &mockClient{
		constantResult: [][]byte{abiEncodeUint256(big.NewInt(6))},
	}
	token := New(mc, "TContractAddr")

	decimals, err := token.Decimals(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint8(6), decimals)
}

func TestTokenTransferReturnsBuilder(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.Transfer(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
		big.NewInt(1_000_000),
	)
	assert.NotNil(t, call)
}

func TestTokenTransferInvalidAddress(t *testing.T) {
	mc := &mockClient{
		constantResult: [][]byte{abiEncodeUint256(big.NewInt(6))},
	}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	// Invalid address is deferred — returns a builder, error surfaces at Send/Build
	call := token.Transfer(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"INVALID_ADDRESS",
		big.NewInt(1_000_000),
	)
	assert.NotNil(t, call)

	// Error surfaces at terminal call
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to address")
}

func TestTokenApproveInvalidAddress(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.Approve(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"INVALID",
		big.NewInt(1_000_000),
	)
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid spender address")
}

func TestTokenTransferFromInvalidAddress(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.TransferFrom(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"INVALID",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
		big.NewInt(1_000_000),
	)
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from address")
}

func TestTokenAllowance(t *testing.T) {
	mc := &mockClient{
		constantResult: [][]byte{abiEncodeUint256(big.NewInt(5_000_000))},
	}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	allowance, err := token.Allowance(context.Background(),
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
	)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(5_000_000), allowance)
}

func TestPadAddress(t *testing.T) {
	// 21-byte TRON address (0x41 prefix + 20 bytes)
	addr := make([]byte, 21)
	addr[0] = 0x41
	for i := 1; i < 21; i++ {
		addr[i] = byte(i)
	}

	padded := padAddress(addr)
	assert.Len(t, padded, 32)
	// First 12 bytes should be zero
	for i := 0; i < 12; i++ {
		assert.Equal(t, byte(0), padded[i])
	}
	// Next 20 bytes should be the EVM address (without 0x41)
	for i := 0; i < 20; i++ {
		assert.Equal(t, byte(i+1), padded[12+i])
	}
}

func TestAddThousandSeparators(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0", "0"},
		{"100", "100"},
		{"1000", "1,000"},
		{"1000000", "1,000,000"},
		{"12345678", "12,345,678"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, addThousandSeparators(tt.input))
		})
	}
}
