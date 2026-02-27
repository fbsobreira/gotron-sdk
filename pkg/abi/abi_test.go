package abi

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
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

// makeOverloadedABI creates a SmartContract_ABI with two overloaded methods:
//   - rollDice(uint256,uint256) returns (uint256)
//   - rollDice(uint256,uint256,address) returns (uint256,bool)
func makeOverloadedABI() *core.SmartContract_ABI {
	return &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "rollDice",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "nonce", Type: "uint256"},
					{Name: "seed", Type: "uint256"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "result", Type: "uint256"},
				},
			},
			{
				Name: "rollDice",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "nonce", Type: "uint256"},
					{Name: "seed", Type: "uint256"},
					{Name: "player", Type: "address"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "result", Type: "uint256"},
					{Name: "won", Type: "bool"},
				},
			},
			{
				Name: "getBalance",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "account", Type: "address"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "balance", Type: "uint256"},
				},
			},
		},
	}
}

func TestEntrySignature(t *testing.T) {
	abi := makeOverloadedABI()

	assert.Equal(t, "rollDice(uint256,uint256)", entrySignature(abi.Entrys[0]))
	assert.Equal(t, "rollDice(uint256,uint256,address)", entrySignature(abi.Entrys[1]))
	assert.Equal(t, "getBalance(address)", entrySignature(abi.Entrys[2]))
}

func TestGetInputsParser_OverloadedMethods(t *testing.T) {
	contractABI := makeOverloadedABI()

	tests := []struct {
		name       string
		method     string
		wantLen    int
		wantTypes  []string
		wantErr    bool
	}{
		{
			name:      "2-param overload by signature",
			method:    "rollDice(uint256,uint256)",
			wantLen:   2,
			wantTypes: []string{"uint256", "uint256"},
		},
		{
			name:      "3-param overload by signature",
			method:    "rollDice(uint256,uint256,address)",
			wantLen:   3,
			wantTypes: []string{"uint256", "uint256", "address"},
		},
		{
			name:    "plain name returns first match (backward compat)",
			method:  "rollDice",
			wantLen: 2,
		},
		{
			name:    "non-overloaded method by name",
			method:  "getBalance",
			wantLen: 1,
		},
		{
			name:    "non-overloaded method by signature",
			method:  "getBalance(address)",
			wantLen: 1,
		},
		{
			name:    "wrong signature",
			method:  "rollDice(bool)",
			wantErr: true,
		},
		{
			name:    "nonexistent method",
			method:  "doesNotExist",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			inputs, err := GetInputsParser(contractABI, tc.method)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, inputs, tc.wantLen)
			for i, wantType := range tc.wantTypes {
				assert.Equal(t, wantType, inputs[i].Type.String(),
					"input[%d] type mismatch", i)
			}
		})
	}
}

func TestGetParser_OverloadedMethods(t *testing.T) {
	contractABI := makeOverloadedABI()

	tests := []struct {
		name      string
		method    string
		wantLen   int
		wantTypes []string
		wantErr   bool
	}{
		{
			name:      "2-param overload outputs",
			method:    "rollDice(uint256,uint256)",
			wantLen:   1,
			wantTypes: []string{"uint256"},
		},
		{
			name:      "3-param overload outputs",
			method:    "rollDice(uint256,uint256,address)",
			wantLen:   2,
			wantTypes: []string{"uint256", "bool"},
		},
		{
			name:    "plain name returns first match (backward compat)",
			method:  "rollDice",
			wantLen: 1,
		},
		{
			name:    "wrong signature",
			method:  "rollDice(bool)",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outputs, err := GetParser(contractABI, tc.method)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, outputs, tc.wantLen)
			for i, wantType := range tc.wantTypes {
				assert.Equal(t, wantType, outputs[i].Type.String(),
					"output[%d] type mismatch", i)
			}
		})
	}
}

func TestSignature_OverloadedMethodsProduceDifferentSelectors(t *testing.T) {
	sig1 := "rollDice(uint256,uint256)"
	sig2 := "rollDice(uint256,uint256,address)"

	selector1 := hex.EncodeToString(Signature(sig1))
	selector2 := hex.EncodeToString(Signature(sig2))

	assert.NotEqual(t, selector1, selector2,
		"overloaded methods must produce different selectors")
}

func TestMatchEntry(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Name: "transfer",
		Inputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "to", Type: "address"},
			{Name: "value", Type: "uint256"},
		},
	}

	assert.True(t, matchEntry(entry, "transfer"))
	assert.True(t, matchEntry(entry, "transfer(address,uint256)"))
	assert.False(t, matchEntry(entry, "transfer(address)"))
	assert.False(t, matchEntry(entry, "approve"))
	assert.False(t, matchEntry(entry, "approve(address,uint256)"))
}
