package common_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBytesToHexString(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty bytes",
			input: []byte{},
			want:  "0x",
		},
		{
			name:  "single zero byte",
			input: []byte{0x00},
			want:  "0x00",
		},
		{
			name:  "single byte",
			input: []byte{0xab},
			want:  "0xab",
		},
		{
			name:  "multiple bytes",
			input: []byte{0xde, 0xad, 0xbe, 0xef},
			want:  "0xdeadbeef",
		},
		{
			name:  "all zeros",
			input: []byte{0x00, 0x00, 0x00},
			want:  "0x000000",
		},
		{
			name:  "all 0xff",
			input: []byte{0xff, 0xff},
			want:  "0xffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.BytesToHexString(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHexStringToBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:    "empty string returns error",
			input:   "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "valid hex without prefix",
			input:   "deadbeef",
			want:    []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr: false,
		},
		{
			name:    "valid hex with 0x prefix",
			input:   "0xdeadbeef",
			want:    []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr: false,
		},
		{
			name:    "single byte with prefix",
			input:   "0xff",
			want:    []byte{0xff},
			wantErr: false,
		},
		{
			name:    "single byte without prefix",
			input:   "ab",
			want:    []byte{0xab},
			wantErr: false,
		},
		{
			name:    "odd-length hex without prefix returns error",
			input:   "abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid characters",
			input:   "zzzz",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "all zeros",
			input:   "0x0000",
			want:    []byte{0x00, 0x00},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := common.HexStringToBytes(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToHex(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty slice returns 0x0",
			input: []byte{},
			want:  "0x0",
		},
		{
			name:  "nil slice returns 0x0",
			input: nil,
			want:  "0x0",
		},
		{
			name:  "single byte",
			input: []byte{0xab},
			want:  "0xab",
		},
		{
			name:  "multiple bytes",
			input: []byte{0xde, 0xad, 0xbe, 0xef},
			want:  "0xdeadbeef",
		},
		{
			name:  "leading zero byte",
			input: []byte{0x00, 0x01},
			want:  "0x0001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.ToHex(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToHexArray(t *testing.T) {
	tests := []struct {
		name  string
		input [][]byte
		want  []string
	}{
		{
			name:  "empty array",
			input: [][]byte{},
			want:  []string{},
		},
		{
			name:  "single element",
			input: [][]byte{{0xab, 0xcd}},
			want:  []string{"0xabcd"},
		},
		{
			name:  "multiple elements",
			input: [][]byte{{0xab}, {0xcd, 0xef}, {}},
			want:  []string{"0xab", "0xcdef", "0x0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.ToHexArray(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFromHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:    "hex with 0x prefix",
			input:   "0xdeadbeef",
			want:    []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr: false,
		},
		{
			name:    "hex with 0X prefix",
			input:   "0Xdeadbeef",
			want:    []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr: false,
		},
		{
			name:    "hex without prefix",
			input:   "deadbeef",
			want:    []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr: false,
		},
		{
			name:    "odd-length hex gets padded",
			input:   "0x1",
			want:    []byte{0x01},
			wantErr: false,
		},
		{
			name:    "odd-length without prefix gets padded",
			input:   "abc",
			want:    []byte{0x0a, 0xbc},
			wantErr: false,
		},
		{
			name:    "empty string after prefix",
			input:   "0x",
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "invalid hex characters",
			input:   "0xzzzz",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "single byte",
			input:   "0xff",
			want:    []byte{0xff},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := common.FromHex(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHas0xPrefix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "lowercase 0x prefix",
			input: "0xdeadbeef",
			want:  true,
		},
		{
			name:  "uppercase 0X prefix",
			input: "0Xdeadbeef",
			want:  true,
		},
		{
			name:  "no prefix",
			input: "deadbeef",
			want:  false,
		},
		{
			name:  "empty string",
			input: "",
			want:  false,
		},
		{
			name:  "just 0x",
			input: "0x",
			want:  true,
		},
		{
			name:  "single character",
			input: "0",
			want:  false,
		},
		{
			name:  "starts with 0 but not x",
			input: "0a1234",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.Has0xPrefix(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsHex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid even-length lowercase hex",
			input: "deadbeef",
			want:  true,
		},
		{
			name:  "valid even-length uppercase hex",
			input: "DEADBEEF",
			want:  true,
		},
		{
			name:  "valid mixed case",
			input: "DeAdBeEf",
			want:  true,
		},
		{
			name:  "odd-length is invalid",
			input: "abc",
			want:  false,
		},
		{
			name:  "empty string is valid",
			input: "",
			want:  true,
		},
		{
			name:  "contains invalid character",
			input: "deadbezf",
			want:  false,
		},
		{
			name:  "with 0x prefix is invalid (x is not hex)",
			input: "0xdead",
			want:  false,
		},
		{
			name:  "all digits",
			input: "0123456789",
			want:  true,
		},
		{
			name:  "single valid pair",
			input: "ff",
			want:  true,
		},
		{
			name:  "spaces are invalid",
			input: "de ad",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.IsHex(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCopyBytes(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		result := common.CopyBytes(nil)
		assert.Nil(t, result)
	})

	t.Run("empty slice returns empty slice", func(t *testing.T) {
		result := common.CopyBytes([]byte{})
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("copies data correctly", func(t *testing.T) {
		original := []byte{0xde, 0xad, 0xbe, 0xef}
		copied := common.CopyBytes(original)
		assert.Equal(t, original, copied)
	})

	t.Run("modifying copy does not affect original", func(t *testing.T) {
		original := []byte{0x01, 0x02, 0x03}
		copied := common.CopyBytes(original)
		copied[0] = 0xff
		assert.Equal(t, byte(0x01), original[0], "original should not be modified")
		assert.Equal(t, byte(0xff), copied[0])
	})
}

func TestLeftPadBytes(t *testing.T) {
	tests := []struct {
		name   string
		slice  []byte
		length int
		want   []byte
	}{
		{
			name:   "pad short slice",
			slice:  []byte{0x01, 0x02},
			length: 4,
			want:   []byte{0x00, 0x00, 0x01, 0x02},
		},
		{
			name:   "slice already at target length",
			slice:  []byte{0x01, 0x02},
			length: 2,
			want:   []byte{0x01, 0x02},
		},
		{
			name:   "slice longer than target returns original",
			slice:  []byte{0x01, 0x02, 0x03},
			length: 2,
			want:   []byte{0x01, 0x02, 0x03},
		},
		{
			name:   "empty slice padded",
			slice:  []byte{},
			length: 3,
			want:   []byte{0x00, 0x00, 0x00},
		},
		{
			name:   "single byte padded to 4",
			slice:  []byte{0xff},
			length: 4,
			want:   []byte{0x00, 0x00, 0x00, 0xff},
		},
		{
			name:   "zero target length",
			slice:  []byte{0x01},
			length: 0,
			want:   []byte{0x01},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.LeftPadBytes(tt.slice, tt.length)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRightPadBytes(t *testing.T) {
	tests := []struct {
		name   string
		slice  []byte
		length int
		want   []byte
	}{
		{
			name:   "pad short slice",
			slice:  []byte{0x01, 0x02},
			length: 4,
			want:   []byte{0x01, 0x02, 0x00, 0x00},
		},
		{
			name:   "slice already at target length",
			slice:  []byte{0x01, 0x02},
			length: 2,
			want:   []byte{0x01, 0x02},
		},
		{
			name:   "slice longer than target returns original",
			slice:  []byte{0x01, 0x02, 0x03},
			length: 2,
			want:   []byte{0x01, 0x02, 0x03},
		},
		{
			name:   "empty slice padded",
			slice:  []byte{},
			length: 3,
			want:   []byte{0x00, 0x00, 0x00},
		},
		{
			name:   "single byte padded to 4",
			slice:  []byte{0xff},
			length: 4,
			want:   []byte{0xff, 0x00, 0x00, 0x00},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.RightPadBytes(tt.slice, tt.length)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTrimLeftZeroes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{
			name:  "no leading zeroes",
			input: []byte{0x01, 0x02, 0x03},
			want:  []byte{0x01, 0x02, 0x03},
		},
		{
			name:  "leading zeroes trimmed",
			input: []byte{0x00, 0x00, 0x01, 0x02},
			want:  []byte{0x01, 0x02},
		},
		{
			name:  "all zeroes",
			input: []byte{0x00, 0x00, 0x00},
			want:  []byte{},
		},
		{
			name:  "empty slice",
			input: []byte{},
			want:  []byte{},
		},
		{
			name:  "single non-zero byte",
			input: []byte{0xff},
			want:  []byte{0xff},
		},
		{
			name:  "single zero byte",
			input: []byte{0x00},
			want:  []byte{},
		},
		{
			name:  "zero in the middle not trimmed",
			input: []byte{0x01, 0x00, 0x02},
			want:  []byte{0x01, 0x00, 0x02},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.TrimLeftZeroes(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBytes2Hex(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty bytes",
			input: []byte{},
			want:  "",
		},
		{
			name:  "single byte",
			input: []byte{0xab},
			want:  "ab",
		},
		{
			name:  "multiple bytes",
			input: []byte{0xde, 0xad, 0xbe, 0xef},
			want:  "deadbeef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.Bytes2Hex(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHex2Bytes(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:    "valid hex",
			input:   "deadbeef",
			want:    []byte{0xde, 0xad, 0xbe, 0xef},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "odd-length hex is error",
			input:   "abc",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid characters",
			input:   "zzzz",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := common.Hex2Bytes(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHex2BytesFixed(t *testing.T) {
	tests := []struct {
		name string
		str  string
		flen int
		want []byte
	}{
		{
			name: "exact length",
			str:  "deadbeef",
			flen: 4,
			want: []byte{0xde, 0xad, 0xbe, 0xef},
		},
		{
			name: "shorter than fixed length pads left",
			str:  "ab",
			flen: 4,
			want: []byte{0x00, 0x00, 0x00, 0xab},
		},
		{
			name: "longer than fixed length crops left",
			str:  "deadbeef",
			flen: 2,
			want: []byte{0xbe, 0xef},
		},
		{
			name: "empty string with fixed length",
			str:  "",
			flen: 2,
			want: []byte{0x00, 0x00},
		},
		{
			name: "zero fixed length with data",
			str:  "ab",
			flen: 0,
			want: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.Hex2BytesFixed(tt.str, tt.flen)
			assert.Equal(t, tt.want, got)
		})
	}
}
