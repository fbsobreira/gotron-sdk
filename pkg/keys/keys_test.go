package keys_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	t.Run("produces a valid private key", func(t *testing.T) {
		pk, err := keys.GenerateKey()
		require.NoError(t, err)
		require.NotNil(t, pk)

		// Private key bytes must be exactly 32 bytes.
		assert.Len(t, pk.Serialize(), 32)
	})

	t.Run("produces unique keys on successive calls", func(t *testing.T) {
		pk1, err := keys.GenerateKey()
		require.NoError(t, err)

		pk2, err := keys.GenerateKey()
		require.NoError(t, err)

		assert.NotEqual(t, pk1.Serialize(), pk2.Serialize(),
			"two generated keys must differ")
	})
}

func TestGetPrivateKeyFromHex(t *testing.T) {
	// A well-known 32-byte hex key (all 1s).
	const validHex = "0000000000000000000000000000000000000000000000000000000000000001"

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:  "valid 32-byte hex",
			input: validHex,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: "invalid private key length",
		},
		{
			name:    "odd-length hex",
			input:   "abc",
			wantErr: "failed to decode private key hex",
		},
		{
			name:    "non-hex characters",
			input:   "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantErr: "failed to decode private key hex",
		},
		{
			name:    "too short (16 bytes)",
			input:   "00000000000000000000000000000001",
			wantErr: "invalid private key length",
		},
		{
			name:    "too long (33 bytes)",
			input:   "000000000000000000000000000000000000000000000000000000000000000001",
			wantErr: "invalid private key length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pk, err := keys.GetPrivateKeyFromHex(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, pk)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pk)
				assert.Len(t, pk.Serialize(), 32)
			}
		})
	}
}

func TestGetPrivateKeyFromBytes(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		wantErr string
	}{
		{
			name:  "valid 32 bytes",
			input: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
		},
		{
			name:    "empty slice",
			input:   []byte{},
			wantErr: "invalid private key length: 0",
		},
		{
			name:    "nil slice",
			input:   nil,
			wantErr: "invalid private key length: 0",
		},
		{
			name:    "too short (31 bytes)",
			input:   make([]byte, 31),
			wantErr: "invalid private key length: 31",
		},
		{
			name:    "too long (33 bytes)",
			input:   make([]byte, 33),
			wantErr: "invalid private key length: 33",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pk, err := keys.GetPrivateKeyFromBytes(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, pk)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pk)
			}
		})
	}

	t.Run("valid 32 bytes with non-zero value", func(t *testing.T) {
		b := make([]byte, 32)
		b[31] = 1
		pk, err := keys.GetPrivateKeyFromBytes(b)
		require.NoError(t, err)
		assert.Equal(t, b, pk.Serialize())
	})
}

func TestEncodeHex(t *testing.T) {
	t.Run("round-trip with GetPrivateKeyFromHex", func(t *testing.T) {
		// Use a known private key hex.
		const inputHex = "b5a4cea271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28" // gitleaks:allow
		pk, err := keys.GetPrivateKeyFromHex(inputHex)
		require.NoError(t, err)

		dump := keys.EncodeHex(pk, pk.PubKey())
		require.NotNil(t, dump)

		// The private key hex should be the input hex prefixed with 0x.
		assert.Equal(t, "0x"+inputHex, dump.PrivateKey)

		// Public key fields must not be empty.
		assert.NotEmpty(t, dump.PublicKeyCompressed, "compressed public key must not be empty")
		assert.NotEmpty(t, dump.PublicKey, "uncompressed public key must not be empty")

		// Compressed key is 33 bytes (0x prefix + 66 hex chars).
		assert.Len(t, dump.PublicKeyCompressed, 2+66)
		// Uncompressed key is 65 bytes (0x prefix + 130 hex chars).
		assert.Len(t, dump.PublicKey, 2+130)
	})

	t.Run("private key round-trip preserves value", func(t *testing.T) {
		pk, err := keys.GenerateKey()
		require.NoError(t, err)

		dump := keys.EncodeHex(pk, pk.PubKey())

		// Strip 0x prefix and decode back.
		hexStr := strings.TrimPrefix(dump.PrivateKey, "0x")
		recovered, err := keys.GetPrivateKeyFromHex(hexStr)
		require.NoError(t, err)

		assert.Equal(t, pk.Serialize(), recovered.Serialize(),
			"round-tripped key must equal original")
	})
}

