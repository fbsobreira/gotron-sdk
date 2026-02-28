package address

import (
	"bytes"
	"database/sql/driver"
	"encoding/base64"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Well-known TRON addresses used across tests.
const (
	testBase58Addr1 = "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1"
	testBase58Addr2 = "TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9"
	testHexAddr1    = "41b9f4a69c5bae7cb8190e345d5de734779976a79c"
)

func TestBase64ToAddress(t *testing.T) {
	// Build a known address in raw bytes, then base64-encode it for round-trip.
	knownAddr, err := Base58ToAddress(testBase58Addr1)
	require.NoError(t, err)
	knownB64 := base64.StdEncoding.EncodeToString(knownAddr.Bytes())

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, addr Address)
	}{
		{
			name:    "valid base64 from known address",
			input:   knownB64,
			wantErr: false,
			check: func(t *testing.T, addr Address) {
				assert.Equal(t, knownAddr.Bytes(), addr.Bytes())
			},
		},
		{
			name:    "empty string produces empty address",
			input:   "",
			wantErr: false,
			check: func(t *testing.T, addr Address) {
				assert.Empty(t, addr)
			},
		},
		{
			name:    "valid base64 of arbitrary bytes",
			input:   base64.StdEncoding.EncodeToString([]byte{0x41, 0x01, 0x02}),
			wantErr: false,
			check: func(t *testing.T, addr Address) {
				assert.Equal(t, Address([]byte{0x41, 0x01, 0x02}), addr)
			},
		},
		{
			name:    "invalid base64 characters",
			input:   "!!!not-base64!!!",
			wantErr: true,
		},
		{
			name:    "truncated base64 padding",
			input:   "QUFB", // valid: decodes to "AAA"
			wantErr: false,
			check: func(t *testing.T, addr Address) {
				assert.Len(t, addr, 3)
			},
		},
		{
			name:    "invalid padding",
			input:   "QUFB===",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := Base64ToAddress(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, addr)
			}
		})
	}
}

func TestHexToAddress(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantNil   bool
		wantBytes []byte
	}{
		{
			name:  "valid hex without prefix",
			input: testHexAddr1,
			wantBytes: func() []byte {
				b, _ := common.FromHex(testHexAddr1)
				return b
			}(),
		},
		{
			name:  "valid hex with 0x prefix",
			input: "0x" + testHexAddr1,
			wantBytes: func() []byte {
				b, _ := common.FromHex(testHexAddr1)
				return b
			}(),
		},
		{
			name:  "valid hex with 0X prefix (uppercase)",
			input: "0X" + testHexAddr1,
			wantBytes: func() []byte {
				b, _ := common.FromHex(testHexAddr1)
				return b
			}(),
		},
		{
			name:    "invalid hex characters",
			input:   "ZZZZ",
			wantNil: true,
		},
		{
			name:  "short hex value",
			input: "0x41",
			wantBytes: func() []byte {
				b, _ := common.FromHex("41")
				return b
			}(),
		},
		{
			name:  "odd-length hex gets padded",
			input: "0x1",
			wantBytes: func() []byte {
				b, _ := common.FromHex("01")
				return b
			}(),
		},
		{
			name:  "empty string returns empty address",
			input: "",
			wantBytes: func() []byte {
				b, _ := common.FromHex("")
				return b
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := HexToAddress(tt.input)
			if tt.wantNil {
				assert.Nil(t, addr)
				return
			}
			require.NotNil(t, addr)
			assert.Equal(t, tt.wantBytes, addr.Bytes())
		})
	}
}

func TestBigToAddress(t *testing.T) {
	tests := []struct {
		name  string
		input *big.Int
		check func(t *testing.T, addr Address)
	}{
		{
			name:  "zero value produces all-zero 21 bytes",
			input: big.NewInt(0),
			check: func(t *testing.T, addr Address) {
				assert.Len(t, addr, AddressLength)
				assert.Equal(t, make([]byte, AddressLength), addr.Bytes())
			},
		},
		{
			name:  "small value is left-padded",
			input: big.NewInt(1),
			check: func(t *testing.T, addr Address) {
				assert.Len(t, addr, AddressLength)
				// last byte should be 1, rest zeros
				assert.Equal(t, byte(0x01), addr[AddressLength-1])
				for i := 0; i < AddressLength-1; i++ {
					assert.Equal(t, byte(0x00), addr[i], "byte %d should be zero", i)
				}
			},
		},
		{
			name: "value matching TRON prefix byte",
			input: func() *big.Int {
				b, _ := common.FromHex(testHexAddr1)
				return new(big.Int).SetBytes(b)
			}(),
			check: func(t *testing.T, addr Address) {
				expected, _ := common.FromHex(testHexAddr1)
				assert.Equal(t, expected, addr.Bytes())
				assert.Equal(t, TronBytePrefix, addr[0])
			},
		},
		{
			name:  "large value exactly 21 bytes",
			input: new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 168), big.NewInt(1)), // 2^168 - 1 = 21 bytes of 0xFF
			check: func(t *testing.T, addr Address) {
				assert.Len(t, addr, AddressLength)
				for i := range addr {
					assert.Equal(t, byte(0xFF), addr[i], "byte %d", i)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := BigToAddress(tt.input)
			tt.check(t, addr)
		})
	}
}

