package keystore

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- newKey ----------

func TestNewKey(t *testing.T) {
	t.Run("generates a valid key", func(t *testing.T) {
		key, err := newKey(rand.Reader)
		require.NoError(t, err)
		require.NotNil(t, key)

		assert.NotNil(t, key.PrivateKey, "private key must not be nil")
		assert.NotEmpty(t, key.Address, "address must not be empty")
		assert.NotEqual(t, uuid.UUID(nil), key.ID, "UUID must be set")
	})

	t.Run("generates unique keys", func(t *testing.T) {
		key1, err := newKey(rand.Reader)
		require.NoError(t, err)

		key2, err := newKey(rand.Reader)
		require.NoError(t, err)

		assert.NotEqual(t, key1.ID.String(), key2.ID.String(),
			"two generated keys must have different UUIDs")
		assert.NotEqual(t, key1.PrivateKey.D.Bytes(), key2.PrivateKey.D.Bytes(),
			"two generated keys must have different private keys")
	})
}

// ---------- newKeyFromECDSA ----------

func TestNewKeyFromECDSA(t *testing.T) {
	t.Run("creates key from ECDSA private key", func(t *testing.T) {
		privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		require.NoError(t, err)

		key := newKeyFromECDSA(privKey)
		require.NotNil(t, key)

		assert.Equal(t, privKey, key.PrivateKey)
		assert.NotEmpty(t, key.Address)
		assert.NotEqual(t, uuid.UUID(nil), key.ID)

		// Address must match what address.PubkeyToAddress produces.
		expectedAddr := address.PubkeyToAddress(privKey.PublicKey)
		assert.Equal(t, expectedAddr, key.Address)
	})
}

// ---------- Key MarshalJSON / UnmarshalJSON ----------

func TestKeyJSON(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "round-trip marshal then unmarshal"},
		{name: "different key round-trip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
			require.NoError(t, err)

			original := newKeyFromECDSA(privKey)

			data, err := json.Marshal(original)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			var restored Key
			err = json.Unmarshal(data, &restored)
			require.NoError(t, err)

			// Verify fields match.
			assert.Equal(t, original.ID.String(), restored.ID.String(),
				"UUID must round-trip")
			assert.Equal(t, original.Address, address.Address(restored.Address),
				"address must round-trip")
			assert.Equal(t, original.PrivateKey.D.Bytes(), restored.PrivateKey.D.Bytes(),
				"private key must round-trip")
		})
	}

	t.Run("unmarshal invalid JSON fails", func(t *testing.T) {
		var k Key
		err := json.Unmarshal([]byte("{invalid-json"), &k)
		assert.Error(t, err)
	})

	t.Run("unmarshal missing fields fails", func(t *testing.T) {
		var k Key
		err := json.Unmarshal([]byte(`{"address":"","privatekey":"invalid","id":"","version":3}`), &k)
		assert.Error(t, err)
	})

	t.Run("marshal produces expected fields", func(t *testing.T) {
		privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		require.NoError(t, err)
		key := newKeyFromECDSA(privKey)

		data, err := json.Marshal(key)
		require.NoError(t, err)

		var parsed map[string]interface{}
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)

		assert.Contains(t, parsed, "address")
		assert.Contains(t, parsed, "privatekey")
		assert.Contains(t, parsed, "id")
		assert.Contains(t, parsed, "version")
		assert.Equal(t, float64(version), parsed["version"])
	})
}

// Note: NewKeyForDirectICAP is not tested here because it uses recursive retries
// until the generated address fits into < 155 bits, which can overflow the stack
// with the standard rand.Reader (each call reads a fixed 64-byte buffer and derives
// a deterministic reader, so the chance of hitting a valid address per attempt is
// very low). The function is exercised indirectly via integration if needed.

// ---------- storeNewKey ----------