func TestGetPrivateKeyFromHex_knownVector(t *testing.T) {
	// Test vector: private key hex -> expected serialized bytes.
	const inputHex = "b5a4cea271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28" // gitleaks:allow
	expected, err := hex.DecodeString(inputHex)
	require.NoError(t, err)

	pk, err := keys.GetPrivateKeyFromHex(inputHex)
	require.NoError(t, err)
	assert.Equal(t, expected, pk.Serialize())
}

func TestGetPrivateKeyFromHex_prefixAndCase(t *testing.T) {
	// Known private key in lowercase.
	const lowerHex = "b5a4cea271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28" // gitleaks:allow

	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "0x prefix is rejected (contains non-hex 'x')",
			input:   "0x" + lowerHex,
			wantErr: "failed to decode private key hex",
		},
		{
			name:    "0X prefix is rejected",
			input:   "0X" + lowerHex,
			wantErr: "failed to decode private key hex",
		},
		{
			name:  "uppercase hex is accepted",
			input: strings.ToUpper(lowerHex),
		},
		{
			name:  "mixed case hex is accepted",
			input: "B5A4CEA271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pk, err := keys.GetPrivateKeyFromHex(tt.input)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, pk)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pk)

				// Regardless of input case, the serialized bytes must match.
				expectedBytes, decErr := hex.DecodeString(lowerHex)
				require.NoError(t, decErr)
				assert.Equal(t, expectedBytes, pk.Serialize(),
					"key bytes must match regardless of hex case")
			}
		})
	}
}

func TestGetPrivateKeyFromBytes_knownPublicKey(t *testing.T) {
	// Use a well-known private key and verify the derived public key.
	const privHex = "b5a4cea271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28" // gitleaks:allow
	privBytes, err := hex.DecodeString(privHex)
	require.NoError(t, err)

	pk, err := keys.GetPrivateKeyFromBytes(privBytes)
	require.NoError(t, err)
	require.NotNil(t, pk)

	// Verify the private key round-trips.
	assert.Equal(t, privBytes, pk.Serialize())

	// Verify the public key is deterministic and has the right sizes.
	pub := pk.PubKey()
	require.NotNil(t, pub)

	compressed := pub.SerializeCompressed()
	uncompressed := pub.SerializeUncompressed()

	assert.Len(t, compressed, 33, "compressed public key must be 33 bytes")
	assert.Len(t, uncompressed, 65, "uncompressed public key must be 65 bytes")

	// Compressed key must start with 0x02 or 0x03.
	assert.Contains(t, []byte{0x02, 0x03}, compressed[0],
		"compressed public key must start with 0x02 or 0x03")

	// Uncompressed key must start with 0x04.
	assert.Equal(t, byte(0x04), uncompressed[0],
		"uncompressed public key must start with 0x04")

	// Loading the same bytes again must produce the same public key.
	pk2, err := keys.GetPrivateKeyFromBytes(privBytes)
	require.NoError(t, err)
	assert.Equal(t, pub.SerializeCompressed(), pk2.PubKey().SerializeCompressed(),
		"same private key bytes must produce the same public key")
}

func TestGenerateKey_roundTrip(t *testing.T) {
	// GenerateKey -> EncodeHex -> GetPrivateKeyFromHex -> verify keys match.
	t.Run("full round-trip preserves private and public keys", func(t *testing.T) {
		original, err := keys.GenerateKey()
		require.NoError(t, err)

		dump := keys.EncodeHex(original, original.PubKey())
		require.NotNil(t, dump)

		// Strip the 0x prefix for GetPrivateKeyFromHex.
		hexStr := strings.TrimPrefix(dump.PrivateKey, "0x")
		recovered, err := keys.GetPrivateKeyFromHex(hexStr)
		require.NoError(t, err)

		// Private key bytes must be identical.
		assert.Equal(t, original.Serialize(), recovered.Serialize(),
			"round-tripped private key must match original")

		// Public key bytes must be identical.
		assert.Equal(t,
			original.PubKey().SerializeCompressed(),
			recovered.PubKey().SerializeCompressed(),
			"round-tripped public key (compressed) must match original")
		assert.Equal(t,
			original.PubKey().SerializeUncompressed(),
			recovered.PubKey().SerializeUncompressed(),
			"round-tripped public key (uncompressed) must match original")
	})

	t.Run("multiple round-trips are stable", func(t *testing.T) {
		pk, err := keys.GenerateKey()
		require.NoError(t, err)

		// Round-trip three times.
		current := pk
		for i := range 3 {
			dump := keys.EncodeHex(current, current.PubKey())
			hexStr := strings.TrimPrefix(dump.PrivateKey, "0x")
			current, err = keys.GetPrivateKeyFromHex(hexStr)
			require.NoError(t, err, "round-trip %d failed", i)
		}

		assert.Equal(t, pk.Serialize(), current.Serialize(),
			"key must be stable after multiple round-trips")
	})
}

