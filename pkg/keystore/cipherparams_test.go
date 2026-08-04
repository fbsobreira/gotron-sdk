package keystore_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/require"
)

// keystoreWithCrypto overrides individual crypto fields on an otherwise
// well-formed v3 document so only the cipher-side fields are under test.
func keystoreWithCrypto(t *testing.T, crypto map[string]interface{}) []byte {
	t.Helper()
	doc := map[string]interface{}{
		"address": "41a614f803b6fd780986a42c78ec9c7f77e6ded13c",
		"crypto":  crypto,
		"id":      "3198bc9c-6672-5ab3-d995-4942343ae5b6",
		"version": 3,
	}
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	return b
}

func validCrypto() map[string]interface{} {
	return map[string]interface{}{
		"cipher":       "aes-128-ctr",
		"ciphertext":   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"cipherparams": map[string]interface{}{"iv": "0123456789abcdef0123456789abcdef"},
		"kdf":          "scrypt",
		"kdfparams": map[string]interface{}{
			"dklen": 32, "n": 4096, "p": 1, "r": 8,
			"salt": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		"mac": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
}

// Cipher-side fields are unauthenticated until the MAC is checked. Wrong IV
// length previously panicked in cipher.NewCTR after a matching passphrase; huge
// ciphertext/mac hex allocated before KDF. These must return errors, not panic.
func TestDecryptKey_MalformedCipherFields(t *testing.T) {
	tests := map[string]func() map[string]interface{}{
		"iv too short": func() map[string]interface{} {
			c := validCrypto()
			c["cipherparams"] = map[string]interface{}{"iv": "0123456789abcdef"} // 8 bytes
			return c
		},
		"iv too long": func() map[string]interface{} {
			c := validCrypto()
			c["cipherparams"] = map[string]interface{}{"iv": strings.Repeat("ab", 17)} // 17 bytes
			return c
		},
		"iv hex beyond limit": func() map[string]interface{} {
			c := validCrypto()
			c["cipherparams"] = map[string]interface{}{"iv": strings.Repeat("ab", 33)} // > 16*2 hex
			return c
		},
		"mac too short": func() map[string]interface{} {
			c := validCrypto()
			c["mac"] = "0123456789abcdef"
			return c
		},
		"mac hex beyond limit": func() map[string]interface{} {
			c := validCrypto()
			c["mac"] = strings.Repeat("ab", 33) // > 32*2
			return c
		},
		"ciphertext hex beyond limit": func() map[string]interface{} {
			c := validCrypto()
			c["ciphertext"] = strings.Repeat("ab", 1025) // > 1024 bytes
			return c
		},
		"empty ciphertext": func() map[string]interface{} {
			c := validCrypto()
			c["ciphertext"] = ""
			return c
		},
	}

	for name, build := range tests {
		t.Run(name, func(t *testing.T) {
			doc := keystoreWithCrypto(t, build())
			require.NotPanics(t, func() {
				_, err := keystore.DecryptKey(doc, "passphrase")
				require.Error(t, err)
			})
		})
	}
}

// V1 CBC decrypt previously panicked in CryptBlocks when ciphertext was not a
// multiple of the AES block size, even after a successful MAC check.
func TestDecryptKey_V1CiphertextNotBlockAligned(t *testing.T) {
	doc := map[string]interface{}{
		"address": "41a614f803b6fd780986a42c78ec9c7f77e6ded13c",
		"crypto": map[string]interface{}{
			"cipher":       "aes-128-cbc",
			"ciphertext":   "0123456789abcdef01", // 9 bytes — not a multiple of 16
			"cipherparams": map[string]interface{}{"iv": "0123456789abcdef0123456789abcdef"},
			"kdf":          "scrypt",
			"kdfparams": map[string]interface{}{
				"dklen": 32, "n": 4096, "p": 1, "r": 8,
				"salt": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
			"mac": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		"id":      "3198bc9c-6672-5ab3-d995-4942343ae5b6",
		"version": "1",
	}
	b, err := json.Marshal(doc)
	require.NoError(t, err)

	require.NotPanics(t, func() {
		_, err := keystore.DecryptKey(b, "passphrase")
		require.Error(t, err)
	})
}
