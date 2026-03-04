package common_test

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBytesToHash(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantHex  string
		wantSize int
	}{
		{
			name:     "exact 32 bytes",
			input:    bytes32(0x01),
			wantHex:  "0101010101010101010101010101010101010101010101010101010101010101",
			wantSize: 32,
		},
		{
			name:     "shorter than 32 bytes right-aligned",
			input:    []byte{0xde, 0xad},
			wantHex:  "000000000000000000000000000000000000000000000000000000000000dead",
			wantSize: 32,
		},
		{
			name:     "longer than 32 bytes crops from left",
			input:    append([]byte{0xff, 0xee}, bytes32(0xab)...),
			wantHex:  "abababababababababababababababababababababababababababababababab",
			wantSize: 32,
		},
		{
			name:     "empty input gives zero hash",
			input:    []byte{},
			wantHex:  "0000000000000000000000000000000000000000000000000000000000000000",
			wantSize: 32,
		},
		{
			name:     "nil input gives zero hash",
			input:    nil,
			wantHex:  "0000000000000000000000000000000000000000000000000000000000000000",
			wantSize: 32,
		},
		{
			name:     "single byte",
			input:    []byte{0x42},
			wantHex:  "0000000000000000000000000000000000000000000000000000000000000042",
			wantSize: 32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := common.BytesToHash(tt.input)
			got := hex.EncodeToString(h.Bytes())
			assert.Equal(t, tt.wantHex, got)
			assert.Len(t, h.Bytes(), tt.wantSize)
		})
	}
}

func TestBigToHash(t *testing.T) {
	tests := []struct {
		name    string
		input   *big.Int
		wantHex string
	}{
		{
			name:    "zero",
			input:   big.NewInt(0),
			wantHex: "0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:    "small number",
			input:   big.NewInt(255),
			wantHex: "00000000000000000000000000000000000000000000000000000000000000ff",
		},
		{
			name:    "large number",
			input:   big.NewInt(0x1234567890),
			wantHex: "0000000000000000000000000000000000000000000000000000001234567890",
		},
		{
			name:    "max 256-bit value",
			input:   new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)),
			wantHex: "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := common.BigToHash(tt.input)
			got := hex.EncodeToString(h.Bytes())
			assert.Equal(t, tt.wantHex, got)
		})
	}
}

func TestHexToHash(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantHex string
		wantErr bool
	}{
		{
			name:    "valid 32-byte hex with 0x prefix",
			input:   "0x000000000000000000000000000000000000000000000000000000000000002a",
			wantHex: "000000000000000000000000000000000000000000000000000000000000002a",
			wantErr: false,
		},
		{
			name:    "valid short hex with prefix",
			input:   "0xdeadbeef",
			wantHex: "00000000000000000000000000000000000000000000000000000000deadbeef",
			wantErr: false,
		},
		{
			name:    "valid hex without prefix",
			input:   "ff",
			wantHex: "00000000000000000000000000000000000000000000000000000000000000ff",
			wantErr: false,
		},
		{
			name:    "empty string returns error",
			input:   "",
			wantHex: "",
			wantErr: true,
		},
		{
			name:    "invalid hex returns error",
			input:   "0xzzzz",
			wantHex: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := common.HexToHash(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got := hex.EncodeToString(h.Bytes())
			assert.Equal(t, tt.wantHex, got)
		})
	}
}

func TestHashBytes(t *testing.T) {
	h := common.BytesToHash([]byte{0xab, 0xcd})
	b := h.Bytes()
	assert.Len(t, b, 32)
	assert.Equal(t, byte(0xab), b[30])
	assert.Equal(t, byte(0xcd), b[31])
}

func TestHashBig(t *testing.T) {
	tests := []struct {
		name  string
		input *big.Int
	}{
		{
			name:  "zero",
			input: big.NewInt(0),
		},
		{
			name:  "small value",
			input: big.NewInt(42),
		},
		{
			name:  "large value",
			input: new(big.Int).SetBytes([]byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := common.BigToHash(tt.input)
			got := h.Big()
			assert.Equal(t, 0, tt.input.Cmp(got), "round-trip big.Int should match")
		})
	}
}

func TestHashHex(t *testing.T) {
	h := common.BytesToHash([]byte{0xde, 0xad})
	hexStr := h.Hex()
	assert.True(t, strings.HasPrefix(hexStr, "0x"), "Hex() should have 0x prefix")
	// 32 bytes = 64 hex chars + "0x" = 66
	assert.Len(t, hexStr, 66)
}

func TestHashString(t *testing.T) {
	h := common.BytesToHash([]byte{0xab})
	// String() should return the same as Hex()
	assert.Equal(t, h.Hex(), h.String())
}

func TestHashTerminalString(t *testing.T) {
	h := common.BytesToHash([]byte{0x01, 0x02, 0x03})
	ts := h.TerminalString()
	// TerminalString shows first 3 bytes and last 3 bytes separated by "..."
	assert.Contains(t, ts, "\u2026", "should contain ellipsis character")
	assert.NotEmpty(t, ts)
}

func TestHashSetBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantLast byte
	}{
		{
			name:     "short input right-aligns",
			input:    []byte{0x42},
			wantLast: 0x42,
		},
		{
			name:     "exact 32 bytes",
			input:    bytes32(0xaa),
			wantLast: 0xaa,
		},
		{
			name:     "longer than 32 bytes crops left",
			input:    append([]byte{0xff}, bytes32(0xbb)...),
			wantLast: 0xbb,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var h common.Hash
			h.SetBytes(tt.input)
			assert.Equal(t, tt.wantLast, h.Bytes()[31])
			assert.Len(t, h.Bytes(), 32)
		})
	}
}

func TestKeccak256(t *testing.T) {
	// Known Keccak-256 test vectors
	tests := []struct {
		name    string
		input   []byte
		wantHex string
	}{
		{
			name:    "empty input",
			input:   []byte{},
			wantHex: "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470",
		},
		{
			name:    "hello world",
			input:   []byte("hello world"),
			wantHex: "47173285a8d7341e5e972fc677286384f802f8ef42a5ec5f03bbfa254cb01fad",
		},
		{
			name:    "single zero byte",
			input:   []byte{0x00},
			wantHex: "bc36789e7a1e281436464229828f817d6612f7b477d66591ff96a9e064bcc98a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.Keccak256(tt.input)
			got := hex.EncodeToString(result)
			assert.Equal(t, tt.wantHex, got)
			assert.Len(t, result, 32, "Keccak256 output must be 32 bytes")
		})
	}
}

func TestKeccak256Deterministic(t *testing.T) {
	input := []byte("deterministic test")
	result1 := common.Keccak256(input)
	result2 := common.Keccak256(input)
	assert.Equal(t, result1, result2, "same input must produce same hash")
}

func TestKeccak256DifferentInputs(t *testing.T) {
	h1 := common.Keccak256([]byte("input1"))
	h2 := common.Keccak256([]byte("input2"))
	assert.NotEqual(t, h1, h2, "different inputs should produce different hashes")
}

// bytes32 creates a 32-byte slice filled with the given value.
func bytes32(val byte) []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = val
	}
	return b
}
