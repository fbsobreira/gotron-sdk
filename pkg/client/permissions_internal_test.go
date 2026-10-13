package client

import (
	"encoding/hex"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPermissionOperations_MakePermissionRoundTrip checks that decoding a
// bitmap and encoding the result with makePermission (the encoder
// UpdateAccountPermissionCtx uses) gives back the same bytes, so no operation
// is gained or lost in the round trip.
func TestPermissionOperations_MakePermissionRoundTrip(t *testing.T) {
	allNamed := make([]int, 0, len(core.Transaction_Contract_ContractType_name))
	for v := range core.Transaction_Contract_ContractType_name {
		allNamed = append(allNamed, int(v))
	}

	tests := []struct {
		name   string
		bitmap []byte
	}{
		{name: "none", bitmap: nil},
		{name: "every named type", bitmap: bitmapOf(allNamed...)},
		{name: "single high bit", bitmap: bitmapOf(59)},
		{name: "single low bit", bitmap: bitmapOf(0)},
		{name: "byte boundaries", bitmap: bitmapOf(6, 8, 15, 16, 31, 32)},
		{name: "wide set across many bytes", bitmap: bitmapOf(0, 1, 2, 3, 4, 5, 6, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 30, 31, 33, 41, 42, 43, 44, 45, 46, 48, 49, 52, 53, 54, 55, 56, 57, 58, 59)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops, err := PermissionOperations(tt.bitmap)
			require.NoError(t, err)

			p, err := makePermission("active", core.Permission_Active, 2, 1, ops,
				map[string]int64{"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b": 1})
			require.NoError(t, err)
			assert.Equal(t, tt.bitmap, p.GetOperations())
		})
	}
}

// TestPermissionOperations_NileDefaultActive decodes the default active
// operations bitmap Nile reports (getActiveDefaultOperations) and checks that
// makePermission re-encodes it byte for byte.
func TestPermissionOperations_NileDefaultActive(t *testing.T) {
	prefix, err := hex.DecodeString("7fff1fc0033efb07")
	require.NoError(t, err)
	bitmap := make([]byte, 32)
	copy(bitmap, prefix)

	ops, err := PermissionOperations(bitmap)
	require.NoError(t, err)

	for _, bit := range []int32{20, 32, 51} {
		name, ok := core.Transaction_Contract_ContractType_name[bit]
		require.True(t, ok, "bit %d has no contract type name", bit)
		assert.True(t, ops[name], "bit %d (%s) must be decoded", bit, name)
	}
	assert.True(t, ops["CustomContract"])
	assert.True(t, ops["GetContract"])
	assert.True(t, ops["ShieldedTransferContract"])

	p, err := makePermission("active", core.Permission_Active, 2, 1, ops,
		map[string]int64{"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b": 1})
	require.NoError(t, err)
	assert.Equal(t, bitmap, p.GetOperations())
}

func bitmapOf(bits ...int) []byte {
	b := make([]byte, 32)
	for _, n := range bits {
		b[n/8] |= 1 << (n % 8)
	}
	return b
}
