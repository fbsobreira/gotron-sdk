package abi

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
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

func TestABIParamArrayUint256FromJSON(t *testing.T) {
	// Issue #120: uint256[] loaded via LoadFromJSON produces []interface{}, not []string
	param, err := LoadFromJSON(`[{"uint256[]": ["100", "200"]}]`)
	require.NoError(t, err)
	b, err := GetPaddedParam(param)
	require.NoError(t, err)
	// offset(32) + length(32) + 2 elements(64) = 128
	assert.Len(t, b, 128, fmt.Sprintf("Wrong length %d/%d", len(b), 128))
}

func TestABIParamSliceUint256(t *testing.T) {
	// Dynamic-length uint256[] with []string input
	b, err := GetPaddedParam([]Param{{"uint256[]": []string{"100", "200"}}})
	require.NoError(t, err)
	assert.Len(t, b, 128, fmt.Sprintf("Wrong length %d/%d", len(b), 128))
}

func TestABIParamArrayUint256HexFromJSON_PR95(t *testing.T) {
	// PR #95: hex uint256[] via JSON triggers same []interface{} issue as #120
	param, err := LoadFromJSON(`[{"uint256[]":["0x8157de19c158b16582821e315285b4dc"]}]`)
	require.NoError(t, err)
	b, err := GetPaddedParam(param)
	require.NoError(t, err)
	assert.Greater(t, len(b), 0, "encoded hex uint256[] should not be empty")
}

func TestABIParamArrayBytesSlice(t *testing.T) {
	// Issue #131: bytes[] (array of dynamic bytes)
	param, err := LoadFromJSON(`[{"bytes[]": ["deadbeef", "cafebabe"]}]`)
	require.NoError(t, err)
	b, err := GetPaddedParam(param)
	require.NoError(t, err)
	// offset(32) + length(32) + 2 element offsets(64) + 2 elements (len+data each, padded)
	// = 32 + 32 + 64 + (32+32) + (32+32) = 256
	assert.Len(t, b, 256, "unexpected encoded length for bytes[]")
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
		name      string
		method    string
		wantLen   int
		wantTypes []string
		wantErr   bool
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
		Type: core.SmartContract_ABI_Entry_Function,
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

func TestGetParser_SkipsNonFunctionEntries(t *testing.T) {
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
			{
				Name: "Transfer",
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

	// Should skip the event and return the function outputs
	outputs, err := GetParser(contractABI, "Transfer")
	require.NoError(t, err)
	assert.Len(t, outputs, 1)
	assert.Equal(t, "bool", outputs[0].Type.String())

	// GetInputsParser should also skip the event
	inputs, err := GetInputsParser(contractABI, "Transfer")
	require.NoError(t, err)
	assert.Len(t, inputs, 2)
}

func TestParseTopicsIntoMap(t *testing.T) {
	// Build a Transfer event: Transfer(address indexed from, address indexed to, uint256 value)
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

	indexed, _, err := GetEventParser(contractABI, "Transfer")
	require.NoError(t, err)
	assert.Len(t, indexed, 2)

	// Create topic bytes (20-byte address padded to 32)
	fromAddr := make([]byte, 32)
	fromAddr[31] = 0x01
	toAddr := make([]byte, 32)
	toAddr[31] = 0x02

	out := make(map[string]interface{})
	err = ParseTopicsIntoMap(out, indexed, [][]byte{fromAddr, toAddr})
	require.NoError(t, err)
	assert.Len(t, out, 2)

	// Verify addresses were converted to TRON format (start with 0x41 prefix)
	fromResult, fromOk := out["from"]
	require.True(t, fromOk)
	fromTronAddr, fromIsAddr := fromResult.(address.Address)
	require.True(t, fromIsAddr, "expected address.Address type")
	assert.Equal(t, byte(0x41), fromTronAddr[0], "TRON address should start with 0x41")
	assert.Equal(t, byte(0x01), fromTronAddr[20], "from address last byte should be 0x01")

	toResult, toOk := out["to"]
	require.True(t, toOk)
	toTronAddr, toIsAddr := toResult.(address.Address)
	require.True(t, toIsAddr, "expected address.Address type")
	assert.Equal(t, byte(0x41), toTronAddr[0], "TRON address should start with 0x41")
	assert.Equal(t, byte(0x02), toTronAddr[20], "to address last byte should be 0x02")
}

func TestParseTopicsIntoMap_NilOut(t *testing.T) {
	err := ParseTopicsIntoMap(nil, eABI.Arguments{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out is nil")
}

func TestGetEventParser(t *testing.T) {
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
			{
				Name: "doSomething",
				Type: core.SmartContract_ABI_Entry_Function,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "x", Type: "uint256"},
				},
			},
		},
	}

	// Should find the event
	indexed, nonIndexed, err := GetEventParser(contractABI, "Transfer")
	require.NoError(t, err)
	assert.Len(t, indexed, 2)
	assert.Len(t, nonIndexed, 1)
	assert.Equal(t, "from", indexed[0].Name)
	assert.Equal(t, "to", indexed[1].Name)
	assert.Equal(t, "value", nonIndexed[0].Name)

	// Should not find a function as event
	_, _, err = GetEventParser(contractABI, "doSomething")
	require.Error(t, err)

	// Should not find nonexistent event
	_, _, err = GetEventParser(contractABI, "Approval")
	require.Error(t, err)
}

func TestEntrySignature_UsesRawTypes(t *testing.T) {
	// The Solidity compiler always emits canonical types (uint256, not uint).
	// entrySignature uses the raw type strings from ABI entries directly,
	// which matches what callers provide in method signatures.
	entry := &core.SmartContract_ABI_Entry{
		Name: "foo",
		Inputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "a", Type: "uint256"},
			{Name: "b", Type: "bool"},
		},
	}
	assert.Equal(t, "foo(uint256,bool)", entrySignature(entry))

	// No inputs produces empty parens
	entryNoInputs := &core.SmartContract_ABI_Entry{
		Name: "bar",
	}
	assert.Equal(t, "bar()", entrySignature(entryNoInputs))
}
