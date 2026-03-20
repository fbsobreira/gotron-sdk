package trc20enc

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectorBytes(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		wantHex  string
	}{
		{"Name", SelectorName, "06fdde03"},
		{"Symbol", SelectorSymbol, "95d89b41"},
		{"Decimals", SelectorDecimals, "313ce567"},
		{"TotalSupply", SelectorTotalSupply, "18160ddd"},
		{"BalanceOf", SelectorBalanceOf, "70a08231"},
		{"Transfer", SelectorTransfer, "a9059cbb"},
		{"Approve", SelectorApprove, "095ea7b3"},
		{"TransferFrom", SelectorTransferFrom, "23b872dd"},
		{"Allowance", SelectorAllowance, "dd62ed3e"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := SelectorBytes(tt.selector)
			require.NotNil(t, b)
			assert.Equal(t, tt.wantHex, hex.EncodeToString(b))
		})
	}
}

func TestSelectorBytesUnknown(t *testing.T) {
	assert.Nil(t, SelectorBytes("deadbeef"))
}

func TestPadAddress(t *testing.T) {
	addr := make([]byte, 21)
	addr[0] = 0x41
	for i := 1; i < 21; i++ {
		addr[i] = byte(i)
	}

	padded := PadAddress(addr)
	assert.Len(t, padded, 32)
	for i := 0; i < 12; i++ {
		assert.Equal(t, byte(0), padded[i])
	}
	for i := 0; i < 20; i++ {
		assert.Equal(t, byte(i+1), padded[12+i])
	}
}

func TestPadUint256(t *testing.T) {
	tests := []struct {
		name    string
		input   *big.Int
		wantErr string
	}{
		{"zero", big.NewInt(0), ""},
		{"positive", big.NewInt(42), ""},
		{"nil", nil, "nil value"},
		{"negative", big.NewInt(-1), "negative"},
		{"exceeds 256 bits", new(big.Int).Lsh(big.NewInt(1), 257), "exceeds 256 bits"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := PadUint256(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Len(t, result, 32)
		})
	}
}

func TestDecodeUint256(t *testing.T) {
	buf := make([]byte, 32)
	big.NewInt(42).FillBytes(buf)

	n, err := DecodeUint256(buf)
	require.NoError(t, err)
	assert.Equal(t, int64(42), n.Int64())
}

func TestDecodeUint256Short(t *testing.T) {
	_, err := DecodeUint256(make([]byte, 31))
	assert.Error(t, err)
}

func TestDecodeString(t *testing.T) {
	// ABI-encoded "USDT"
	buf := make([]byte, 96)
	buf[31] = 0x20 // offset = 32
	big.NewInt(4).FillBytes(buf[32:64])
	copy(buf[64:], []byte("USDT"))

	s, err := DecodeString(buf)
	require.NoError(t, err)
	assert.Equal(t, "USDT", s)
}

func TestDecodeStringEmpty(t *testing.T) {
	_, err := DecodeString(nil)
	assert.Error(t, err)
}

func TestEncodeBalanceOf(t *testing.T) {
	addr, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)

	result := EncodeBalanceOf(addr)
	assert.NotEmpty(t, result)
	// Should start with the balanceOf selector
	assert.Equal(t, SelectorBalanceOf, result[:8])
}

func TestEncodeTransferCall(t *testing.T) {
	addr, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)

	result, err := EncodeTransferCall(addr, big.NewInt(1000))
	require.NoError(t, err)
	assert.Equal(t, SelectorTransfer, result[:8])
}

func TestEncodeApproveCall(t *testing.T) {
	addr, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)

	result, err := EncodeApproveCall(addr, big.NewInt(500))
	require.NoError(t, err)
	assert.Equal(t, SelectorApprove, result[:8])
}

func TestEncodeTransferFromCall(t *testing.T) {
	from, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	to, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)

	result, err := EncodeTransferFromCall(from, to, big.NewInt(100))
	require.NoError(t, err)
	assert.Equal(t, SelectorTransferFrom, result[:8])
}

func TestEncodeTransferFromCallError(t *testing.T) {
	from, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	to, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)

	_, err = EncodeTransferFromCall(from, to, nil)
	assert.Error(t, err)
}

func TestDecodeUint256Results(t *testing.T) {
	buf := make([]byte, 32)
	big.NewInt(99).FillBytes(buf)

	n, err := DecodeUint256Results([][]byte{buf})
	require.NoError(t, err)
	assert.Equal(t, int64(99), n.Int64())
}

func TestDecodeUint256ResultsEmpty(t *testing.T) {
	_, err := DecodeUint256Results(nil)
	assert.Error(t, err)
}

func TestDecodeStringResults(t *testing.T) {
	buf := make([]byte, 96)
	buf[31] = 0x20
	big.NewInt(3).FillBytes(buf[32:64])
	copy(buf[64:], []byte("ABC"))

	s, err := DecodeStringResults([][]byte{buf})
	require.NoError(t, err)
	assert.Equal(t, "ABC", s)
}

func TestDecodeStringResultsEmpty(t *testing.T) {
	_, err := DecodeStringResults(nil)
	assert.Error(t, err)

	_, err = DecodeStringResults([][]byte{{}})
	assert.Error(t, err)
}

func TestEncodeWithTwoAddresses(t *testing.T) {
	addr1, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)
	addr2, err := address.Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	require.NoError(t, err)

	sel := SelectorBytes(SelectorAllowance)
	result := EncodeWithTwoAddresses(sel, addr1, addr2)
	assert.Len(t, result, 4+32+32)
}

func TestEncodeAddressAmountError(t *testing.T) {
	addr, err := address.Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	require.NoError(t, err)

	_, err = EncodeAddressAmount(SelectorBytes(SelectorTransfer), addr, nil)
	assert.Error(t, err)
}
