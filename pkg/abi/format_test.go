package abi

import (
	"encoding/json"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatABIEntry_TypeLabels(t *testing.T) {
	tests := []struct {
		name     string
		enumType core.SmartContract_ABI_Entry_EntryType
		want     string
	}{
		{"constructor", core.SmartContract_ABI_Entry_Constructor, "constructor"},
		{"function", core.SmartContract_ABI_Entry_Function, "function"},
		{"event", core.SmartContract_ABI_Entry_Event, "event"},
		{"fallback", core.SmartContract_ABI_Entry_Fallback, "fallback"},
		{"receive", core.SmartContract_ABI_Entry_Receive, "receive"},
		{"error", core.SmartContract_ABI_Entry_Error, "error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &core.SmartContract_ABI_Entry{Type: tt.enumType}
			got := FormatABIEntry(entry)
			assert.Equal(t, tt.want, got["type"])
		})
	}
}

func TestFormatABIEntry_UnknownTypeOmitted(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{Type: core.SmartContract_ABI_Entry_UnknownEntryType}
	got := FormatABIEntry(entry)
	_, exists := got["type"]
	assert.False(t, exists, "type should be omitted for UnknownEntryType")
}

func TestFormatABIEntry_OutOfRangeTypeOmitted(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{Type: core.SmartContract_ABI_Entry_EntryType(99)}
	got := FormatABIEntry(entry)
	_, exists := got["type"]
	assert.False(t, exists, "type should be omitted for out-of-range enum value")
}

func TestFormatABIEntry_StateMutabilityLabels(t *testing.T) {
	tests := []struct {
		name string
		sm   core.SmartContract_ABI_Entry_StateMutabilityType
		want string
	}{
		{"pure", core.SmartContract_ABI_Entry_Pure, "pure"},
		{"view", core.SmartContract_ABI_Entry_View, "view"},
		{"nonpayable", core.SmartContract_ABI_Entry_Nonpayable, "nonpayable"},
		{"payable", core.SmartContract_ABI_Entry_Payable, "payable"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &core.SmartContract_ABI_Entry{
				Type:            core.SmartContract_ABI_Entry_Function,
				StateMutability: tt.sm,
			}
			got := FormatABIEntry(entry)
			assert.Equal(t, tt.want, got["stateMutability"])
		})
	}
}

func TestFormatABIEntry_MutabilityOmittedForEvents(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Type:            core.SmartContract_ABI_Entry_Event,
		StateMutability: core.SmartContract_ABI_Entry_View,
	}
	got := FormatABIEntry(entry)
	_, exists := got["stateMutability"]
	assert.False(t, exists, "stateMutability should be omitted for events")
}

func TestFormatABIEntry_MutabilityOmittedForErrors(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Type:            core.SmartContract_ABI_Entry_Error,
		StateMutability: core.SmartContract_ABI_Entry_Nonpayable,
	}
	got := FormatABIEntry(entry)
	_, exists := got["stateMutability"]
	assert.False(t, exists, "stateMutability should be omitted for error entries")
}

func TestFormatABIEntry_WithInputsAndOutputs(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Name:            "transfer",
		Type:            core.SmartContract_ABI_Entry_Function,
		StateMutability: core.SmartContract_ABI_Entry_Nonpayable,
		Inputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "to", Type: "address"},
			{Name: "value", Type: "uint256"},
		},
		Outputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "", Type: "bool"},
		},
	}

	got := FormatABIEntry(entry)
	assert.Equal(t, "transfer", got["name"])
	assert.Equal(t, "function", got["type"])
	assert.Equal(t, "nonpayable", got["stateMutability"])

	inputs, ok := got["inputs"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, inputs, 2)
	assert.Equal(t, "to", inputs[0]["name"])
	assert.Equal(t, "address", inputs[0]["type"])
	assert.Equal(t, "value", inputs[1]["name"])
	assert.Equal(t, "uint256", inputs[1]["type"])

	outputs, ok := got["outputs"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, outputs, 1)
	assert.Equal(t, "bool", outputs[0]["type"])
}

