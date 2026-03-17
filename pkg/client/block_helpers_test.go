package client_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testWitnessBase58 is a known valid TRON address used as test fixture.
const testWitnessBase58 = "TPL66VK2gCXNCD7EJg9pgJRfqcRazjhUZY"

// testWitnessBytes returns a fresh copy of the witness address bytes for each test.
func testWitnessBytes(t *testing.T) []byte {
	t.Helper()
	addr, err := address.Base58ToAddress(testWitnessBase58)
	require.NoError(t, err)
	require.True(t, addr.IsValid())
	return addr.Bytes()
}

func TestBlockExtentionWitnessAddress(t *testing.T) {
	wb := testWitnessBytes(t)

	tests := []struct {
		name  string
		block *api.BlockExtention
		want  address.Address
	}{
		{
			name:  "nil block",
			block: nil,
			want:  nil,
		},
		{
			name:  "nil block header",
			block: &api.BlockExtention{},
			want:  nil,
		},
		{
			name:  "nil raw data",
			block: &api.BlockExtention{BlockHeader: &core.BlockHeader{}},
			want:  nil,
		},
		{
			name: "empty witness address",
			block: &api.BlockExtention{BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{WitnessAddress: []byte{}},
			}},
			want: nil,
		},
		{
			name: "invalid witness address (wrong length)",
			block: &api.BlockExtention{BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{WitnessAddress: []byte{0x41, 0x01, 0x02}},
			}},
			want: nil,
		},
		{
			name: "valid witness address",
			block: &api.BlockExtention{BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{WitnessAddress: wb},
			}},
			want: address.Address(wb),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.BlockExtentionWitnessAddress(tt.block)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBlockExtentionWitnessBase58(t *testing.T) {
	wb := testWitnessBytes(t)

	tests := []struct {
		name  string
		block *api.BlockExtention
		want  string
	}{
		{
			name:  "nil block",
			block: nil,
			want:  "",
		},
		{
			name:  "nil block header",
			block: &api.BlockExtention{},
			want:  "",
		},
		{
			name:  "nil raw data",
			block: &api.BlockExtention{BlockHeader: &core.BlockHeader{}},
			want:  "",
		},
		{
			name: "valid witness address",
			block: &api.BlockExtention{BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{WitnessAddress: wb},
			}},
			want: testWitnessBase58,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.BlockExtentionWitnessBase58(tt.block)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBlockWitnessAddress(t *testing.T) {
	wb := testWitnessBytes(t)

	tests := []struct {
		name  string
		block *core.Block
		want  address.Address
	}{
		{
			name:  "nil block",
			block: nil,
			want:  nil,
		},
		{
			name:  "nil block header",
			block: &core.Block{},
			want:  nil,
		},
		{
			name:  "nil raw data",
			block: &core.Block{BlockHeader: &core.BlockHeader{}},
			want:  nil,
		},
		{
			name: "empty witness address",
			block: &core.Block{BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{WitnessAddress: []byte{}},
			}},
			want: nil,
		},
		{
			name: "invalid witness address (wrong length)",
			block: &core.Block{BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{WitnessAddress: []byte{0x41, 0x01, 0x02}},
			}},
			want: nil,
		},
		{
			name: "valid witness address",
			block: &core.Block{BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{WitnessAddress: wb},
			}},
			want: address.Address(wb),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.BlockWitnessAddress(tt.block)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBlockWitnessBase58(t *testing.T) {
	wb := testWitnessBytes(t)

	tests := []struct {
		name  string
		block *core.Block
		want  string
	}{
		{
			name:  "nil block",
			block: nil,
			want:  "",
		},
		{
			name:  "nil block header",
			block: &core.Block{},
			want:  "",
		},
		{
			name:  "nil raw data",
			block: &core.Block{BlockHeader: &core.BlockHeader{}},
			want:  "",
		},
		{
			name: "valid witness address",
			block: &core.Block{BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{WitnessAddress: wb},
			}},
			want: testWitnessBase58,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.BlockWitnessBase58(tt.block)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBlockExtentionWitnessAddress_does_not_alias_protobuf(t *testing.T) {
	wb := testWitnessBytes(t)
	block := &api.BlockExtention{BlockHeader: &core.BlockHeader{
		RawData: &core.BlockHeaderRaw{WitnessAddress: wb},
	}}

	addr := client.BlockExtentionWitnessAddress(block)
	require.NotNil(t, addr)

	// Mutate the returned address; the protobuf field must be unchanged.
	original := make([]byte, len(wb))
	copy(original, wb)
	addr[0] = 0x00

	assert.Equal(t, original, []byte(block.BlockHeader.RawData.WitnessAddress),
		"mutating the returned Address must not affect the protobuf message")
}
