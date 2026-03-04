package common_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DecodeBase58(t *testing.T) {

	invalidAddresses := []string{
		"TronEnergyioE1Z3ukeRv38sYkv5Jn55bL",
		"TronEnergyioNijNo8g3LF2ABKUAae6D2Z",
		"TronEnergyio3ZMcXA5hSjrTxaioKGgqyr",
	}

	validAddresses := []string{
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		"TVj7RNVHy6thbM7BWdSe9G6gXwKhjhdNZS",
		"THPvaUhoh2Qn2y9THCZML3H815hhFhn5YC",
	}

	for _, addr := range invalidAddresses {
		_, err := common.DecodeCheck(addr)
		assert.NotNil(t, err)
	}

	for _, addr := range validAddresses {
		_, err := common.DecodeCheck(addr)
		assert.Nil(t, err)
	}

}

func TestEncode(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty input",
			input: []byte{},
			want:  "",
		},
		{
			name:  "single zero byte encodes to leading 1",
			input: []byte{0x00},
			want:  "1",
		},
		{
			name:  "multiple leading zeros",
			input: []byte{0x00, 0x00, 0x01},
			want:  "112",
		},
		{
			name:  "known value",
			input: []byte{0x01},
			want:  "2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.Encode(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []byte
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    []byte{},
			wantErr: false,
		},
		{
			name:    "single 1 decodes to zero byte",
			input:   "1",
			want:    []byte{0x00},
			wantErr: false,
		},
		{
			name:    "valid base58 string",
			input:   "2",
			want:    []byte{0x01},
			wantErr: false,
		},
		{
			name:    "invalid character (0 is not in base58)",
			input:   "0",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid character (O is not in base58)",
			input:   "O",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid character (I is not in base58)",
			input:   "I",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid character (l is not in base58)",
			input:   "l",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := common.Decode(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "single byte",
			input: []byte{0x42},
		},
		{
			name:  "multiple bytes",
			input: []byte{0xde, 0xad, 0xbe, 0xef},
		},
		{
			name:  "leading zero preserved",
			input: []byte{0x00, 0x01, 0x02},
		},
		{
			name:  "all zeros",
			input: []byte{0x00, 0x00, 0x00},
		},
		{
			name:  "all 0xff",
			input: []byte{0xff, 0xff, 0xff},
		},
		{
			name:  "21 bytes like TRON address",
			input: []byte{0x41, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := common.Encode(tt.input)
			decoded, err := common.Decode(encoded)
			require.NoError(t, err)
			assert.Equal(t, tt.input, decoded, "round-trip Encode -> Decode should produce original bytes")
		})
	}
}

func TestEncodeCheckDecodeCheckRoundTrip(t *testing.T) {
	// A valid 21-byte TRON address payload (prefix 0x41 + 20 bytes)
	validPayloads := [][]byte{
		{0x41, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
			0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14},
		{0x41, 0xff, 0xfe, 0xfd, 0xfc, 0xfb, 0xfa, 0xf9, 0xf8, 0xf7, 0xf6,
			0xf5, 0xf4, 0xf3, 0xf2, 0xf1, 0xf0, 0xef, 0xee, 0xed, 0xec},
		{0x41, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
	}

	for i, payload := range validPayloads {
		encoded := common.EncodeCheck(payload)
		decoded, err := common.DecodeCheck(encoded)
		require.NoError(t, err, "payload %d should decode without error", i)
		assert.Equal(t, payload, decoded, "round-trip EncodeCheck -> DecodeCheck should match for payload %d", i)
	}
}

func TestDecodeCheckInvalidCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "too short after decoding",
			input:   "1",
			wantErr: "b58 check error",
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: "b58 check error",
		},
		{
			name:    "wrong checksum",
			input:   "TronEnergyioE1Z3ukeRv38sYkv5Jn55bL",
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := common.DecodeCheck(tt.input)
			require.Error(t, err)
			if tt.wantErr != "" {
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDecodeCheckWrongPrefix(t *testing.T) {
	// Build a 21-byte payload with wrong prefix (0x42 instead of 0x41)
	payload := make([]byte, 21)
	payload[0] = 0x42 // wrong prefix
	for i := 1; i < 21; i++ {
		payload[i] = byte(i)
	}
	// Use EncodeCheck to get proper checksum but wrong prefix
	encoded := common.EncodeCheck(payload)

	_, err := common.DecodeCheck(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid prefix")
}

func TestDecodeCheckWrongLength(t *testing.T) {
	// Build a 15-byte payload with correct prefix but wrong length
	payload := make([]byte, 15)
	payload[0] = 0x41
	for i := 1; i < 15; i++ {
		payload[i] = byte(i)
	}
	encoded := common.EncodeCheck(payload)

	_, err := common.DecodeCheck(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid address length")
}

func TestDecodeCheckKnownAddresses(t *testing.T) {
	// Well-known TRON addresses that should decode successfully
	tests := []struct {
		name    string
		address string
	}{
		{
			name:    "USDT TRC20 contract",
			address: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		},
		{
			name:    "valid address 1",
			address: "TVj7RNVHy6thbM7BWdSe9G6gXwKhjhdNZS",
		},
		{
			name:    "valid address 2",
			address: "THPvaUhoh2Qn2y9THCZML3H815hhFhn5YC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoded, err := common.DecodeCheck(tt.address)
			require.NoError(t, err)
			assert.Len(t, decoded, 21, "decoded TRON address should be 21 bytes")
			assert.Equal(t, byte(0x41), decoded[0], "first byte should be mainnet prefix 0x41")
		})
	}
}

func TestEncodeCheckDeterministic(t *testing.T) {
	payload := []byte{0x41, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}

	encoded1 := common.EncodeCheck(payload)
	encoded2 := common.EncodeCheck(payload)
	assert.Equal(t, encoded1, encoded2, "EncodeCheck must be deterministic")
}

func TestEncodeCheckDoesNotMutateInput(t *testing.T) {
	payload := []byte{0x41, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
		0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14}
	original := append([]byte(nil), payload...)

	_ = common.EncodeCheck(payload)
	assert.Equal(t, original, payload, "EncodeCheck must not mutate caller input")
}
