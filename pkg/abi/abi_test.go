package abi

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestABIParam(t *testing.T) {
	ss, _ := new(big.Int).SetString("100000000000000000000", 10)
	b, err := GetPaddedParam([]Param{
		{"string": "KLV Test Token"},
		{"string": "KLV"},
		{"uint8": uint8(6)},
		{"uint256": ss},
	})
	require.Nil(t, err)
	assert.Len(t, b, 256, fmt.Sprintf("Wrong length %d/%d", len(b), 256))

	b, err = GetPaddedParam([]Param{
		{"string": "KLV Test Token"},
		{"string": "KLV"},
		{"uint8": "6"},
		{"uint256": ss.String()},
	})
	require.Nil(t, err)
	assert.Len(t, b, 256, fmt.Sprintf("Wrong length %d/%d", len(b), 256))
}

func TestABIParamArray(t *testing.T) {
	param, err := LoadFromJSON(`
	[
		{"address[2]":["TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R", "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"]}
	]
	`)
	require.Nil(t, err)
	b, err := GetPaddedParam(param)
	require.Nil(t, err)
	assert.Len(t, b, 64, fmt.Sprintf("Wrong length %d/%d", len(b), 64))
	assert.Equal(t, "000000000000000000000000364b03e0815687edaf90b81ff58e496dea7383d7000000000000000000000000364b03e0815687edaf90b81ff58e496dea7383d7", hex.EncodeToString(b))
}

func TestABIParamArrayUint256(t *testing.T) {
	b, err := GetPaddedParam([]Param{{"uint256[2]": []string{"100000000000000000000", "200000000000000000000"}}})
	require.Nil(t, err)
	assert.Len(t, b, 64, fmt.Sprintf("Wrong length %d/%d", len(b), 64))
	assert.Equal(t, "0000000000000000000000000000000000000000000000056bc75e2d6310000000000000000000000000000000000000000000000000000ad78ebc5ac6200000", hex.EncodeToString(b))
}

func TestABIParamArrayBytes(t *testing.T) {

	param, err := LoadFromJSON(`
	[
		{"bytes32": "0001020001020001020001020001020001020001020001020001020001020001"}
	]
	`)
	require.Nil(t, err)
	b, err := GetPaddedParam(param)
	require.Nil(t, err)
	assert.Len(t, b, 32, fmt.Sprintf("Wrong length %d/%d", len(b), 64))
	assert.Equal(t, "0001020001020001020001020001020001020001020001020001020001020001", hex.EncodeToString(b))
}

func TestABI_HEXuint256(t *testing.T) {
	b, err := GetPaddedParam([]Param{
		{"uint256": "43981"},
		{"uint256": "0xABCD"},
	})
	require.Nil(t, err)
	assert.Len(t, b, 64, fmt.Sprintf("Wrong length %d/%d", len(b), 256))
	assert.Equal(t, "000000000000000000000000000000000000000000000000000000000000abcd000000000000000000000000000000000000000000000000000000000000abcd", hex.EncodeToString(b))
}
