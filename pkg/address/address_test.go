package address

import (
	"bytes"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

func TestAddress_Scan(t *testing.T) {
	validAddress, err := Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// correct case
	want := validAddress
	a := &Address{}
	src := validAddress.Bytes()
	err = a.Scan(src)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !bytes.Equal(a.Bytes(), want.Bytes()) {
		t.Errorf("got %v, want %v", *a, want)
	}

	// invalid type of src
	a = &Address{}
	err = a.Scan("not a byte slice")
	if err == nil {
		t.Errorf("expected an error, but got none")
	}

	// invalid length of src
	a = &Address{}
	src = make([]byte, 4)
	err = a.Scan(src)
	if err == nil {
		t.Errorf("expected an error, but got none")
	}
	src = make([]byte, 22) // Creating a byte array with the wrong length
	err = a.Scan(src)
	if err == nil {
		t.Errorf("expected an error, but got none")
	}
}

func TestAddress_IsValid(t *testing.T) {
	tests := []struct {
		name    string
		address Address
		want    bool
	}{
		{
			name: "valid address",
			address: func() Address {
				addr, _ := Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
				return addr
			}(),
			want: true,
		},
		{
			name:    "nil address",
			address: nil,
			want:    false,
		},
		{
			name:    "empty address",
			address: Address{},
			want:    false,
		},
		{
			name:    "wrong length",
			address: Address{0x41, 0x00, 0x00}, // too short
			want:    false,
		},
		{
			name: "wrong prefix",
			address: func() Address {
				addr := make([]byte, AddressLength)
				addr[0] = 0x42 // wrong prefix
				return Address(addr)
			}(),
			want: false,
		},
		{
			name: "valid mainnet address",
			address: func() Address {
				addr, _ := Base58ToAddress("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
				return addr
			}(),
			want: true,
		},
		{
			name: "base58 decode without validation - valid",
			address: func() Address {
				// Manually decode a valid base58 address without using Base58ToAddress
				decoded, _ := common.Decode("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
				// Remove checksum (last 4 bytes)
				if len(decoded) > 4 {
					return Address(decoded[:len(decoded)-4])
				}
				return Address(decoded)
			}(),
			want: true,
		},
		{
			name: "base58 decode without validation - wrong prefix",
			address: func() Address {
				// Create a base58 string with wrong prefix
				data := make([]byte, 21)
				data[0] = 0x42 // wrong prefix
				for i := 1; i < 21; i++ {
					data[i] = byte(i)
				}
				encoded := common.EncodeCheck(data)
				decoded, _ := common.Decode(encoded)
				if len(decoded) > 4 {
					return Address(decoded[:len(decoded)-4])
				}
				return Address(decoded)
			}(),
			want: false,
		},
		{
			name: "hex decode - valid TRON address",
			address: func() Address {
				// Valid TRON address in hex (with 0x41 prefix)
				hexBytes, _ := common.FromHex("41b9f4a69c5bae7cb8190e345d5de734779976a79c")
				return Address(hexBytes)
			}(),
			want: true,
		},
		{
			name: "hex decode - ethereum style address",
			address: func() Address {
				// Ethereum address (starts with different bytes)
				hexBytes, _ := common.FromHex("742d35Cc6634C0532925a3b844Bc9e7595f8b4e0")
				// Prepend any byte to make it 21 bytes
				return Address(append([]byte{0x00}, hexBytes...))
			}(),
			want: false,
		},
		{
			name: "hex decode - wrong prefix",
			address: func() Address {
				// Hex with wrong prefix (0x42 instead of 0x41)
				hexBytes, _ := common.FromHex("42b9f4a69c5bae7cb8190e345d5de734779976a79c")
				return Address(hexBytes)
			}(),
			want: false,
		},
		{
			name: "manually constructed - correct length wrong prefix",
			address: func() Address {
				// Create exactly 21 bytes with wrong prefix
				data := make([]byte, AddressLength)
				data[0] = 0x40 // Wrong prefix (should be 0x41)
				return Address(data)
			}(),
			want: false,
		},
		{
			name: "manually constructed - all zeros except prefix",
			address: func() Address {
				// Valid prefix but all other bytes are zero
				data := make([]byte, AddressLength)
				data[0] = TronBytePrefix
				return Address(data)
			}(),
			want: true, // This is technically valid format-wise
		},
		{
			name: "decode address with valid checksum but wrong length - TronEnergyioE1Z3ukeRv38sYkv5Jn55bL",
			address: func() Address {
				// Try to decode, this should fail due to length check in DecodeCheck
				decoded, err := common.Decode("TronEnergyioE1Z3ukeRv38sYkv5Jn55bL")
				if err != nil {
					panic("DecodeCheck should not fail for this address")
				}
				return Address(decoded)
			}(),
			want: false, // Should be invalid due to wrong length
		},
		{
			name: "decode TronEnergyioNijNo8g3LF2ABKUAae6D2Z - invalid format",
			address: func() Address {
				decoded, err := common.Decode("TronEnergyioNijNo8g3LF2ABKUAae6D2Z")
				if err != nil {
					panic("DecodeCheck should not fail for this address")
				}
				return Address(decoded)
			}(),
			want: false,
		},
		{
			name: "decode TronEnergyio3ZMcXA5hSjrTxaioKGgqyr - invalid format",
			address: func() Address {
				decoded, err := common.Decode("TronEnergyio3ZMcXA5hSjrTxaioKGgqyr")
				if err != nil {
					panic("DecodeCheck should not fail for this address")
				}
				return Address(decoded)
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.address.IsValid(); got != tt.want {
				t.Errorf("Address.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
