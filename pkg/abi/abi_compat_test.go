package abi

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

// Library compatibility: previously valid public-API inputs must keep the same
// encodings. These golden values are the contract downstream projects rely on.

func keccak4(s string) string {
	h := sha3.NewLegacyKeccak256()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil)[:4])
}

func TestCompat_SignatureCanonicalIsRawKeccak(t *testing.T) {
	// Canonical signatures (no whitespace) must still be keccak256(exact string)[:4].
	methods := []string{
		"transfer(address,uint256)",
		"approve(address,uint256)",
		"transferFrom(address,address,uint256)",
		"balanceOf(address)",
		"allowance(address,address)",
		"totalSupply()",
		"decimals()",
		"name()",
		"symbol()",
		"rentResource(address,uint256,uint32)",
		"returnResource(address,uint256,uint32)",
	}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			got := hex.EncodeToString(Signature(method))
			assert.Equal(t, keccak4(method), got)
		})
	}

	assert.Equal(t, "a9059cbb", hex.EncodeToString(Signature("transfer(address,uint256)")))
	assert.Equal(t, "095ea7b3", hex.EncodeToString(Signature("approve(address,uint256)")))
	assert.Equal(t, "70a08231", hex.EncodeToString(Signature("balanceOf(address)")))
	assert.Equal(t, "18160ddd", hex.EncodeToString(Signature("totalSupply()")))
	assert.Equal(t, "23b872dd", hex.EncodeToString(Signature("transferFrom(address,address,uint256)")))
}

func TestCompat_TRC20TransferPack(t *testing.T) {
	// Documented SDK pattern: quoted JSON + canonical signature.
	params, err := LoadFromJSON(
		`[{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"}, {"uint256": "1000000"}]`,
	)
	require.NoError(t, err)

	packed, err := Pack("transfer(address,uint256)", params)
	require.NoError(t, err)

	assert.Equal(t,
		"a9059cbb"+
			"000000000000000000000000364b03e0815687edaf90b81ff58e496dea7383d7"+
			"00000000000000000000000000000000000000000000000000000000000f4240",
		hex.EncodeToString(packed),
	)
}

func TestCompat_JustLendJSONPack(t *testing.T) {
	params, err := LoadFromJSON(
		`[{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"}, {"uint256": "1000000"}, {"uint32": "1"}]`,
	)
	require.NoError(t, err)

	packed, err := Pack("rentResource(address,uint256,uint32)", params)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(packed), 4)
	assert.Equal(t, keccak4("rentResource(address,uint256,uint32)"), hex.EncodeToString(packed[:4]))
}

func TestCompat_LoadFromJSONQuotedValuesStayStrings(t *testing.T) {
	params, err := LoadFromJSON(
		`[{"address": "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"}, {"uint256": "1000000"}, {"uint32": "1"}]`,
	)
	require.NoError(t, err)
	require.Len(t, params, 3)

	_, ok := params[0]["address"].(string)
	assert.True(t, ok, "address must remain a string")
	_, ok = params[1]["uint256"].(string)
	assert.True(t, ok, "quoted uint256 must remain a string")
	_, ok = params[2]["uint32"].(string)
	assert.True(t, ok, "quoted uint32 must remain a string")
}

func TestCompat_NativeAndStringIntsPackIdentically(t *testing.T) {
	large := new(big.Int)
	_, ok := large.SetString("100000000000000000000", 10)
	require.True(t, ok)

	fromBig, err := GetPaddedParam([]Param{{"uint256": large}})
	require.NoError(t, err)
	fromString, err := GetPaddedParam([]Param{{"uint256": large.String()}})
	require.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(fromString), hex.EncodeToString(fromBig))

	fromNative, err := GetPaddedParam([]Param{{"uint8": uint8(6)}})
	require.NoError(t, err)
	fromUint8String, err := GetPaddedParam([]Param{{"uint8": "6"}})
	require.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(fromUint8String), hex.EncodeToString(fromNative))

	fromUint32, err := GetPaddedParam([]Param{{"uint32": uint32(1)}})
	require.NoError(t, err)
	fromUint32String, err := GetPaddedParam([]Param{{"uint32": "1"}})
	require.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(fromUint32String), hex.EncodeToString(fromUint32))
}

func TestCompat_Int256NegativeOne(t *testing.T) {
	b, err := GetPaddedParam([]Param{{"int256": "-1"}})
	require.NoError(t, err)
	assert.Equal(t, "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", hex.EncodeToString(b))
}

func TestCompat_Bytes32AndHexUint256(t *testing.T) {
	b, err := GetPaddedParam([]Param{
		{"bytes32": "0001020001020001020001020001020001020001020001020001020001020001"},
	})
	require.NoError(t, err)
	assert.Equal(t, "0001020001020001020001020001020001020001020001020001020001020001", hex.EncodeToString(b))

	hexed, err := GetPaddedParam([]Param{
		{"uint256": "43981"},
		{"uint256": "0xABCD"},
	})
	require.NoError(t, err)
	assert.Equal(t,
		"000000000000000000000000000000000000000000000000000000000000abcd"+
			"000000000000000000000000000000000000000000000000000000000000abcd",
		hex.EncodeToString(hexed),
	)
}

func TestCompat_ConstructorStyleStrings(t *testing.T) {
	b, err := GetPaddedParam([]Param{
		{"string": "KLV Test Token"},
		{"string": "KLV"},
		{"uint8": uint8(6)},
		{"uint256": "100000000000000000000"},
	})
	require.NoError(t, err)
	assert.Len(t, b, 256)
}
