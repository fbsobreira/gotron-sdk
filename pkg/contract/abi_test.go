package contract_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONtoABI_ERC20(t *testing.T) {
	erc20ABI := `[
		{
			"constant": true,
			"inputs": [{"name": "_owner", "type": "address"}],
			"name": "balanceOf",
			"outputs": [{"name": "balance", "type": "uint256"}],
			"type": "function",
			"stateMutability": "view"
		},
		{
			"constant": false,
			"inputs": [
				{"name": "_to", "type": "address"},
				{"name": "_value", "type": "uint256"}
			],
			"name": "transfer",
			"outputs": [{"name": "", "type": "bool"}],
			"type": "function",
			"stateMutability": "nonpayable"
		},
		{
			"anonymous": false,
			"inputs": [
				{"indexed": true, "name": "from", "type": "address"},
				{"indexed": true, "name": "to", "type": "address"},
				{"indexed": false, "name": "value", "type": "uint256"}
			],
			"name": "Transfer",
			"type": "event"
		}
	]`

	abi, err := contract.JSONtoABI(erc20ABI)
	require.NoError(t, err)
	require.NotNil(t, abi)
	require.Len(t, abi.Entrys, 3)

	t.Run("balanceOf function", func(t *testing.T) {
		entry := abi.Entrys[0]
		assert.Equal(t, "balanceOf", entry.Name)
		assert.True(t, entry.Constant)
		assert.Equal(t, core.SmartContract_ABI_Entry_Function, entry.Type)
		assert.Equal(t, core.SmartContract_ABI_Entry_View, entry.StateMutability)
		require.Len(t, entry.Inputs, 1)
		assert.Equal(t, "_owner", entry.Inputs[0].Name)
		assert.Equal(t, "address", entry.Inputs[0].Type)
		require.Len(t, entry.Outputs, 1)
		assert.Equal(t, "uint256", entry.Outputs[0].Type)
	})

	t.Run("transfer function", func(t *testing.T) {
		entry := abi.Entrys[1]
		assert.Equal(t, "transfer", entry.Name)
		assert.False(t, entry.Constant)
		assert.Equal(t, core.SmartContract_ABI_Entry_Function, entry.Type)
		assert.Equal(t, core.SmartContract_ABI_Entry_Nonpayable, entry.StateMutability)
		require.Len(t, entry.Inputs, 2)
		assert.Equal(t, "_to", entry.Inputs[0].Name)
		assert.Equal(t, "_value", entry.Inputs[1].Name)
		require.Len(t, entry.Outputs, 1)
		assert.Equal(t, "bool", entry.Outputs[0].Type)
	})

	t.Run("Transfer event", func(t *testing.T) {
		entry := abi.Entrys[2]
		assert.Equal(t, "Transfer", entry.Name)
		assert.False(t, entry.Anonymous)
		assert.Equal(t, core.SmartContract_ABI_Entry_Event, entry.Type)
		require.Len(t, entry.Inputs, 3)
		assert.True(t, entry.Inputs[0].Indexed)
		assert.True(t, entry.Inputs[1].Indexed)
		assert.False(t, entry.Inputs[2].Indexed)
	})
}

func TestJSONtoABI_EntryTypes(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantType core.SmartContract_ABI_Entry_EntryType
	}{
		{
			name:     "constructor",
			json:     `[{"type": "constructor", "inputs": [{"name": "supply", "type": "uint256"}]}]`,
			wantType: core.SmartContract_ABI_Entry_Constructor,
		},
		{
			name:     "fallback",
			json:     `[{"type": "fallback", "payable": true, "stateMutability": "payable"}]`,
			wantType: core.SmartContract_ABI_Entry_Fallback,
		},
		{
			name:     "unknown type",
			json:     `[{"type": "receive", "name": "test"}]`,
			wantType: core.SmartContract_ABI_Entry_UnknownEntryType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abi, err := contract.JSONtoABI(tt.json)
			require.NoError(t, err)
			require.Len(t, abi.Entrys, 1)
			assert.Equal(t, tt.wantType, abi.Entrys[0].Type)
		})
	}
}