func TestEncodeHex_differentKeys(t *testing.T) {
	// Verify that different private keys produce different encoded outputs.
	pk1, err := keys.GenerateKey()
	require.NoError(t, err)

	pk2, err := keys.GenerateKey()
	require.NoError(t, err)

	dump1 := keys.EncodeHex(pk1, pk1.PubKey())
	dump2 := keys.EncodeHex(pk2, pk2.PubKey())

	assert.NotEqual(t, dump1.PrivateKey, dump2.PrivateKey,
		"different keys must produce different private key hex")
	assert.NotEqual(t, dump1.PublicKeyCompressed, dump2.PublicKeyCompressed,
		"different keys must produce different compressed public key hex")
	assert.NotEqual(t, dump1.PublicKey, dump2.PublicKey,
		"different keys must produce different uncompressed public key hex")
}

func TestEncodeHex_format(t *testing.T) {
	tests := []struct {
		name   string
		keyHex string
	}{
		{
			name:   "key with value 1",
			keyHex: "0000000000000000000000000000000000000000000000000000000000000001",
		},
		{
			name:   "known mnemonic-derived key",
			keyHex: "b5a4cea271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28", // gitleaks:allow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pk, err := keys.GetPrivateKeyFromHex(tt.keyHex)
			require.NoError(t, err)

			dump := keys.EncodeHex(pk, pk.PubKey())
			require.NotNil(t, dump)

			// All fields must have 0x prefix.
			assert.True(t, strings.HasPrefix(dump.PrivateKey, "0x"),
				"private key hex must start with 0x")
			assert.True(t, strings.HasPrefix(dump.PublicKeyCompressed, "0x"),
				"compressed public key hex must start with 0x")
			assert.True(t, strings.HasPrefix(dump.PublicKey, "0x"),
				"uncompressed public key hex must start with 0x")

			// Private key: 0x + 64 hex chars = 66 chars total.
			assert.Len(t, dump.PrivateKey, 2+64,
				"private key hex must be 66 chars (0x + 64)")
		})
	}
}

func TestFromMnemonicSeedAndPassphrase(t *testing.T) {
	// Standard BIP39 test mnemonic.
	const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	t.Run("index 0 produces known private key", func(t *testing.T) {
		private, pub := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "", 0)
		require.NotNil(t, private)
		require.NotNil(t, pub)

		expected := "b5a4cea271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28" // gitleaks:allow
		assert.Equal(t, expected, hex.EncodeToString(private.Serialize()))
	})

	t.Run("different indices produce different keys", func(t *testing.T) {
		pk0, _ := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "", 0)
		pk1, _ := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "", 1)
		require.NotNil(t, pk0)
		require.NotNil(t, pk1)

		assert.NotEqual(t, pk0.Serialize(), pk1.Serialize(),
			"different derivation indices must produce different keys")
	})

	t.Run("same mnemonic and index is deterministic", func(t *testing.T) {
		pk1, pub1 := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "", 0)
		pk2, pub2 := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "", 0)
		require.NotNil(t, pk1)
		require.NotNil(t, pk2)

		assert.Equal(t, pk1.Serialize(), pk2.Serialize(),
			"same mnemonic and index must produce the same private key")
		assert.Equal(t, pub1.SerializeCompressed(), pub2.SerializeCompressed(),
			"same mnemonic and index must produce the same public key")
	})

	t.Run("passphrase changes the derived key", func(t *testing.T) {
		pkNoPass, _ := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "", 0)
		pkWithPass, _ := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "my-secret", 0)
		require.NotNil(t, pkNoPass)
		require.NotNil(t, pkWithPass)

		assert.NotEqual(t, pkNoPass.Serialize(), pkWithPass.Serialize(),
			"a passphrase must change the derived key")
	})

	t.Run("returned public key matches private key", func(t *testing.T) {
		private, pub := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "", 0)
		require.NotNil(t, private)
		require.NotNil(t, pub)

		// The public key from the returned pair must match what we derive from the private key.
		assert.Equal(t, private.PubKey().SerializeCompressed(), pub.SerializeCompressed(),
			"returned public key must correspond to the returned private key")
	})

	t.Run("negative derivation index is rejected", func(t *testing.T) {
		private, pub := keys.FromMnemonicSeedAndPassphrase(testMnemonic, "", -1)
		assert.Nil(t, private)
		assert.Nil(t, pub)
	})
}
