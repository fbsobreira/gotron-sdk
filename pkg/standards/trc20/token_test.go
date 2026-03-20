package trc20

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/standards/trc20enc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements contract.Client for testing.
// It returns constantResult for constant calls and can optionally return an error.
type mockClient struct {
	constantResult [][]byte
	// err is returned by all methods when set.
	err error
	// results holds sequential results for constant calls. Each Call()
	// pops the first entry. When empty, constantResult is used as fallback.
	results [][][]byte
	// lastData captures the last data argument passed to WithData calls.
	lastData []byte
}

// nextConstantResult returns the next result, cycling through sequential
// results first, then falling back to constantResult.
func (m *mockClient) nextConstantResult() [][]byte {
	if len(m.results) > 0 {
		r := m.results[0]
		m.results = m.results[1:]
		return r
	}
	return m.constantResult
}

func (m *mockClient) TriggerConstantContractCtx(_ context.Context, _, _, _, _ string, _ ...client.ConstantCallOption) (*api.TransactionExtention, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &api.TransactionExtention{
		ConstantResult: m.nextConstantResult(),
		Result:         &api.Return{Result: true},
	}, nil
}

func (m *mockClient) TriggerContractCtx(_ context.Context, _, _, _, _ string, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &api.TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{{}},
			},
		},
		Result: &api.Return{Result: true},
	}, nil
}

func (m *mockClient) TriggerConstantContractWithDataCtx(_ context.Context, _, _ string, data []byte, _ ...client.ConstantCallOption) (*api.TransactionExtention, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.lastData = data
	return &api.TransactionExtention{
		ConstantResult: m.nextConstantResult(),
		Result:         &api.Return{Result: true},
	}, nil
}

func (m *mockClient) TriggerContractWithDataCtx(_ context.Context, _, _ string, data []byte, _, _ int64, _ string, _ int64) (*api.TransactionExtention, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.lastData = data
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
	if m.err != nil {
		return nil, m.err
	}
	return &api.EstimateEnergyMessage{
		Result:         &api.Return{Result: true},
		EnergyRequired: 100000,
	}, nil
}

func (m *mockClient) EstimateEnergyWithDataCtx(_ context.Context, _, _ string, _ []byte, _ int64, _ string, _ int64) (*api.EstimateEnergyMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &api.EstimateEnergyMessage{
		Result:         &api.Return{Result: true},
		EnergyRequired: 100000,
	}, nil
}

func (m *mockClient) BroadcastCtx(_ context.Context, _ *core.Transaction) (*api.Return, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &api.Return{Result: true}, nil
}

func (m *mockClient) GetTransactionInfoByIDCtx(_ context.Context, _ string) (*core.TransactionInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &core.TransactionInfo{}, nil
}

// abiEncodeString encodes a string the way Solidity does (offset + length + data).
// Only supports strings up to 32 bytes (single ABI word).
func abiEncodeString(s string) []byte {
	if len(s) > 32 {
		panic("abiEncodeString: test helper only supports strings <= 32 bytes")
	}
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

// --- Existing tests (kept as-is) ---

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

// --- New tests for encode.go ---

func TestEncodeWithAddress(t *testing.T) {
	addr, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)

	result := encodeWithAddress(trc20enc.SelectorBalanceOf, addr)

	// 4-byte selector + 32-byte padded address = 36 bytes
	assert.Len(t, result, 36)

	// Verify selector prefix
	assert.Equal(t, trc20enc.SelectorBalanceOf, hex.EncodeToString(result[:4]))

	// Verify the 20-byte EVM address is at the right offset (bytes 16..36)
	// First 12 bytes of padding should be zero
	for i := 4; i < 16; i++ {
		assert.Equal(t, byte(0), result[i], "expected zero padding at byte %d", i)
	}
	// The EVM part should match addr[1:] (skip 0x41 prefix)
	assert.Equal(t, []byte(addr[1:]), result[16:36])
}

