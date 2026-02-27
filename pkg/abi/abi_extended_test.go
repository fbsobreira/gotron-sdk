package abi

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// LoadFromJSON
// ---------------------------------------------------------------------------

func TestLoadFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		wantErr bool
	}{
		{
			name:    "empty string returns nil",
			input:   "",
			wantLen: 0,
			wantErr: false,
		},
		{
			name:    "valid single param",
			input:   `[{"uint256":"100"}]`,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "valid multiple params",
			input:   `[{"address":"TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},{"uint256":"100"}]`,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{not valid json`,
			wantErr: true,
		},
		{
			name:    "wrong JSON shape - object instead of array",
			input:   `{"uint256":"100"}`,
			wantErr: true,
		},
		{
			name:    "empty array",
			input:   `[]`,
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := LoadFromJSON(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.input == "" {
				assert.Nil(t, result)
			} else {
				assert.Len(t, result, tc.wantLen)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Signature
// ---------------------------------------------------------------------------

func TestSignature(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		wantHex string
		wantLen int
	}{
		{
			name:    "transfer(address,uint256)",
			method:  "transfer(address,uint256)",
			wantHex: "a9059cbb",
			wantLen: 4,
		},
		{
			name:    "approve(address,uint256)",
			method:  "approve(address,uint256)",
			wantHex: "095ea7b3",
			wantLen: 4,
		},
		{
			name:    "balanceOf(address)",
			method:  "balanceOf(address)",
			wantHex: "70a08231",
			wantLen: 4,
		},
		{
			name:    "empty method",
			method:  "",
			wantLen: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sig := Signature(tc.method)
			assert.Len(t, sig, tc.wantLen)
			if tc.wantHex != "" {
				assert.Equal(t, tc.wantHex, hex.EncodeToString(sig))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// convetToAddress (internal)
// ---------------------------------------------------------------------------

func TestConvetToAddress(t *testing.T) {
	t.Run("valid tron address string", func(t *testing.T) {
		addr, err := convetToAddress("TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R")
		require.NoError(t, err)
		assert.NotEqual(t, eCommon.Address{}, addr)
	})

	t.Run("invalid address string", func(t *testing.T) {
		_, err := convetToAddress("not_a_valid_address")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid address")
	})

	t.Run("non-string type returns error", func(t *testing.T) {
		_, err := convetToAddress(12345)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid address")
	})

	t.Run("nil value returns error", func(t *testing.T) {
		_, err := convetToAddress(nil)
		require.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// convertToInt (internal)
// ---------------------------------------------------------------------------

func TestConvertToInt(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		input    string
		expected interface{}
	}{
		// Unsigned integers
		{
			name:     "uint8",
			typeName: "uint8",
			input:    "255",
			expected: uint8(255),
		},
		{
			name:     "uint16",
			typeName: "uint16",
			input:    "65535",
			expected: uint16(65535),
		},
		{
			name:     "uint32",
			typeName: "uint32",
			input:    "4294967295",
			expected: uint32(4294967295),
		},
		{
			name:     "uint64",
			typeName: "uint64",
			input:    "18446744073709551615",
			expected: uint64(18446744073709551615),
		},
		// Signed integers
		{
			name:     "int8",
			typeName: "int8",
			input:    "-128",
			expected: int8(-128),
		},
		{
			name:     "int16",
			typeName: "int16",
			input:    "-32768",
			expected: int16(-32768),
		},
		{
			name:     "int32",
			typeName: "int32",
			input:    "-2147483648",
			expected: int32(-2147483648),
		},
		{
			name:     "int64",
			typeName: "int64",
			input:    "-9223372036854775808",
			expected: int64(-9223372036854775808),
		},
		// Big integers (size > 64)
		{
			name:     "uint256 decimal",
			typeName: "uint256",
			input:    "115792089237316195423570985008687907853269984665640564039457584007913129639935",
		},
		{
			name:     "uint256 hex",
			typeName: "uint256",
			input:    "0xABCDEF",
		},
		{
			name:     "int256 negative",
			typeName: "int256",
			input:    "-1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ty, err := eABI.NewType(tc.typeName, "", nil)
			require.NoError(t, err)

			result := convertToInt(ty, tc.input)
			require.NotNil(t, result)

			if tc.expected != nil {
				assert.Equal(t, tc.expected, result)
			}
			// For big ints just verify we got a *big.Int
			if ty.Size > 64 || (ty.T != eABI.IntTy && ty.T != eABI.UintTy) ||
				(ty.T == eABI.IntTy && ty.Size > 64) ||
				(ty.T == eABI.UintTy && ty.Size > 64) {
				if tc.expected == nil {
					_, ok := result.(*big.Int)
					assert.True(t, ok, "expected *big.Int for %s", tc.typeName)
				}
			}
		})
	}
}

func TestConvertToInt_InvalidStrings(t *testing.T) {
	// The original convertToInt silently returns zero/nil for invalid input.
	// Verify it does not panic on bad input.
	tests := []struct {
		name     string
		typeName string
		input    interface{}
	}{
		{name: "invalid uint8", typeName: "uint8", input: "not-a-number"},
		{name: "overflow uint8", typeName: "uint8", input: "256"},
		{name: "invalid int16", typeName: "int16", input: "abc"},
		{name: "invalid uint256 decimal", typeName: "uint256", input: "xyz"},
		{name: "invalid uint256 hex", typeName: "uint256", input: "0xnothex"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ty, err := eABI.NewType(tt.typeName, "", nil)
			require.NoError(t, err)

			assert.NotPanics(t, func() {
				convertToInt(ty, tt.input)
			})
		})
	}
}

// ---------------------------------------------------------------------------
// convertToBytes (internal)
// ---------------------------------------------------------------------------

func TestConvertToBytes(t *testing.T) {
	t.Run("hex string to dynamic bytes", func(t *testing.T) {
		ty, err := eABI.NewType("bytes", "", nil)
		require.NoError(t, err)

		result, err := convertToBytes(ty, "deadbeef")
		require.NoError(t, err)
		b, ok := result.([]byte)
		require.True(t, ok)
		assert.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, b)
	})

	t.Run("base64 string to dynamic bytes", func(t *testing.T) {
		ty, err := eABI.NewType("bytes", "", nil)
		require.NoError(t, err)

		original := []byte("hello world")
		b64 := base64.StdEncoding.EncodeToString(original)

		result, err := convertToBytes(ty, b64)
		require.NoError(t, err)
		b, ok := result.([]byte)
		require.True(t, ok)
		assert.Equal(t, original, b)
	})

	t.Run("invalid string not hex not base64", func(t *testing.T) {
		ty, err := eABI.NewType("bytes", "", nil)
		require.NoError(t, err)

		_, err = convertToBytes(ty, "!!!not-valid!!!")
		require.Error(t, err)
	})

	t.Run("bytes1 fixed size", func(t *testing.T) {
		ty, err := eABI.NewType("bytes1", "", nil)
		require.NoError(t, err)

		result, err := convertToBytes(ty, "ab")
		require.NoError(t, err)
		arr, ok := result.([1]byte)
		require.True(t, ok)
		assert.Equal(t, [1]byte{0xab}, arr)
	})

	t.Run("bytes2 fixed size", func(t *testing.T) {
		ty, err := eABI.NewType("bytes2", "", nil)
		require.NoError(t, err)

		result, err := convertToBytes(ty, "abcd")
		require.NoError(t, err)
		arr, ok := result.([2]byte)
		require.True(t, ok)
		assert.Equal(t, [2]byte{0xab, 0xcd}, arr)
	})

	t.Run("bytes8 fixed size", func(t *testing.T) {
		ty, err := eABI.NewType("bytes8", "", nil)
		require.NoError(t, err)

		hexStr := "0102030405060708"
		result, err := convertToBytes(ty, hexStr)
		require.NoError(t, err)
		arr, ok := result.([8]byte)
		require.True(t, ok)
		assert.Equal(t, [8]byte{1, 2, 3, 4, 5, 6, 7, 8}, arr)
	})

	t.Run("bytes16 fixed size", func(t *testing.T) {
		ty, err := eABI.NewType("bytes16", "", nil)
		require.NoError(t, err)

		hexStr := "0102030405060708090a0b0c0d0e0f10"
		result, err := convertToBytes(ty, hexStr)
		require.NoError(t, err)
		arr, ok := result.([16]byte)
		require.True(t, ok)
		assert.Equal(t, byte(0x01), arr[0])
		assert.Equal(t, byte(0x10), arr[15])
	})

	t.Run("bytes32 fixed size", func(t *testing.T) {
		ty, err := eABI.NewType("bytes32", "", nil)
		require.NoError(t, err)

		hexStr := "0001020001020001020001020001020001020001020001020001020001020001"
		result, err := convertToBytes(ty, hexStr)
		require.NoError(t, err)
		_, ok := result.([32]byte)
		require.True(t, ok)
	})

	t.Run("fixed bytes size mismatch", func(t *testing.T) {
		ty, err := eABI.NewType("bytes2", "", nil)
		require.NoError(t, err)

		// provide 4 bytes when only 2 expected
		_, err = convertToBytes(ty, "aabbccdd")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid size")
	})

	t.Run("non-string value passes through unchanged", func(t *testing.T) {
		ty, err := eABI.NewType("bytes", "", nil)
		require.NoError(t, err)

		input := []byte{0x01, 0x02}
		result, err := convertToBytes(ty, input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})
}

// ---------------------------------------------------------------------------
// toStringSlice (internal)
// ---------------------------------------------------------------------------

func TestToStringSlice(t *testing.T) {
	t.Run("[]string passes through", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result, err := toStringSlice(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("[]interface{} with strings converts", func(t *testing.T) {
		input := []interface{}{"100", "200", "300"}
		result, err := toStringSlice(input)
		require.NoError(t, err)
		assert.Equal(t, []string{"100", "200", "300"}, result)
	})

	t.Run("[]interface{} with non-string element errors", func(t *testing.T) {
		input := []interface{}{"100", 200, "300"}
		_, err := toStringSlice(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "element 1 is not a string")
	})

	t.Run("unsupported type errors", func(t *testing.T) {
		_, err := toStringSlice(42)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected string slice")
	})
}

// ---------------------------------------------------------------------------
// convertSmallIntSlice (internal)
// ---------------------------------------------------------------------------

func TestConvertSmallIntSlice(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		input    []string
		validate func(t *testing.T, result interface{})
	}{
		{
			name:     "uint8 slice",
			typeName: "uint8",
			input:    []string{"0", "127", "255"},
			validate: func(t *testing.T, result interface{}) {
				v, ok := result.([]uint8)
				require.True(t, ok)
				assert.Equal(t, []uint8{0, 127, 255}, v)
			},
		},
		{
			name:     "uint16 slice",
			typeName: "uint16",
			input:    []string{"0", "1000", "65535"},
			validate: func(t *testing.T, result interface{}) {
				v, ok := result.([]uint16)
				require.True(t, ok)
				assert.Equal(t, []uint16{0, 1000, 65535}, v)
			},
		},
		{
			name:     "uint32 slice",
			typeName: "uint32",
			input:    []string{"0", "100000", "4294967295"},
			validate: func(t *testing.T, result interface{}) {
				v, ok := result.([]uint32)
				require.True(t, ok)
				assert.Equal(t, []uint32{0, 100000, 4294967295}, v)
			},
		},
		{
			name:     "uint64 slice",
			typeName: "uint64",
			input:    []string{"0", "18446744073709551615"},
			validate: func(t *testing.T, result interface{}) {
				v, ok := result.([]uint64)
				require.True(t, ok)
				assert.Equal(t, []uint64{0, 18446744073709551615}, v)
			},
		},
		{
			name:     "int8 slice",
			typeName: "int8",
			input:    []string{"-128", "0", "127"},
			validate: func(t *testing.T, result interface{}) {
				v, ok := result.([]int8)
				require.True(t, ok)
				assert.Equal(t, []int8{-128, 0, 127}, v)
			},
		},
		{
			name:     "int16 slice",
			typeName: "int16",
			input:    []string{"-32768", "0", "32767"},
			validate: func(t *testing.T, result interface{}) {
				v, ok := result.([]int16)
				require.True(t, ok)
				assert.Equal(t, []int16{-32768, 0, 32767}, v)
			},
		},
		{
			name:     "int32 slice",
			typeName: "int32",
			input:    []string{"-2147483648", "0", "2147483647"},
			validate: func(t *testing.T, result interface{}) {
				v, ok := result.([]int32)
				require.True(t, ok)
				assert.Equal(t, []int32{-2147483648, 0, 2147483647}, v)
			},
		},
		{
			name:     "int64 slice",
			typeName: "int64",
			input:    []string{"-9223372036854775808", "0", "9223372036854775807"},
			validate: func(t *testing.T, result interface{}) {
				v, ok := result.([]int64)
				require.True(t, ok)
				assert.Equal(t, []int64{-9223372036854775808, 0, 9223372036854775807}, v)
			},
		},
		{
			name:     "empty slice",
			typeName: "uint32",
			input:    []string{},
			validate: func(t *testing.T, result interface{}) {
				v, ok := result.([]uint32)
				require.True(t, ok)
				assert.Empty(t, v)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Build the element type from the slice element type name
			elemTy, err := eABI.NewType(tc.typeName, "", nil)
			require.NoError(t, err)

			result := convertSmallIntSlice(elemTy, tc.input)
			require.NotNil(t, result)
			tc.validate(t, result)
		})
	}
}

func TestConvertSmallIntSlice_Fallback(t *testing.T) {
	// The default case falls back to big.Int for unexpected sizes.
	// We can trigger this with a uint24 type which has Size=24 (not 8/16/32/64).
	// However, eABI.NewType may not support uint24 directly.
	// Instead, we construct a type manually or use uint128 which also falls through.
	// Actually, go-ethereum ABI supports uint256 but Size=256 > 64 so it won't enter
	// convertSmallIntSlice at all (it's handled earlier). Let's use uint48.
	// Actually, eABI.NewType only supports standard EVM sizes (multiples of 8 up to 256).
	// Let's test with uint24 which is valid in Solidity.
	elemTy, err := eABI.NewType("uint24", "", nil)
	require.NoError(t, err)

	result := convertSmallIntSlice(elemTy, []string{"100", "200"})
	// Should fall through to big.Int default case since 24 is not 8/16/32/64
	bigSlice, ok := result.([]*big.Int)
	require.True(t, ok, "expected []*big.Int for uint24 fallback, got %T", result)
	assert.Equal(t, big.NewInt(100), bigSlice[0])
	assert.Equal(t, big.NewInt(200), bigSlice[1])
}

func TestConvertSmallIntSlice_InvalidInput(t *testing.T) {
	// The original convertSmallIntSlice silently returns zero for invalid input.
	// Verify it does not panic.
	elemTy, err := eABI.NewType("uint8", "", nil)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		convertSmallIntSlice(elemTy, []string{"1", "overflow-999"})
	})
}

// ---------------------------------------------------------------------------
// Pack (exported)
// ---------------------------------------------------------------------------

func TestPack(t *testing.T) {
	t.Run("pack transfer call", func(t *testing.T) {
		params := []Param{
			{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},
			{"uint256": "1000000"},
		}
		result, err := Pack("transfer(address,uint256)", params)
		require.NoError(t, err)

		// First 4 bytes should be method selector
		assert.Equal(t, "a9059cbb", hex.EncodeToString(result[:4]))
		// Total length: 4 (selector) + 32 (address) + 32 (uint256) = 68
		assert.Len(t, result, 68)
	})

	t.Run("pack with no params", func(t *testing.T) {
		result, err := Pack("totalSupply()", nil)
		require.NoError(t, err)
		// Only the 4-byte selector
		assert.Len(t, result, 4)
	})

	t.Run("pack with invalid param type", func(t *testing.T) {
		params := []Param{
			{"invalidtype!!!": "123"},
		}
		_, err := Pack("foo(invalidtype!!!)", params)
		require.Error(t, err)
	})

	t.Run("pack with string param", func(t *testing.T) {
		params := []Param{
			{"string": "hello"},
		}
		result, err := Pack("greet(string)", params)
		require.NoError(t, err)
		require.Len(t, result, 4+32+32+32)
		assert.Equal(t, hex.EncodeToString(Signature("greet(string)")), hex.EncodeToString(result[:4]))
		assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000020", hex.EncodeToString(result[4:36]))
		assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000005", hex.EncodeToString(result[36:68]))
		assert.Equal(t, "68656c6c6f000000000000000000000000000000000000000000000000000000", hex.EncodeToString(result[68:100]))
	})
}

// ---------------------------------------------------------------------------
// GetPaddedParam edge cases
// ---------------------------------------------------------------------------

func TestGetPaddedParam_Errors(t *testing.T) {
	t.Run("param with multiple keys errors", func(t *testing.T) {
		params := []Param{
			{"uint256": "100", "address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},
		}
		_, err := GetPaddedParam(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid param")
	})

	t.Run("invalid type name errors", func(t *testing.T) {
		params := []Param{
			{"not_a_real_type": "123"},
		}
		_, err := GetPaddedParam(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid param")
	})

	t.Run("invalid address in address param", func(t *testing.T) {
		params := []Param{
			{"address": "not_valid"},
		}
		_, err := GetPaddedParam(params)
		require.Error(t, err)
	})

	t.Run("invalid address in address array", func(t *testing.T) {
		params := []Param{
			{"address[]": []interface{}{"not_a_valid_address"}},
		}
		_, err := GetPaddedParam(params)
		require.Error(t, err)
	})

	t.Run("address array with non-slice value", func(t *testing.T) {
		params := []Param{
			{"address[]": "not_a_slice"},
		}
		_, err := GetPaddedParam(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to convert array of addresses")
	})

	t.Run("int array with non-convertible value", func(t *testing.T) {
		params := []Param{
			{"uint256[]": 12345},
		}
		_, err := GetPaddedParam(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to convert array of ints")
	})

	t.Run("bytes array with non-slice value", func(t *testing.T) {
		params := []Param{
			{"bytes[]": "not_a_slice"},
		}
		_, err := GetPaddedParam(params)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unable to convert array of bytes")
	})

	t.Run("bytes array with invalid hex element", func(t *testing.T) {
		params := []Param{
			{"bytes[]": []interface{}{"!!!invalid_hex!!!"}},
		}
		_, err := GetPaddedParam(params)
		require.Error(t, err)
	})
}

func TestGetPaddedParam_IntTypes(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		value    string
	}{
		{"int8", "int8", "-1"},
		{"int16", "int16", "-256"},
		{"int32", "int32", "-70000"},
		{"int64", "int64", "-1000000000"},
		{"int256", "int256", "-1"},
		{"uint8", "uint8", "255"},
		{"uint16", "uint16", "65535"},
		{"uint32", "uint32", "4294967295"},
		{"uint64", "uint64", "18446744073709551615"},
		{"uint256", "uint256", "1000000000000000000"},
		{"uint256 hex", "uint256", "0xff"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := []Param{{tc.typeName: tc.value}}
			b, err := GetPaddedParam(params)
			require.NoError(t, err)
			assert.Len(t, b, 32, "ABI encoding of %s should be 32 bytes", tc.typeName)
		})
	}
}

func TestGetPaddedParam_IntTypes_Overflow(t *testing.T) {
	// With the current implementation, convertToInt silently truncates
	// overflowing values. Verify the function does not panic.
	tests := []struct {
		name     string
		typeName string
		value    string
	}{
		{"overflow uint8", "uint8", "256"},
		{"overflow int8", "int8", "128"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := []Param{{tc.typeName: tc.value}}
			// strconv.ParseUint/ParseInt silently returns 0 on error,
			// so we get a zero-value encoding rather than an error.
			b, err := GetPaddedParam(params)
			require.NoError(t, err)
			assert.Len(t, b, 32)
		})
	}
}

func TestGetPaddedParam_SmallIntArrays(t *testing.T) {
	tests := []struct {
		name     string
		typeName string
		values   []string
	}{
		{"uint8[]", "uint8[]", []string{"1", "2", "3"}},
		{"uint16[]", "uint16[]", []string{"1000", "2000"}},
		{"uint32[]", "uint32[]", []string{"100000", "200000"}},
		{"uint64[]", "uint64[]", []string{"1000000000", "2000000000"}},
		{"int8[]", "int8[]", []string{"-1", "0", "1"}},
		{"int16[]", "int16[]", []string{"-1000", "1000"}},
		{"int32[]", "int32[]", []string{"-100000", "100000"}},
		{"int64[]", "int64[]", []string{"-1000000000", "1000000000"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params := []Param{{tc.typeName: tc.values}}
			b, err := GetPaddedParam(params)
			require.NoError(t, err)
			// Dynamic array: offset(32) + length(32) + N*32 elements
			expected := 32 + 32 + len(tc.values)*32
			assert.Len(t, b, expected, "unexpected length for %s", tc.typeName)
		})
	}
}

func TestGetPaddedParam_SmallIntArraysFromJSON(t *testing.T) {
	// JSON unmarshaling produces []interface{}, not []string
	tests := []struct {
		name    string
		json    string
		wantLen int
	}{
		{
			name:    "uint8[] from JSON",
			json:    `[{"uint8[]": ["1", "2", "3"]}]`,
			wantLen: 32 + 32 + 3*32,
		},
		{
			name:    "int32[] from JSON",
			json:    `[{"int32[]": ["-100", "0", "100"]}]`,
			wantLen: 32 + 32 + 3*32,
		},
		{
			name:    "uint64[] from JSON",
			json:    `[{"uint64[]": ["1000000000", "2000000000"]}]`,
			wantLen: 32 + 32 + 2*32,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			params, err := LoadFromJSON(tc.json)
			require.NoError(t, err)
			b, err := GetPaddedParam(params)
			require.NoError(t, err)
			assert.Len(t, b, tc.wantLen)
		})
	}
}

func TestGetPaddedParam_Bool(t *testing.T) {
	params := []Param{{"bool": true}}
	b, err := GetPaddedParam(params)
	require.NoError(t, err)
	assert.Len(t, b, 32)
	// Last byte should be 1
	assert.Equal(t, byte(1), b[31])
}

func TestGetPaddedParam_DynamicBytes(t *testing.T) {
	params := []Param{{"bytes": "deadbeef"}}
	b, err := GetPaddedParam(params)
	require.NoError(t, err)
	// Dynamic bytes: offset(32) + length(32) + data padded to 32
	assert.Len(t, b, 96)
}

func TestGetPaddedParam_Base64Bytes(t *testing.T) {
	// Provide bytes as base64 string
	original := []byte{0xde, 0xad, 0xbe, 0xef}
	b64 := base64.StdEncoding.EncodeToString(original)

	params := []Param{{"bytes": b64}}
	b, err := GetPaddedParam(params)
	require.NoError(t, err)
	require.Len(t, b, 96)
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000020", hex.EncodeToString(b[:32]))
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000004", hex.EncodeToString(b[32:64]))
	assert.Equal(t, "deadbeef00000000000000000000000000000000000000000000000000000000", hex.EncodeToString(b[64:96]))
}

func TestGetPaddedParam_MultipleAddresses(t *testing.T) {
	params := []Param{
		{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},
		{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},
	}
	b, err := GetPaddedParam(params)
	require.NoError(t, err)
	assert.Len(t, b, 64) // 2 * 32
	// Both should encode to same value
	assert.Equal(t, hex.EncodeToString(b[:32]), hex.EncodeToString(b[32:]))
}

// ---------------------------------------------------------------------------
// GetParser error paths
// ---------------------------------------------------------------------------

func TestGetParser_InvalidOutputType(t *testing.T) {
	contractABI := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "badFunc",
				Type: core.SmartContract_ABI_Entry_Function,
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "x", Type: "invalid_type!!!"},
				},
			},
		},
	}

	_, err := GetParser(contractABI, "badFunc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid param")
}

func TestGetInputsParser_InvalidInputType(t *testing.T) {
	contractABI := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "badFunc",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "x", Type: "invalid_type!!!"},
				},
			},
		},
	}

	_, err := GetInputsParser(contractABI, "badFunc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid param")
}

// ---------------------------------------------------------------------------
// GetEventParser edge cases
// ---------------------------------------------------------------------------

func TestGetEventParser_InvalidParamType(t *testing.T) {
	contractABI := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "BadEvent",
				Type: core.SmartContract_ABI_Entry_Event,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "x", Type: "invalid_type!!!"},
				},
			},
		},
	}

	_, _, err := GetEventParser(contractABI, "BadEvent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid param")
}

func TestGetEventParser_BySignature(t *testing.T) {
	contractABI := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "Transfer",
				Type: core.SmartContract_ABI_Entry_Event,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "from", Type: "address", Indexed: true},
					{Name: "to", Type: "address", Indexed: true},
					{Name: "value", Type: "uint256"},
				},
			},
		},
	}

	indexed, nonIndexed, err := GetEventParser(contractABI, "Transfer(address,address,uint256)")
	require.NoError(t, err)
	assert.Len(t, indexed, 2)
	assert.Len(t, nonIndexed, 1)
}

func TestGetEventParser_AllIndexed(t *testing.T) {
	contractABI := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "Approval",
				Type: core.SmartContract_ABI_Entry_Event,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "owner", Type: "address", Indexed: true},
					{Name: "spender", Type: "address", Indexed: true},
					{Name: "value", Type: "uint256", Indexed: true},
				},
			},
		},
	}

	indexed, nonIndexed, err := GetEventParser(contractABI, "Approval")
	require.NoError(t, err)
	assert.Len(t, indexed, 3)
	assert.Len(t, nonIndexed, 0)
}

func TestGetEventParser_NoIndexed(t *testing.T) {
	contractABI := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "DataStored",
				Type: core.SmartContract_ABI_Entry_Event,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "key", Type: "uint256"},
					{Name: "value", Type: "uint256"},
				},
			},
		},
	}

	indexed, nonIndexed, err := GetEventParser(contractABI, "DataStored")
	require.NoError(t, err)
	assert.Len(t, indexed, 0)
	assert.Len(t, nonIndexed, 2)
}

// ---------------------------------------------------------------------------
// ParseTopicsIntoMap edge cases
// ---------------------------------------------------------------------------

func TestParseTopicsIntoMap_NonAddressTopics(t *testing.T) {
	// Event with indexed uint256 (not address) -- should not be converted
	tyUint, err := eABI.NewType("uint256", "", nil)
	require.NoError(t, err)

	fields := eABI.Arguments{
		{Name: "tokenId", Type: tyUint, Indexed: true},
	}

	topic := eCommon.BigToHash(big.NewInt(42))
	out := make(map[string]interface{})
	err = ParseTopicsIntoMap(out, fields, [][]byte{topic.Bytes()})
	require.NoError(t, err)

	val, ok := out["tokenId"]
	require.True(t, ok)
	// Should remain a *big.Int, not be converted to address
	bigVal, isBigInt := val.(*big.Int)
	require.True(t, isBigInt, "expected *big.Int, got %T", val)
	assert.Equal(t, big.NewInt(42), bigVal)
}

func TestParseTopicsIntoMap_EmptyTopics(t *testing.T) {
	out := make(map[string]interface{})
	err := ParseTopicsIntoMap(out, eABI.Arguments{}, [][]byte{})
	require.NoError(t, err)
	assert.Empty(t, out)
}

func TestParseTopicsIntoMap_NilOutMap(t *testing.T) {
	err := ParseTopicsIntoMap(nil, eABI.Arguments{}, [][]byte{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out is nil")
}

func TestParseTopicsIntoMap_MixedTypes(t *testing.T) {
	// address indexed + uint256 indexed
	tyAddr, err := eABI.NewType("address", "", nil)
	require.NoError(t, err)
	tyUint, err := eABI.NewType("uint256", "", nil)
	require.NoError(t, err)

	fields := eABI.Arguments{
		{Name: "sender", Type: tyAddr, Indexed: true},
		{Name: "amount", Type: tyUint, Indexed: true},
	}

	addrTopic := make([]byte, 32)
	addrTopic[31] = 0xAB
	amountTopic := eCommon.BigToHash(big.NewInt(999))

	out := make(map[string]interface{})
	err = ParseTopicsIntoMap(out, fields, [][]byte{addrTopic, amountTopic.Bytes()})
	require.NoError(t, err)

	// sender should be converted to TRON address
	senderVal, ok := out["sender"]
	require.True(t, ok)
	tronAddr, isAddr := senderVal.(address.Address)
	require.True(t, isAddr, "expected address.Address, got %T", senderVal)
	expected := make(address.Address, 21)
	expected[0] = address.TronBytePrefix
	copy(expected[1:], addrTopic[12:])
	assert.Equal(t, expected, tronAddr)

	// amount should be *big.Int
	amountVal, ok := out["amount"]
	require.True(t, ok)
	bigVal, isBigInt := amountVal.(*big.Int)
	require.True(t, isBigInt, "expected *big.Int, got %T", amountVal)
	assert.Equal(t, big.NewInt(999), bigVal)
}

// ---------------------------------------------------------------------------
// Pack + unpack round-trip
// ---------------------------------------------------------------------------

func TestPack_RoundTrip(t *testing.T) {
	// Pack a transfer(address,uint256) call and verify we can unpack it
	contractABI := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "transfer",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "to", Type: "address"},
					{Name: "amount", Type: "uint256"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "success", Type: "bool"},
				},
			},
		},
	}

	params := []Param{
		{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},
		{"uint256": "1000000"},
	}

	packed, err := Pack("transfer(address,uint256)", params)
	require.NoError(t, err)

	// Verify selector
	assert.Equal(t, "a9059cbb", hex.EncodeToString(packed[:4]))

	// Parse inputs back
	inputParser, err := GetInputsParser(contractABI, "transfer(address,uint256)")
	require.NoError(t, err)

	values, err := inputParser.UnpackValues(packed[4:])
	require.NoError(t, err)
	assert.Len(t, values, 2)

	// First value should be an address
	addr, ok := values[0].(eCommon.Address)
	require.True(t, ok)
	expectedTronAddr, err := address.Base58ToAddress("TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R")
	require.NoError(t, err)
	expectedEthAddr := eCommon.BytesToAddress(expectedTronAddr.Bytes()[len(expectedTronAddr.Bytes())-20:])
	assert.Equal(t, expectedEthAddr, addr)

	// Second value should be *big.Int with value 1000000
	amount, ok := values[1].(*big.Int)
	require.True(t, ok)
	assert.Equal(t, big.NewInt(1000000), amount)
}

// ---------------------------------------------------------------------------
// GetParser / GetInputsParser with empty ABI
// ---------------------------------------------------------------------------

func TestGetParser_EmptyABI(t *testing.T) {
	contractABI := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{},
	}

	_, err := GetParser(contractABI, "anyMethod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetInputsParser_EmptyABI(t *testing.T) {
	contractABI := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{},
	}

	_, err := GetInputsParser(contractABI, "anyMethod")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ---------------------------------------------------------------------------
// GetParser with nil entries
// ---------------------------------------------------------------------------

func TestGetParser_NilABI(t *testing.T) {
	contractABI := &core.SmartContract_ABI{}
	_, err := GetParser(contractABI, "anyMethod")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// entrySignature edge cases
// ---------------------------------------------------------------------------

func TestEntrySignature_MultipleComplexTypes(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Name: "complexMethod",
		Inputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "a", Type: "address"},
			{Name: "b", Type: "uint256[]"},
			{Name: "c", Type: "bytes32"},
			{Name: "d", Type: "string"},
		},
	}

	expected := "complexMethod(address,uint256[],bytes32,string)"
	assert.Equal(t, expected, entrySignature(entry))
}

// ---------------------------------------------------------------------------
// GetPaddedParam with fixed-size int arrays
// ---------------------------------------------------------------------------

func TestGetPaddedParam_FixedSizeUintArray(t *testing.T) {
	// uint256[3] - fixed size array
	params := []Param{
		{"uint256[3]": []string{"100", "200", "300"}},
	}
	b, err := GetPaddedParam(params)
	require.NoError(t, err)
	// Fixed array: 3 * 32 = 96
	assert.Len(t, b, 96)
}

func TestGetPaddedParam_FixedSizeAddressArray(t *testing.T) {
	param, err := LoadFromJSON(fmt.Sprintf(`[{"address[1]":["%s"]}]`, "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"))
	require.NoError(t, err)
	b, err := GetPaddedParam(param)
	require.NoError(t, err)
	assert.Len(t, b, 32)
}

// ---------------------------------------------------------------------------
// Multiple string params
// ---------------------------------------------------------------------------

func TestGetPaddedParam_MultipleStrings(t *testing.T) {
	params := []Param{
		{"string": "hello"},
		{"string": "world"},
	}
	b, err := GetPaddedParam(params)
	require.NoError(t, err)
	ty, err := eABI.NewType("string", "", nil)
	require.NoError(t, err)
	args := eABI.Arguments{{Type: ty}, {Type: ty}}
	values, err := args.UnpackValues(b)
	require.NoError(t, err)
	require.Len(t, values, 2)
	assert.Equal(t, "hello", values[0])
	assert.Equal(t, "world", values[1])
}

// ---------------------------------------------------------------------------
// Encoding consistency: same input produces same output
// ---------------------------------------------------------------------------

func TestGetPaddedParam_Deterministic(t *testing.T) {
	params := []Param{
		{"uint256": "42"},
		{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},
		{"bool": true},
	}

	b1, err := GetPaddedParam(params)
	require.NoError(t, err)

	b2, err := GetPaddedParam(params)
	require.NoError(t, err)

	assert.Equal(t, b1, b2, "encoding should be deterministic")
}

// ---------------------------------------------------------------------------
// matchEntry additional cases
// ---------------------------------------------------------------------------

func TestMatchEntry_EmptyInputs(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Name:   "noArgs",
		Inputs: []*core.SmartContract_ABI_Entry_Param{},
	}

	assert.True(t, matchEntry(entry, "noArgs"))
	assert.True(t, matchEntry(entry, "noArgs()"))
	assert.False(t, matchEntry(entry, "noArgs(uint256)"))
}

// ---------------------------------------------------------------------------
// GetPaddedParam with uint256[] hex values from JSON
// ---------------------------------------------------------------------------

func TestGetPaddedParam_Uint256SliceHexFromJSON(t *testing.T) {
	param, err := LoadFromJSON(`[{"uint256[]":["0x1", "0xff", "0xabcdef"]}]`)
	require.NoError(t, err)
	b, err := GetPaddedParam(param)
	require.NoError(t, err)
	// offset(32) + length(32) + 3 elements(96) = 160
	assert.Len(t, b, 160)
}

// ---------------------------------------------------------------------------
// GetPaddedParam with int256 (signed big int)
// ---------------------------------------------------------------------------

func TestGetPaddedParam_Int256(t *testing.T) {
	params := []Param{{"int256": "-1"}}
	b, err := GetPaddedParam(params)
	require.NoError(t, err)
	assert.Len(t, b, 32)
	// -1 in two's complement should be all ff bytes
	assert.Equal(t, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", hex.EncodeToString(b))
}

// ---------------------------------------------------------------------------
// GetPaddedParam with bytes (base64 fallback path)
// ---------------------------------------------------------------------------

func TestGetPaddedParam_BytesBase64(t *testing.T) {
	// "AQID" is base64 for [0x01, 0x02, 0x03]
	params := []Param{{"bytes": "AQID"}}
	b, err := GetPaddedParam(params)
	require.NoError(t, err)
	require.Len(t, b, 96)
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000020", hex.EncodeToString(b[:32]))
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000003", hex.EncodeToString(b[32:64]))
	assert.Equal(t, "0102030000000000000000000000000000000000000000000000000000000000", hex.EncodeToString(b[64:96]))
}

// ---------------------------------------------------------------------------
// GetPaddedParam with bytes invalid
// ---------------------------------------------------------------------------

func TestGetPaddedParam_BytesInvalidEncoding(t *testing.T) {
	// String that is neither valid hex nor valid base64
	params := []Param{{"bytes": "!!!invalid!!!"}}
	_, err := GetPaddedParam(params)
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// ParseTopicsIntoMap error from underlying go-ethereum ParseTopicsIntoMap
// ---------------------------------------------------------------------------

func TestParseTopicsIntoMap_TopicCountMismatch(t *testing.T) {
	// Provide 2 indexed fields but only 1 topic -- this should cause
	// go-ethereum's ParseTopicsIntoMap to return an error.
	tyAddr, err := eABI.NewType("address", "", nil)
	require.NoError(t, err)

	fields := eABI.Arguments{
		{Name: "from", Type: tyAddr, Indexed: true},
		{Name: "to", Type: tyAddr, Indexed: true},
	}

	// Only one topic instead of the required two
	singleTopic := make([]byte, 32)
	singleTopic[31] = 0x01

	out := make(map[string]interface{})
	err = ParseTopicsIntoMap(out, fields, [][]byte{singleTopic})
	require.Error(t, err, "mismatched topic count should produce an error")
}