func TestFormatABIEntry_EventWithIndexed(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Name: "Transfer",
		Type: core.SmartContract_ABI_Entry_Event,
		Inputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "from", Type: "address", Indexed: true},
			{Name: "to", Type: "address", Indexed: true},
			{Name: "value", Type: "uint256"},
		},
	}

	got := FormatABIEntry(entry)
	assert.Equal(t, "event", got["type"])

	inputs, ok := got["inputs"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, inputs, 3)
	assert.Equal(t, true, inputs[0]["indexed"])
	assert.Equal(t, true, inputs[1]["indexed"])
	_, hasIndexed := inputs[2]["indexed"]
	assert.False(t, hasIndexed, "non-indexed param should omit indexed key")
}

// ---------------------------------------------------------------------------
// Canonical per-entry-type field shapes
// ---------------------------------------------------------------------------

func TestFormatABIEntry_ConstructorShape(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Type:            core.SmartContract_ABI_Entry_Constructor,
		StateMutability: core.SmartContract_ABI_Entry_Nonpayable,
		Inputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "supply", Type: "uint256"},
		},
	}
	got := FormatABIEntry(entry)
	assert.Equal(t, "constructor", got["type"])
	assert.Equal(t, "nonpayable", got["stateMutability"])
	assert.NotNil(t, got["inputs"])
	_, hasOutputs := got["outputs"]
	assert.False(t, hasOutputs, "constructor should not have outputs")
}

func TestFormatABIEntry_FallbackShape(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Type:            core.SmartContract_ABI_Entry_Fallback,
		StateMutability: core.SmartContract_ABI_Entry_Payable,
		Payable:         true,
	}
	got := FormatABIEntry(entry)
	assert.Equal(t, "fallback", got["type"])
	assert.Equal(t, "payable", got["stateMutability"])
	_, hasName := got["name"]
	assert.False(t, hasName, "fallback should not have name")
	_, hasInputs := got["inputs"]
	assert.False(t, hasInputs, "fallback should not have inputs")
	_, hasOutputs := got["outputs"]
	assert.False(t, hasOutputs, "fallback should not have outputs")
}

func TestFormatABIEntry_ReceiveShape(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Type:            core.SmartContract_ABI_Entry_Receive,
		StateMutability: core.SmartContract_ABI_Entry_Payable,
	}
	got := FormatABIEntry(entry)
	assert.Equal(t, "receive", got["type"])
	assert.Equal(t, "payable", got["stateMutability"])
	_, hasName := got["name"]
	assert.False(t, hasName, "receive should not have name")
	_, hasInputs := got["inputs"]
	assert.False(t, hasInputs, "receive should not have inputs")
	_, hasOutputs := got["outputs"]
	assert.False(t, hasOutputs, "receive should not have outputs")
}

func TestFormatABIEntry_EventShape(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Name: "Transfer",
		Type: core.SmartContract_ABI_Entry_Event,
	}
	got := FormatABIEntry(entry)
	assert.Equal(t, "event", got["type"])
	assert.Equal(t, "Transfer", got["name"])
	_, hasMut := got["stateMutability"]
	assert.False(t, hasMut, "event should not have stateMutability")
	_, hasOutputs := got["outputs"]
	assert.False(t, hasOutputs, "event should not have outputs")
}

func TestFormatABIEntry_ErrorShape(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Name: "InsufficientBalance",
		Type: core.SmartContract_ABI_Entry_Error,
		Inputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "balance", Type: "uint256"},
		},
	}
	got := FormatABIEntry(entry)
	assert.Equal(t, "error", got["type"])
	assert.Equal(t, "InsufficientBalance", got["name"])
	_, hasMut := got["stateMutability"]
	assert.False(t, hasMut, "error should not have stateMutability")

	inputs, ok := got["inputs"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, inputs, 1)

	_, hasOutputs := got["outputs"]
	assert.False(t, hasOutputs, "error should not have outputs per Solidity ABI spec")
}

// ---------------------------------------------------------------------------
// Boolean flags
// ---------------------------------------------------------------------------