func TestPubkeyToAddress(t *testing.T) {
	t.Run("deterministic key produces valid TRON address", func(t *testing.T) {
		// Use a fixed private key for deterministic output.
		privKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
		require.NoError(t, err)

		addr := PubkeyToAddress(privKey.PublicKey)

		require.Len(t, addr, AddressLength, "address must be 21 bytes")
		assert.Equal(t, TronBytePrefix, addr[0], "first byte must be TRON prefix 0x41")
		assert.True(t, addr.IsValid())

		// Calling again with the same key should yield the same address.
		addr2 := PubkeyToAddress(privKey.PublicKey)
		assert.Equal(t, addr.Bytes(), addr2.Bytes(), "same key must produce same address")
	})

	t.Run("different keys produce different addresses", func(t *testing.T) {
		key1, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
		require.NoError(t, err)
		key2, err := crypto.HexToECDSA("0000000000000000000000000000000000000000000000000000000000000001")
		require.NoError(t, err)

		addr1 := PubkeyToAddress(key1.PublicKey)
		addr2 := PubkeyToAddress(key2.PublicKey)

		assert.NotEqual(t, addr1.Bytes(), addr2.Bytes())
	})

	t.Run("address matches ethereum derivation with TRON prefix", func(t *testing.T) {
		privKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
		require.NoError(t, err)

		ethAddr := crypto.PubkeyToAddress(privKey.PublicKey)
		tronAddr := PubkeyToAddress(privKey.PublicKey)

		// TRON address = 0x41 + ethereum address (20 bytes)
		assert.Equal(t, TronBytePrefix, tronAddr[0])
		assert.Equal(t, ethAddr.Bytes(), []byte(tronAddr[1:]))
	})
}

