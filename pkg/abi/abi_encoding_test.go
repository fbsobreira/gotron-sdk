package abi

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const transferSelector = "a9059cbb"

func TestSignature_StripsWhitespace(t *testing.T) {
	canonical := Signature("transfer(address,uint256)")
	require.Equal(t, transferSelector, hex.EncodeToString(canonical))

	tests := []string{
		"transfer(address, uint256)",
		"transfer(address,uint256) ",
		" transfer(address,uint256)",
		"transfer(address,  uint256)",
		"transfer( address,uint256 )",
		"transfer(address,\tuint256)",
		"transfer(address,\nuint256)",
	}
	for _, method := range tests {
		t.Run(method, func(t *testing.T) {
			got := Signature(method)
			assert.Equal(t, transferSelector, hex.EncodeToString(got),
				"whitespace must not change the selector")
		})
	}
}

func TestPack_WhitespaceSignatureMatchesCanonical(t *testing.T) {
	params := []Param{
		{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},
		{"uint256": "1000000"},
	}

	canonical, err := Pack("transfer(address,uint256)", params)
	require.NoError(t, err)

	spaced, err := Pack("transfer(address, uint256)", params)
	require.NoError(t, err)

	assert.Equal(t, hex.EncodeToString(canonical), hex.EncodeToString(spaced))
	assert.Equal(t, transferSelector, hex.EncodeToString(spaced[:4]))
}

func TestConvertToInt_OutOfRange(t *testing.T) {
	uint256Max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
	uint256Overflow := new(big.Int).Add(uint256Max, big.NewInt(1))
	int256Min := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 255))
	int256Overflow := new(big.Int).Lsh(big.NewInt(1), 255)

	tests := []struct {
		name     string
		typeName string
		input    interface{}
	}{
		{name: "negative uint256", typeName: "uint256", input: "-1"},
		{name: "negative uint8", typeName: "uint8", input: "-1"},
		{name: "uint256 overflow", typeName: "uint256", input: uint256Overflow.String()},
		{name: "int256 overflow", typeName: "int256", input: int256Overflow.String()},
		{name: "uint24 overflow", typeName: "uint24", input: "16777216"},
		{name: "int24 overflow", typeName: "int24", input: "8388608"},
		{name: "int24 underflow", typeName: "int24", input: "-8388609"},
		{name: "uint128 overflow", typeName: "uint128", input: new(big.Int).Lsh(big.NewInt(1), 128).String()},
		{name: "native big.Int overflow uint8", typeName: "uint8", input: big.NewInt(256)},
		{name: "native negative uint256", typeName: "uint256", input: big.NewInt(-1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ty, err := eABI.NewType(tt.typeName, "", nil)
			require.NoError(t, err)

			_, err = convertToInt(ty, tt.input)
			require.Error(t, err, "out-of-range %s must be rejected", tt.typeName)
		})
	}

	t.Run("uint256 max accepted", func(t *testing.T) {
		ty, err := eABI.NewType("uint256", "", nil)
		require.NoError(t, err)
		got, err := convertToInt(ty, uint256Max.String())
		require.NoError(t, err)
		n, ok := got.(*big.Int)
		require.True(t, ok)
		assert.Equal(t, 0, n.Cmp(uint256Max))
	})

	t.Run("int256 min accepted", func(t *testing.T) {
		ty, err := eABI.NewType("int256", "", nil)
		require.NoError(t, err)
		got, err := convertToInt(ty, int256Min.String())
		require.NoError(t, err)
		n, ok := got.(*big.Int)
		require.True(t, ok)
		assert.Equal(t, 0, n.Cmp(int256Min))
	})
}

func TestGetPaddedParam_IntOutOfRange(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		value    interface{}
	}{
		{name: "negative uint256", typeName: "uint256", value: "-1"},
		{name: "uint256 overflow", typeName: "uint256", value: new(big.Int).Lsh(big.NewInt(1), 256).String()},
		{name: "uint24 overflow", typeName: "uint24", value: "16777216"},
		{name: "uint256[] negative element", typeName: "uint256[]", value: []string{"1", "-1"}},
		{name: "uint24[] overflow element", typeName: "uint24[]", value: []string{"16777216"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetPaddedParam([]Param{{tt.typeName: tt.value}})
			require.Error(t, err)
		})
	}
}

func TestConvertToBytes_AllFixedSizes(t *testing.T) {
	for size := 1; size <= 32; size++ {
		t.Run(fmt.Sprintf("bytes%d", size), func(t *testing.T) {
			ty, err := eABI.NewType(fmt.Sprintf("bytes%d", size), "", nil)
			require.NoError(t, err)

			raw := make([]byte, size)
			for i := range raw {
				raw[i] = byte(i + 1)
			}
			got, err := convertToBytes(ty, hex.EncodeToString(raw))
			require.NoError(t, err)

			arr, ok := asByteArray(got, size)
			require.True(t, ok, "bytes%d must encode as [%d]byte, got %T", size, size, got)
			assert.Equal(t, raw, arr)
		})
	}
}

func TestGetPaddedParam_Bytes4AndBytes20(t *testing.T) {
	t.Run("bytes4", func(t *testing.T) {
		b, err := GetPaddedParam([]Param{{"bytes4": "a9059cbb"}})
		require.NoError(t, err)
		require.Len(t, b, 32)
		assert.Equal(t, "a9059cbb"+strings.Repeat("00", 28), hex.EncodeToString(b))
	})

	t.Run("bytes20", func(t *testing.T) {
		payload := strings.Repeat("ab", 20)
		b, err := GetPaddedParam([]Param{{"bytes20": payload}})
		require.NoError(t, err)
		require.Len(t, b, 32)
		assert.Equal(t, payload+strings.Repeat("00", 12), hex.EncodeToString(b))
	})
}

func TestLoadFromJSON_TypedJSONNumbers(t *testing.T) {
	params, err := LoadFromJSON(`[{"uint256": 1000}]`)
	require.NoError(t, err)
	require.Len(t, params, 1)

	b, err := GetPaddedParam(params)
	require.NoError(t, err)
	require.Len(t, b, 32)

	quoted, err := GetPaddedParam([]Param{{"uint256": "1000"}})
	require.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(quoted), hex.EncodeToString(b))
}

func TestLoadFromJSON_TypedJSONNumberPrecision(t *testing.T) {
	large := "100000000000000000000"
	params, err := LoadFromJSON(`[{"uint256": ` + large + `}]`)
	require.NoError(t, err)
	require.Len(t, params, 1)

	val, ok := params[0]["uint256"].(string)
	require.True(t, ok, "typed JSON numbers must be normalized to string, got %T", params[0]["uint256"])
	assert.Equal(t, large, val)

	b, err := GetPaddedParam(params)
	require.NoError(t, err)

	quoted, err := GetPaddedParam([]Param{{"uint256": large}})
	require.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(quoted), hex.EncodeToString(b))
}

func TestLoadFromJSON_TypedJSONNumberArray(t *testing.T) {
	params, err := LoadFromJSON(`[{"uint256[]": [100, 200]}]`)
	require.NoError(t, err)

	b, err := GetPaddedParam(params)
	require.NoError(t, err)

	quoted, err := GetPaddedParam([]Param{{"uint256[]": []string{"100", "200"}}})
	require.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(quoted), hex.EncodeToString(b))
}

func asByteArray(v interface{}, size int) ([]byte, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Array || rv.Len() != size || rv.Type().Elem().Kind() != reflect.Uint8 {
		return nil, false
	}
	out := make([]byte, size)
	reflect.Copy(reflect.ValueOf(out), rv)
	return out, true
}