func TestFormatABIEntry_BooleanFlags(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Type:      core.SmartContract_ABI_Entry_Event,
		Anonymous: true,
		Payable:   true,
		Constant:  true,
	}

	got := FormatABIEntry(entry)
	assert.Equal(t, true, got["anonymous"])
	assert.Equal(t, true, got["payable"])
	assert.Equal(t, true, got["constant"])
}

func TestFormatABIEntry_BooleanFlagsOmittedWhenFalse(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Type: core.SmartContract_ABI_Entry_Function,
	}

	got := FormatABIEntry(entry)
	_, hasAnon := got["anonymous"]
	_, hasPay := got["payable"]
	_, hasConst := got["constant"]
	assert.False(t, hasAnon)
	assert.False(t, hasPay)
	assert.False(t, hasConst)
}

// ---------------------------------------------------------------------------
// Nil guards
// ---------------------------------------------------------------------------

func TestFormatABIEntry_NilEntry(t *testing.T) {
	got := FormatABIEntry(nil)
	require.NotNil(t, got)
	assert.Empty(t, got)
}

func TestFormatABI_NilABI(t *testing.T) {
	result := FormatABI((*core.SmartContract_ABI)(nil))
	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestFormatABI_EmptyABI(t *testing.T) {
	result := FormatABI(&core.SmartContract_ABI{})
	require.NotNil(t, result)
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// FormatABI
// ---------------------------------------------------------------------------

func TestFormatABI(t *testing.T) {
	abi := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name:            "balanceOf",
				Type:            core.SmartContract_ABI_Entry_Function,
				StateMutability: core.SmartContract_ABI_Entry_View,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "owner", Type: "address"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "", Type: "uint256"},
				},
			},
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

	result := FormatABI(abi)
	require.Len(t, result, 2)

	assert.Equal(t, "balanceOf", result[0]["name"])
	assert.Equal(t, "function", result[0]["type"])
	assert.Equal(t, "view", result[0]["stateMutability"])

	assert.Equal(t, "Transfer", result[1]["name"])
	assert.Equal(t, "event", result[1]["type"])
}

// ---------------------------------------------------------------------------
// JSON round-trip
// ---------------------------------------------------------------------------

func TestFormatABIEntry_JSONRoundTrip(t *testing.T) {
	entry := &core.SmartContract_ABI_Entry{
		Name:            "transfer",
		Type:            core.SmartContract_ABI_Entry_Function,
		StateMutability: core.SmartContract_ABI_Entry_Nonpayable,
		Inputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "to", Type: "address"},
			{Name: "value", Type: "uint256"},
		},
		Outputs: []*core.SmartContract_ABI_Entry_Param{
			{Name: "", Type: "bool"},
		},
	}

	data, err := json.Marshal(FormatABIEntry(entry))
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	assert.Equal(t, "function", parsed["type"])
	assert.Equal(t, "transfer", parsed["name"])
	assert.Equal(t, "nonpayable", parsed["stateMutability"])
}

func TestFormatABI_JSONRoundTrip(t *testing.T) {
	abi := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name:            "balanceOf",
				Type:            core.SmartContract_ABI_Entry_Function,
				StateMutability: core.SmartContract_ABI_Entry_View,
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "owner", Type: "address"},
				},
				Outputs: []*core.SmartContract_ABI_Entry_Param{
					{Name: "", Type: "uint256"},
				},
			},
			{
				Type:            core.SmartContract_ABI_Entry_Receive,
				StateMutability: core.SmartContract_ABI_Entry_Payable,
			},
		},
	}

	data, err := json.Marshal(FormatABI(abi))
	require.NoError(t, err)

	var parsed []map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed, 2)

	assert.Equal(t, "function", parsed[0]["type"])
	assert.Equal(t, "balanceOf", parsed[0]["name"])

	assert.Equal(t, "receive", parsed[1]["type"])
	_, hasName := parsed[1]["name"]
	assert.False(t, hasName, "receive should not have name in JSON")
}