func TestHex(t *testing.T) {
	tests := []struct {
		name    string
		address Address
		check   func(t *testing.T, hex string)
	}{
		{
			name: "known address produces 0x-prefixed hex",
			address: func() Address {
				addr, _ := Base58ToAddress(testBase58Addr1)
				return addr
			}(),
			check: func(t *testing.T, hex string) {
				assert.True(t, strings.HasPrefix(hex, "0x"), "hex should start with 0x")
				// 21 bytes = 42 hex chars + "0x" prefix = 44 chars
				assert.Len(t, hex, 44)
			},
		},
		{
			name:    "single byte address",
			address: Address{0x41},
			check: func(t *testing.T, hex string) {
				assert.Equal(t, "0x41", hex)
			},
		},
		{
			name: "hex round-trips through HexToAddress",
			address: func() Address {
				addr, _ := Base58ToAddress(testBase58Addr1)
				return addr
			}(),
			check: func(t *testing.T, hex string) {
				roundTripped := HexToAddress(hex)
				original, _ := Base58ToAddress(testBase58Addr1)
				assert.Equal(t, original.Bytes(), roundTripped.Bytes())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.address.Hex()
			tt.check(t, result)
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name    string
		address Address
		want    string
	}{
		{
			name: "valid TRON address returns base58 string",
			address: func() Address {
				addr, _ := Base58ToAddress(testBase58Addr1)
				return addr
			}(),
			want: testBase58Addr1,
		},
		{
			name: "second valid address",
			address: func() Address {
				addr, _ := Base58ToAddress(testBase58Addr2)
				return addr
			}(),
			want: testBase58Addr2,
		},
		{
			name:    "empty address returns empty string",
			address: Address{},
			want:    "",
		},
		{
			name:    "nil address returns empty string",
			address: nil,
			want:    "",
		},
		{
			name:    "address starting with 0x00 returns big.Int string",
			address: Address(append([]byte{0x00}, make([]byte, 20)...)),
			want:    "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.address.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValue(t *testing.T) {
	t.Run("returns byte slice matching address", func(t *testing.T) {
		addr, err := Base58ToAddress(testBase58Addr1)
		require.NoError(t, err)

		val, err := addr.Value()
		require.NoError(t, err)

		valBytes, ok := val.([]byte)
		require.True(t, ok, "Value() should return []byte")
		assert.Equal(t, addr.Bytes(), valBytes)
	})

	t.Run("value implements driver.Valuer interface", func(t *testing.T) {
		addr, err := Base58ToAddress(testBase58Addr1)
		require.NoError(t, err)

		// Verify interface compliance at runtime.
		var _ driver.Valuer = addr
		val, err := addr.Value()
		require.NoError(t, err)
		assert.NotNil(t, val)
	})

	t.Run("empty address returns empty byte slice", func(t *testing.T) {
		addr := Address{}
		val, err := addr.Value()
		require.NoError(t, err)

		valBytes, ok := val.([]byte)
		require.True(t, ok)
		assert.Empty(t, valBytes)
	})
}

func TestRoundTrips(t *testing.T) {
	t.Run("address -> hex -> address", func(t *testing.T) {
		original, err := Base58ToAddress(testBase58Addr1)
		require.NoError(t, err)

		hexStr := original.Hex()
		recovered := HexToAddress(hexStr)

		assert.Equal(t, original.Bytes(), recovered.Bytes())
	})

	t.Run("address -> bytes -> address", func(t *testing.T) {
		original, err := Base58ToAddress(testBase58Addr1)
		require.NoError(t, err)

		raw := original.Bytes()
		recovered := Address(raw)

		assert.Equal(t, original.Bytes(), recovered.Bytes())
		assert.Equal(t, original.String(), recovered.String())
	})

	t.Run("address -> base58 string -> address", func(t *testing.T) {
		original, err := Base58ToAddress(testBase58Addr1)
		require.NoError(t, err)

		base58Str := original.String()
		recovered, err := Base58ToAddress(base58Str)
		require.NoError(t, err)

		assert.Equal(t, original.Bytes(), recovered.Bytes())
	})

	t.Run("address -> base64 -> address", func(t *testing.T) {
		original, err := Base58ToAddress(testBase58Addr1)
		require.NoError(t, err)

		b64 := base64.StdEncoding.EncodeToString(original.Bytes())
		recovered, err := Base64ToAddress(b64)
		require.NoError(t, err)

		assert.Equal(t, original.Bytes(), recovered.Bytes())
	})

	t.Run("address -> big.Int -> address", func(t *testing.T) {
		original, err := Base58ToAddress(testBase58Addr1)
		require.NoError(t, err)

		bi := new(big.Int).SetBytes(original.Bytes())
		recovered := BigToAddress(bi)

		assert.Equal(t, original.Bytes(), recovered.Bytes())
	})

	t.Run("pubkey -> address -> hex -> address preserves identity", func(t *testing.T) {
		privKey, err := crypto.HexToECDSA("fad9c8855b740a0b7ed4c221dbad0f33a83a49cad6b3fe8d5817ac83d38b6a19")
		require.NoError(t, err)

		addr := PubkeyToAddress(privKey.PublicKey)
		hexStr := addr.Hex()
		recovered := HexToAddress(hexStr)

		assert.Equal(t, addr.Bytes(), recovered.Bytes())
		assert.Equal(t, addr.String(), recovered.String())
	})

	t.Run("multiple addresses round-trip independently", func(t *testing.T) {
		addrs := []string{testBase58Addr1, testBase58Addr2}
		for _, b58 := range addrs {
			original, err := Base58ToAddress(b58)
			require.NoError(t, err)

			// hex round-trip
			fromHex := HexToAddress(original.Hex())
			assert.Equal(t, original.Bytes(), fromHex.Bytes(), "hex round-trip failed for %s", b58)

			// string round-trip
			fromStr, err := Base58ToAddress(original.String())
			require.NoError(t, err)
			assert.Equal(t, original.Bytes(), fromStr.Bytes(), "string round-trip failed for %s", b58)
		}
	})
}

func TestBase58ToAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, addr Address)
	}{
		{
			name:    "valid TRON address",
			input:   testBase58Addr1,
			wantErr: false,
			check: func(t *testing.T, addr Address) {
				assert.Len(t, addr, AddressLength)
				assert.Equal(t, TronBytePrefix, addr[0])
			},
		},
		{
			name:    "invalid checksum",
			input:   "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn2", // last char changed
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "random nonsense",
			input:   "not_a_valid_address",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr, err := Base58ToAddress(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.check != nil {
				tt.check(t, addr)
			}
		})
	}
}

func TestBTCECPubkeyToAddress(t *testing.T) {
	t.Run("nil key returns nil", func(t *testing.T) {
		addr := BTCECPubkeyToAddress(nil)
		assert.Nil(t, addr)
	})
}

func TestBTCECPrivkeyToAddress(t *testing.T) {
	t.Run("nil key returns nil", func(t *testing.T) {
		addr := BTCECPrivkeyToAddress(nil)
		assert.Nil(t, addr)
	})
}

func TestBytes(t *testing.T) {
	t.Run("returns underlying slice", func(t *testing.T) {
		raw := []byte{0x41, 0x01, 0x02, 0x03}
		addr := Address(raw)
		assert.Equal(t, raw, addr.Bytes())
	})
}

// Verify compile-time interface compliance.
var _ driver.Valuer = Address{}

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