func TestEncodeTransferFrom(t *testing.T) {
	from, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	to, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)
	amount := big.NewInt(1_000_000)

	result, err := encodeTransferFrom(from, to, amount)
	require.NoError(t, err)

	// 4-byte selector + 32-byte from + 32-byte to + 32-byte amount = 100 bytes
	assert.Len(t, result, 100)

	// Verify selector is transferFrom
	assert.Equal(t, trc20enc.SelectorTransferFrom, hex.EncodeToString(result[:4]))

	// Verify from address (bytes 4..36)
	assert.Equal(t, []byte(from[1:]), result[16:36])

	// Verify to address (bytes 36..68)
	assert.Equal(t, []byte(to[1:]), result[48:68])

	// Verify amount (bytes 68..100)
	decoded := new(big.Int).SetBytes(result[68:100])
	assert.Equal(t, amount, decoded)
}

func TestEncodeTransferFromNilAmount(t *testing.T) {
	from, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	to, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)

	_, err = encodeTransferFrom(from, to, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil value")
}

func TestEncodeTransferFromNegativeAmount(t *testing.T) {
	from, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	to, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)

	_, err = encodeTransferFrom(from, to, big.NewInt(-1))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "negative")
}

func TestPadUint256(t *testing.T) {
	tests := []struct {
		name    string
		input   *big.Int
		wantErr string
	}{
		{
			name:  "zero",
			input: big.NewInt(0),
		},
		{
			name:  "one",
			input: big.NewInt(1),
		},
		{
			name:  "max uint256",
			input: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)),
		},
		{
			name:    "nil",
			input:   nil,
			wantErr: "nil value",
		},
		{
			name:    "negative",
			input:   big.NewInt(-1),
			wantErr: "negative",
		},
		{
			name:    "exceeds 256 bits",
			input:   new(big.Int).Lsh(big.NewInt(1), 257),
			wantErr: "exceeds 256 bits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := padUint256(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, 32)

			// Round-trip: decoded value should equal input
			decoded := new(big.Int).SetBytes(result)
			assert.Equal(t, 0, tt.input.Cmp(decoded), "round-trip mismatch: want %s got %s", tt.input, decoded)
		})
	}
}

func TestPadUint256RoundTrip(t *testing.T) {
	// Test encoding then decoding through decodeUint256
	values := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(1_000_000),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)), // max uint256
	}

	for _, v := range values {
		padded, err := padUint256(v)
		require.NoError(t, err)

		decoded, err := decodeUint256([][]byte{padded})
		require.NoError(t, err)
		assert.Equal(t, 0, v.Cmp(decoded), "round-trip failed for %s", v)
	}
}

func TestDecodeUint256Errors(t *testing.T) {
	tests := []struct {
		name  string
		input [][]byte
	}{
		{
			name:  "nil results",
			input: nil,
		},
		{
			name:  "empty results",
			input: [][]byte{},
		},
		{
			name:  "short data",
			input: [][]byte{make([]byte, 31)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeUint256(tt.input)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "insufficient data")
		})
	}
}

func TestDecodeUint256LargeValue(t *testing.T) {
	// Encode max uint256 and decode it
	maxUint256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	encoded := abiEncodeUint256(maxUint256)
	decoded, err := decodeUint256([][]byte{encoded})
	require.NoError(t, err)
	assert.Equal(t, 0, maxUint256.Cmp(decoded))
}

func TestDecodeUint256Zero(t *testing.T) {
	encoded := abiEncodeUint256(big.NewInt(0))
	decoded, err := decodeUint256([][]byte{encoded})
	require.NoError(t, err)
	assert.Equal(t, int64(0), decoded.Int64())
}

func TestDecodeStringEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   [][]byte
		want    string
		wantErr bool
	}{
		{
			name:    "nil outer slice",
			input:   nil,
			wantErr: true,
		},
		{
			name:    "empty inner byte slice",
			input:   [][]byte{{}},
			wantErr: true,
		},
		{
			name: "non-UTF8 32-byte data",
			input: func() [][]byte {
				b := make([]byte, 32)
				// Fill with invalid UTF-8 sequences
				b[0] = 0xFF
				b[1] = 0xFE
				return [][]byte{b}
			}(),
			wantErr: true,
		},
		{
			name: "32-byte all zeros",
			input: func() [][]byte {
				b := make([]byte, 32)
				return [][]byte{b}
			}(),
			wantErr: true,
		},
		{
			name:  "long token name",
			input: [][]byte{abiEncodeString("Wrapped Bitcoin on TRON Network")},
			want:  "Wrapped Bitcoin on TRON Network",
		},
		{
			name:  "single character",
			input: [][]byte{abiEncodeString("X")},
			want:  "X",
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

func TestPadAddressShortAddress(t *testing.T) {
	// Non-standard length address should produce 32 zero bytes
	short := address.Address([]byte{0x41, 0x01, 0x02})
	padded := padAddress(short)
	assert.Len(t, padded, 32)
	// All zeros because len != AddressLength
	for i := 0; i < 32; i++ {
		assert.Equal(t, byte(0), padded[i], "expected zero at byte %d", i)
	}
}

func TestPadAddressEmpty(t *testing.T) {
	padded := padAddress(address.Address(nil))
	assert.Len(t, padded, 32)
	for i := 0; i < 32; i++ {
		assert.Equal(t, byte(0), padded[i])
	}
}

func TestEncodeWithTwoAddresses(t *testing.T) {
	addr1, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	addr2, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)

	result := encodeWithTwoAddresses(trc20enc.SelectorAllowance, addr1, addr2)

	// 4-byte selector + 32-byte addr1 + 32-byte addr2 = 68 bytes
	assert.Len(t, result, 68)

	// Verify selector
	assert.Equal(t, trc20enc.SelectorAllowance, hex.EncodeToString(result[:4]))

	// Verify first address EVM part
	assert.Equal(t, []byte(addr1[1:]), result[16:36])

	// Verify second address EVM part
	assert.Equal(t, []byte(addr2[1:]), result[48:68])
}

func TestEncodeTransfer(t *testing.T) {
	addr, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)
	amount := big.NewInt(50_000_000)

	result, err := encodeTransfer(trc20enc.SelectorTransfer, addr, amount)
	require.NoError(t, err)

	// 4-byte selector + 32-byte address + 32-byte amount = 68 bytes
	assert.Len(t, result, 68)
	assert.Equal(t, trc20enc.SelectorTransfer, hex.EncodeToString(result[:4]))
	assert.Equal(t, []byte(addr[1:]), result[16:36])

	decoded := new(big.Int).SetBytes(result[36:68])
	assert.Equal(t, amount, decoded)
}

func TestEncodeTransferNilAmount(t *testing.T) {
	addr, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)

	_, err = encodeTransfer(trc20enc.SelectorTransfer, addr, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil value")
}

// --- New tests for token.go ---