func TestStoreNewKey(t *testing.T) {
	t.Run("stores key to disk and returns account", func(t *testing.T) {
		dir := t.TempDir()
		store := &keyStorePassphrase{
			keysDirPath:             dir,
			scryptN:                 LightScryptN,
			scryptP:                 LightScryptP,
			skipKeyFileVerification: true,
		}

		key, acc, err := storeNewKey(store, rand.Reader, "test-pass")
		require.NoError(t, err)
		require.NotNil(t, key)

		assert.NotEmpty(t, acc.Address)
		assert.NotEmpty(t, acc.URL.Path)
		assert.Equal(t, KeyStoreScheme, acc.URL.Scheme)

		// Verify key can be retrieved from disk.
		loaded, err := store.GetKey(acc.Address, acc.URL.Path, "test-pass")
		require.NoError(t, err)
		assert.Equal(t, key.PrivateKey.D.Bytes(), loaded.PrivateKey.D.Bytes())
	})
}

// ---------- keyFileName ----------

func TestKeyFileName(t *testing.T) {
	t.Run("format starts with UTC--", func(t *testing.T) {
		addr := address.PubkeyToAddress(mustGenerateKey(t).PublicKey)
		name := keyFileName(addr)
		assert.True(t, strings.HasPrefix(name, "UTC--"), "key filename must start with UTC--")
	})

	t.Run("different addresses produce different filenames", func(t *testing.T) {
		addr1 := address.PubkeyToAddress(mustGenerateKey(t).PublicKey)
		addr2 := address.PubkeyToAddress(mustGenerateKey(t).PublicKey)

		name1 := keyFileName(addr1)
		name2 := keyFileName(addr2)
		assert.NotEqual(t, name1, name2)
	})
}

// ---------- toISO8601 ----------

func TestToISO8601(t *testing.T) {
	t.Run("UTC timezone ends with Z", func(t *testing.T) {
		utcTime := time.Date(2024, 1, 15, 10, 30, 45, 123456789, time.UTC)
		result := toISO8601(utcTime)
		assert.Contains(t, result, "Z")
		assert.Contains(t, result, "2024-01-15")
	})

	t.Run("contains expected date components", func(t *testing.T) {
		utcTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
		result := toISO8601(utcTime)
		assert.Contains(t, result, "2025-12-31")
		assert.Contains(t, result, "23-59-59")
	})
}

// ---------- EncryptDataV3 ----------

func TestEncryptDataV3(t *testing.T) {
	t.Run("encrypts and produces valid CryptoJSON", func(t *testing.T) {
		data := []byte("hello world secret data 12345!!")
		auth := []byte("password")

		cj, err := EncryptDataV3(data, auth, LightScryptN, LightScryptP)
		require.NoError(t, err)

		assert.Equal(t, "aes-128-ctr", cj.Cipher)
		assert.Equal(t, keyHeaderKDF, cj.KDF)
		assert.NotEmpty(t, cj.CipherText)
		assert.NotEmpty(t, cj.CipherParams.IV)
		assert.NotEmpty(t, cj.MAC)
		assert.NotNil(t, cj.KDFParams)
	})

	t.Run("decrypt round-trip with DecryptDataV3", func(t *testing.T) {
		data := []byte("secret-payload-for-round-trip!!")
		auth := []byte("my-password")

		cj, err := EncryptDataV3(data, auth, LightScryptN, LightScryptP)
		require.NoError(t, err)

		decrypted, err := DecryptDataV3(cj, string(auth))
		require.NoError(t, err)
		assert.Equal(t, data, decrypted)
	})

	t.Run("decrypt with wrong auth fails", func(t *testing.T) {
		data := []byte("secret-data-for-wrong-auth-test")
		auth := []byte("correct-password")

		cj, err := EncryptDataV3(data, auth, LightScryptN, LightScryptP)
		require.NoError(t, err)

		_, err = DecryptDataV3(cj, "wrong-password")
		assert.Error(t, err)
	})
}

// ---------- writeTemporaryKeyFile ----------

func TestWriteTemporaryKeyFile(t *testing.T) {
	t.Run("writes file and returns path", func(t *testing.T) {
		dir := t.TempDir()
		target := dir + "/testkey.json"
		content := []byte(`{"test": true}`)

		tmpPath, err := writeTemporaryKeyFile(target, content)
		require.NoError(t, err)
		assert.NotEmpty(t, tmpPath)
		assert.FileExists(t, tmpPath)
	})
}

// ---------- helpers ----------

func mustGenerateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	require.NoError(t, err)
	return key
}