func TestJSONtoABI_StateMutability(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantState core.SmartContract_ABI_Entry_StateMutabilityType
	}{
		{
			name:      "pure",
			json:      `[{"type": "function", "name": "add", "stateMutability": "pure"}]`,
			wantState: core.SmartContract_ABI_Entry_Pure,
		},
		{
			name:      "view",
			json:      `[{"type": "function", "name": "get", "stateMutability": "view"}]`,
			wantState: core.SmartContract_ABI_Entry_View,
		},
		{
			name:      "nonpayable",
			json:      `[{"type": "function", "name": "set", "stateMutability": "nonpayable"}]`,
			wantState: core.SmartContract_ABI_Entry_Nonpayable,
		},
		{
			name:      "payable",
			json:      `[{"type": "function", "name": "deposit", "stateMutability": "payable"}]`,
			wantState: core.SmartContract_ABI_Entry_Payable,
		},
		{
			name:      "unknown mutability",
			json:      `[{"type": "function", "name": "test", "stateMutability": "bogus"}]`,
			wantState: core.SmartContract_ABI_Entry_UnknownMutabilityType,
		},
		{
			name:      "legacy constant without stateMutability",
			json:      `[{"type": "function", "name": "legacyView", "constant": true}]`,
			wantState: core.SmartContract_ABI_Entry_View,
		},
		{
			name:      "legacy payable without stateMutability",
			json:      `[{"type": "function", "name": "legacyPay", "payable": true}]`,
			wantState: core.SmartContract_ABI_Entry_Payable,
		},
		{
			name:      "legacy default nonpayable",
			json:      `[{"type": "function", "name": "legacySet"}]`,
			wantState: core.SmartContract_ABI_Entry_Nonpayable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abi, err := contract.JSONtoABI(tt.json)
			require.NoError(t, err)
			require.Len(t, abi.Entrys, 1)
			assert.Equal(t, tt.wantState, abi.Entrys[0].StateMutability)
		})
	}
}

func TestJSONtoABI_EmptyArray(t *testing.T) {
	abi, err := contract.JSONtoABI(`[]`)
	require.NoError(t, err)
	require.NotNil(t, abi)
	assert.Empty(t, abi.Entrys)
}

func TestJSONtoABI_AnonymousEvent(t *testing.T) {
	json := `[{"type": "event", "name": "Log", "anonymous": true, "inputs": []}]`
	abi, err := contract.JSONtoABI(json)
	require.NoError(t, err)
	require.Len(t, abi.Entrys, 1)
	assert.True(t, abi.Entrys[0].Anonymous)
}

func TestJSONtoABI_PayableFallback(t *testing.T) {
	json := `[{"type": "fallback", "payable": true, "stateMutability": "payable"}]`
	abi, err := contract.JSONtoABI(json)
	require.NoError(t, err)
	require.Len(t, abi.Entrys, 1)
	assert.True(t, abi.Entrys[0].Payable)
	assert.Equal(t, core.SmartContract_ABI_Entry_Payable, abi.Entrys[0].StateMutability)
}

func TestJSONtoABI_Errors(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"invalid JSON", `{not json}`},
		{"JSON object instead of array", `{"type": "function"}`},
		{"empty string", ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			abi, err := contract.JSONtoABI(tt.json)
			assert.Error(t, err)
			assert.Nil(t, abi)
		})
	}
}

func TestJSONtoABI_MinimalFunction(t *testing.T) {
	json := `[{"type": "function", "name": "doSomething", "inputs": [], "outputs": []}]`
	abi, err := contract.JSONtoABI(json)
	require.NoError(t, err)
	require.Len(t, abi.Entrys, 1)
	assert.Equal(t, "doSomething", abi.Entrys[0].Name)
	assert.Empty(t, abi.Entrys[0].Inputs)
	assert.Empty(t, abi.Entrys[0].Outputs)
}