func TestTokenSymbolError(t *testing.T) {
	mc := &mockClient{
		err: errors.New("connection refused"),
	}
	token := New(mc, "TContractAddr")

	_, err := token.Symbol(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestTokenTotalSupplyError(t *testing.T) {
	mc := &mockClient{
		err: errors.New("rpc error"),
	}
	token := New(mc, "TContractAddr")

	_, err := token.TotalSupply(context.Background())
	assert.Error(t, err)
}

func TestTokenBalanceOf(t *testing.T) {
	balance := big.NewInt(1_500_000) // 1.5 USDT with 6 decimals
	mc := &mockClient{
		results: [][][]byte{
			// First call: balanceOf
			{abiEncodeUint256(balance)},
			// Second call: decimals
			{abiEncodeUint256(big.NewInt(6))},
			// Third call: symbol
			{abiEncodeString("USDT")},
		},
	}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	result, err := token.BalanceOf(context.Background(), "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	assert.Equal(t, balance, result.Raw)
	assert.Equal(t, "USDT", result.Symbol)
	assert.Equal(t, "1.5", result.Display)
}

func TestTokenBalanceOfInvalidAddress(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	_, err := token.BalanceOf(context.Background(), "INVALID_ADDR")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address")
}

func TestTokenBalanceOfClientError(t *testing.T) {
	mc := &mockClient{
		err: errors.New("timeout"),
	}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	_, err := token.BalanceOf(context.Background(), "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	assert.Error(t, err)
}

func TestTokenInfo(t *testing.T) {
	mc := &mockClient{
		results: [][][]byte{
			// Name
			{abiEncodeString("Tether USD")},
			// Symbol
			{abiEncodeString("USDT")},
			// Decimals
			{abiEncodeUint256(big.NewInt(6))},
			// TotalSupply
			{abiEncodeUint256(big.NewInt(1_000_000_000_000))},
		},
	}
	token := New(mc, "TContractAddr")

	info, err := token.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Tether USD", info.Name)
	assert.Equal(t, "USDT", info.Symbol)
	assert.Equal(t, uint8(6), info.Decimals)
	assert.Equal(t, big.NewInt(1_000_000_000_000), info.TotalSupply)
}

func TestTokenInfoNameError(t *testing.T) {
	mc := &mockClient{
		err: errors.New("name failed"),
	}
	token := New(mc, "TContractAddr")

	_, err := token.Info(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name:")
}

func TestTokenInfoSymbolError(t *testing.T) {
	mc := &mockClient{
		results: [][][]byte{
			// Name succeeds
			{abiEncodeString("Tether USD")},
		},
		// Then all subsequent calls fail
		err: nil,
	}
	token := New(mc, "TContractAddr")

	// After the first result is consumed, the fallback constantResult is nil,
	// which will cause decodeString to fail on Symbol.
	_, err := token.Info(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symbol:")
}

func TestTokenDecimalsOutOfRange(t *testing.T) {
	// Value > 255 should fail the uint8 range check
	mc := &mockClient{
		constantResult: [][]byte{abiEncodeUint256(big.NewInt(256))},
	}
	token := New(mc, "TContractAddr")

	_, err := token.Decimals(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of uint8 range")
}

func TestTokenDecimalsClientError(t *testing.T) {
	mc := &mockClient{
		err: errors.New("network error"),
	}
	token := New(mc, "TContractAddr")

	_, err := token.Decimals(context.Background())
	assert.Error(t, err)
}

func TestTokenNameError(t *testing.T) {
	mc := &mockClient{
		err: errors.New("name query failed"),
	}
	token := New(mc, "TContractAddr")

	_, err := token.Name(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name query failed")
}

func TestTokenApproveValidAddress(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.Approve(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
		big.NewInt(1_000_000),
	)
	assert.NotNil(t, call)
	assert.Nil(t, call.Err())

	tx, err := call.Build(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tx)

	// Verify the encoded call data starts with the approve selector
	require.NotEmpty(t, mc.lastData)
	assert.Equal(t, trc20enc.SelectorApprove, hex.EncodeToString(mc.lastData[:4]),
		"call data should start with approve selector")
}

func TestTokenApproveNilAmount(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.Approve(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
		nil,
	)
	assert.NotNil(t, call)
	// Error is deferred
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil value")
}

func TestTokenTransferFromValidAddresses(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.TransferFrom(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1", // caller
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1", // from
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9", // to
		big.NewInt(1_000_000),
	)
	assert.NotNil(t, call)
	assert.Nil(t, call.Err())

	// Verify the encoded call data starts with the transferFrom selector
	tx, err := call.Build(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, tx)
	require.NotEmpty(t, mc.lastData)
	assert.Equal(t, trc20enc.SelectorTransferFrom, hex.EncodeToString(mc.lastData[:4]),
		"call data should start with transferFrom selector")
}

func TestTokenTransferFromInvalidToAddress(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.TransferFrom(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"INVALID_TO",
		big.NewInt(1_000_000),
	)
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to address")
}

func TestTokenTransferFromNilAmount(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.TransferFrom(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
		nil,
	)
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil value")
}

func TestTokenTransferNilAmount(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.Transfer(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
		nil,
	)
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil value")
}

func TestTokenAllowanceInvalidOwner(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	_, err := token.Allowance(context.Background(), "INVALID_OWNER", "TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid owner address")
}

func TestTokenAllowanceInvalidSpender(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	_, err := token.Allowance(context.Background(), "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1", "INVALID_SPENDER")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid spender address")
}

func TestTokenAllowanceClientError(t *testing.T) {
	mc := &mockClient{
		err: errors.New("rpc unavailable"),
	}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	_, err := token.Allowance(context.Background(), "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1", "TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	assert.Error(t, err)
}

func TestFormatBalanceNil(t *testing.T) {
	assert.Equal(t, "0", formatBalance(nil, 6))
}

func TestFormatBalanceZeroDecimals(t *testing.T) {
	assert.Equal(t, "1,000", formatBalance(big.NewInt(1000), 0))
}

func TestFormatBalanceExactWhole(t *testing.T) {
	// 2,000,000 with 6 decimals = 2.0 (no fraction) -> "2"
	assert.Equal(t, "2", formatBalance(big.NewInt(2_000_000), 6))
}

func TestFormatBalanceLeadingZeroFraction(t *testing.T) {
	// 1,000,001 with 6 decimals = 1.000001
	assert.Equal(t, "1.000001", formatBalance(big.NewInt(1_000_001), 6))
}

func TestTokenBalanceOfDecimalsError(t *testing.T) {
	balance := big.NewInt(1_000_000)
	mc := &mockClient{
		results: [][][]byte{
			// balanceOf succeeds
			{abiEncodeUint256(balance)},
			// decimals returns empty (will fail)
			{},
		},
	}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	_, err := token.BalanceOf(context.Background(), "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching decimals")
}

func TestTokenBalanceOfSymbolError(t *testing.T) {
	balance := big.NewInt(1_000_000)
	mc := &mockClient{
		results: [][][]byte{
			// balanceOf succeeds
			{abiEncodeUint256(balance)},
			// decimals succeeds
			{abiEncodeUint256(big.NewInt(6))},
			// symbol returns empty (will fail)
			{},
		},
	}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	_, err := token.BalanceOf(context.Background(), "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fetching symbol")
}

func TestTokenInfoDecimalsError(t *testing.T) {
	mc := &mockClient{
		results: [][][]byte{
			// Name succeeds
			{abiEncodeString("Tether USD")},
			// Symbol succeeds
			{abiEncodeString("USDT")},
			// Decimals returns empty (will fail)
			{},
		},
	}
	token := New(mc, "TContractAddr")

	_, err := token.Info(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decimals:")
}

func TestTokenInfoTotalSupplyError(t *testing.T) {
	mc := &mockClient{
		results: [][][]byte{
			// Name succeeds
			{abiEncodeString("Tether USD")},
			// Symbol succeeds
			{abiEncodeString("USDT")},
			// Decimals succeeds
			{abiEncodeUint256(big.NewInt(6))},
			// TotalSupply returns empty (will fail)
			{},
		},
	}
	token := New(mc, "TContractAddr")

	_, err := token.Info(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "totalSupply:")
}

func TestEncodeTransferZeroAmount(t *testing.T) {
	addr, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)

	result, err := encodeTransfer(trc20enc.SelectorTransfer, addr, big.NewInt(0))
	require.NoError(t, err)
	assert.Len(t, result, 68)

	// Amount should be 32 zero bytes
	decoded := new(big.Int).SetBytes(result[36:68])
	assert.Equal(t, int64(0), decoded.Int64())
}

func TestEncodeTransferFromZeroAmount(t *testing.T) {
	from, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	to, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)

	result, err := encodeTransferFrom(from, to, big.NewInt(0))
	require.NoError(t, err)
	assert.Len(t, result, 100)
}

func TestTokenTransferFromNegativeAmount(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.TransferFrom(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
		big.NewInt(-100),
	)
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "negative")
}

func TestTokenApproveNegativeAmount(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.Approve(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
		big.NewInt(-1),
	)
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "negative")
}

func TestTokenTransferNegativeAmount(t *testing.T) {
	mc := &mockClient{}
	token := New(mc, "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")

	call := token.Transfer(
		"TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1",
		"TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9",
		big.NewInt(-500),
	)
	_, err := call.Build(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "negative")
}

func TestDecodeStringTruncatedABI(t *testing.T) {
	// Create data that looks like ABI but has truncated data field
	buf := make([]byte, 80) // shorter than offset+length would imply
	// offset = 32
	buf[31] = 0x20
	// length = 100 (but only 16 bytes of data available)
	big.NewInt(100).FillBytes(buf[32:64])
	copy(buf[64:], []byte("short"))

	// ABI length check fails (64+100 > 80), so it falls back to fixed 32-byte parse.
	// The first 32 bytes are mostly zeros with 0x20 at byte 31.
	// After null-termination at byte 0, the result is empty, which triggers an error.
	_, err := decodeString([][]byte{buf})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot decode string")
}
