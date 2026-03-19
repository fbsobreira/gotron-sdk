package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- pkcs7Unpad ----------

func TestPkcs7Unpad(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{
			name: "valid padding 1 byte block size 16",
			in:   append(bytes.Repeat([]byte{0xAA}, 15), 0x01),
			want: bytes.Repeat([]byte{0xAA}, 15),
		},
		{
			name: "valid padding 4 bytes block size 16",
			in:   append(bytes.Repeat([]byte{0xBB}, 12), 0x04, 0x04, 0x04, 0x04),
			want: bytes.Repeat([]byte{0xBB}, 12),
		},
		{
			name: "valid full block padding 16",
			in:   bytes.Repeat([]byte{0x10}, 16),
			want: []byte{},
		},
		{
			name: "valid padding 2 bytes in 32-byte input",
			in:   append(bytes.Repeat([]byte{0xCC}, 30), 0x02, 0x02),
			want: bytes.Repeat([]byte{0xCC}, 30),
		},
		{
			name: "valid padding 16 bytes in 32-byte input",
			in:   append(bytes.Repeat([]byte{0xDD}, 16), bytes.Repeat([]byte{0x10}, 16)...),
			want: bytes.Repeat([]byte{0xDD}, 16),
		},
		{
			name: "invalid padding byte mismatch",
			in:   []byte{0xAA, 0xAA, 0xAA, 0x03, 0x03, 0x02},
			want: nil,
		},
		{
			name: "zero padding byte",
			in:   []byte{0xAA, 0xBB, 0xCC, 0x00},
			want: nil,
		},
		{
			name: "empty input",
			in:   []byte{},
			want: nil,
		},
		{
			name: "padding exceeds input length",
			in:   []byte{0x05, 0x05, 0x05},
			want: nil,
		},
		{
			name: "padding exceeds aes block size",
			in:   append(bytes.Repeat([]byte{0xAA}, 15), 0x11),
			want: nil,
		},
		{
			name: "all same byte equals length and within block size",
			// 4 bytes all 0x04 - valid PKCS7 padding
			in:   []byte{0x04, 0x04, 0x04, 0x04},
			want: []byte{},
		},
		{
			name: "all same byte exceeds block size",
			// 17 bytes of 0x11 - 0x11 == 17 > aes.BlockSize
			in:   bytes.Repeat([]byte{0x11}, 17),
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pkcs7Unpad(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------- aesCBCDecrypt ----------

func TestAesCBCDecrypt(t *testing.T) {
	t.Run("round-trip encrypt then decrypt", func(t *testing.T) {
		key := make([]byte, 16)
		_, err := rand.Read(key)
		require.NoError(t, err)

		iv := make([]byte, aes.BlockSize)
		_, err = rand.Read(iv)
		require.NoError(t, err)

		plaintext := []byte("hello AES-CBC decrypt test!!!!!!")
		// Pad with PKCS7
		padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
		padded := make([]byte, len(plaintext)+padLen)
		copy(padded, plaintext)
		for i := len(plaintext); i < len(padded); i++ {
			padded[i] = byte(padLen)
		}

		// Encrypt with standard library
		block, err := aes.NewCipher(key)
		require.NoError(t, err)
		cipherText := make([]byte, len(padded))
		encrypter := cipher.NewCBCEncrypter(block, iv)
		encrypter.CryptBlocks(cipherText, padded)

		// Decrypt with aesCBCDecrypt
		got, err := aesCBCDecrypt(key, cipherText, iv)
		require.NoError(t, err)
		assert.Equal(t, plaintext, got)
	})

	t.Run("wrong key fails pkcs7Unpad", func(t *testing.T) {
		key := make([]byte, 16)
		_, err := rand.Read(key)
		require.NoError(t, err)

		iv := make([]byte, aes.BlockSize)
		_, err = rand.Read(iv)
		require.NoError(t, err)

		plaintext := []byte("this is secret data for testing!")
		padLen := aes.BlockSize - (len(plaintext) % aes.BlockSize)
		padded := make([]byte, len(plaintext)+padLen)
		copy(padded, plaintext)
		for i := len(plaintext); i < len(padded); i++ {
			padded[i] = byte(padLen)
		}

		block, err := aes.NewCipher(key)
		require.NoError(t, err)
		cipherText := make([]byte, len(padded))
		encrypter := cipher.NewCBCEncrypter(block, iv)
		encrypter.CryptBlocks(cipherText, padded)

		// Use a different key to decrypt
		wrongKey := make([]byte, 16)
		_, err = rand.Read(wrongKey)
		require.NoError(t, err)

		_, err = aesCBCDecrypt(wrongKey, cipherText, iv)
		assert.ErrorIs(t, err, ErrDecrypt)
	})

	t.Run("wrong IV produces wrong plaintext", func(t *testing.T) {
		key := make([]byte, 16)
		_, err := rand.Read(key)
		require.NoError(t, err)

		iv := make([]byte, aes.BlockSize)
		_, err = rand.Read(iv)
		require.NoError(t, err)

		// Use data that is exactly one block so only the first block
		// (affected by IV in CBC) is present. The unpadding may or may
		// not fail depending on the garbled last byte.
		plaintext := []byte("0123456789abcdef") // 16 bytes
		padLen := aes.BlockSize
		padded := make([]byte, len(plaintext)+padLen)
		copy(padded, plaintext)
		for i := len(plaintext); i < len(padded); i++ {
			padded[i] = byte(padLen)
		}

		block, err := aes.NewCipher(key)
		require.NoError(t, err)
		cipherText := make([]byte, len(padded))
		encrypter := cipher.NewCBCEncrypter(block, iv)
		encrypter.CryptBlocks(cipherText, padded)

		wrongIV := make([]byte, aes.BlockSize)
		_, err = rand.Read(wrongIV)
		require.NoError(t, err)

		result, err := aesCBCDecrypt(key, cipherText, wrongIV)
		if err == nil {
			// If unpadding happened to succeed, the plaintext must differ
			assert.NotEqual(t, plaintext, result,
				"decryption with wrong IV must produce different plaintext")
		} else {
			assert.ErrorIs(t, err, ErrDecrypt)
		}
	})

	t.Run("invalid key size returns error", func(t *testing.T) {
		_, err := aesCBCDecrypt([]byte("short"), make([]byte, 16), make([]byte, 16))
		assert.Error(t, err)
	})
}

// ---------- aesCTRXOR ----------

func TestAesCTRXOR(t *testing.T) {
	t.Run("encrypt then decrypt round-trip", func(t *testing.T) {
		key := make([]byte, 16)
		_, err := rand.Read(key)
		require.NoError(t, err)

		iv := make([]byte, aes.BlockSize)
		_, err = rand.Read(iv)
		require.NoError(t, err)

		plaintext := []byte("CTR mode is symmetric - encrypt then decrypt recovers plaintext")

		encrypted, err := aesCTRXOR(key, plaintext, iv)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, encrypted, "ciphertext must differ from plaintext")

		decrypted, err := aesCTRXOR(key, encrypted, iv)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("different key produces different ciphertext", func(t *testing.T) {
		key1 := make([]byte, 16)
		key2 := make([]byte, 16)
		iv := make([]byte, aes.BlockSize)
		_, err := rand.Read(key1)
		require.NoError(t, err)
		_, err = rand.Read(key2)
		require.NoError(t, err)
		_, err = rand.Read(iv)
		require.NoError(t, err)

		plaintext := []byte("same plaintext different keys")

		enc1, err := aesCTRXOR(key1, plaintext, iv)
		require.NoError(t, err)

		enc2, err := aesCTRXOR(key2, plaintext, iv)
		require.NoError(t, err)

		assert.NotEqual(t, enc1, enc2, "different keys must produce different ciphertext")
	})

	t.Run("different IV produces different ciphertext", func(t *testing.T) {
		key := make([]byte, 16)
		iv1 := make([]byte, aes.BlockSize)
		iv2 := make([]byte, aes.BlockSize)
		_, err := rand.Read(key)
		require.NoError(t, err)
		_, err = rand.Read(iv1)
		require.NoError(t, err)
		_, err = rand.Read(iv2)
		require.NoError(t, err)

		plaintext := []byte("same plaintext different IVs!!")

		enc1, err := aesCTRXOR(key, plaintext, iv1)
		require.NoError(t, err)

		enc2, err := aesCTRXOR(key, plaintext, iv2)
		require.NoError(t, err)

		assert.NotEqual(t, enc1, enc2, "different IVs must produce different ciphertext")
	})

	t.Run("empty plaintext", func(t *testing.T) {
		key := make([]byte, 16)
		iv := make([]byte, aes.BlockSize)
		_, err := rand.Read(key)
		require.NoError(t, err)
		_, err = rand.Read(iv)
		require.NoError(t, err)

		result, err := aesCTRXOR(key, []byte{}, iv)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("invalid key size returns error", func(t *testing.T) {
		_, err := aesCTRXOR([]byte("bad"), []byte("data"), make([]byte, 16))
		assert.Error(t, err)
	})
}

// ---------- RecoverPubkey ----------

func TestRecoverPubkey(t *testing.T) {
	t.Run("recovers correct address from signature", func(t *testing.T) {
		privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		require.NoError(t, err)

		expectedAddr := address.PubkeyToAddress(privKey.PublicKey)

		hash := make([]byte, 32)
		_, err = rand.Read(hash)
		require.NoError(t, err)

		sig, err := crypto.Sign(hash, privKey)
		require.NoError(t, err)
		require.Len(t, sig, 65)

		recovered, err := RecoverPubkey(hash, sig)
		require.NoError(t, err)
		assert.Equal(t, expectedAddr, recovered)
	})

	t.Run("wrong signature length returns error", func(t *testing.T) {
		_, err := RecoverPubkey(make([]byte, 32), make([]byte, 64))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature length")

		_, err = RecoverPubkey(make([]byte, 32), make([]byte, 66))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid signature length")
	})

	t.Run("v >= 27 normalization", func(t *testing.T) {
		privKey, err := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
		require.NoError(t, err)

		expectedAddr := address.PubkeyToAddress(privKey.PublicKey)

		hash := make([]byte, 32)
		_, err = rand.Read(hash)
		require.NoError(t, err)

		sig, err := crypto.Sign(hash, privKey)
		require.NoError(t, err)

		// Add 27 to V byte to simulate Ethereum-style signatures
		sigWithHighV := make([]byte, 65)
		copy(sigWithHighV, sig)
		sigWithHighV[64] += 27

		recovered, err := RecoverPubkey(hash, sigWithHighV)
		require.NoError(t, err)
		assert.Equal(t, expectedAddr, recovered)
	})

	t.Run("invalid signature bytes returns error", func(t *testing.T) {
		hash := make([]byte, 32)
		_, err := rand.Read(hash)
		require.NoError(t, err)

		// 65 bytes of zeros is an invalid ECDSA signature
		badSig := make([]byte, 65)
		_, err = RecoverPubkey(hash, badSig)
		assert.Error(t, err)
	})
}

// ---------- getKDFKey ----------

func TestGetKDFKey(t *testing.T) {
	t.Run("unsupported KDF name returns error", func(t *testing.T) {
		cj := CryptoJSON{
			KDF: "argon2id",
			KDFParams: map[string]interface{}{
				"salt":  hex.EncodeToString(make([]byte, 32)),
				"dklen": 32,
			},
		}
		_, err := getKDFKey(cj, "password")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unsupported KDF")
	})

	t.Run("unsupported PRF for pbkdf2 returns error", func(t *testing.T) {
		cj := CryptoJSON{
			KDF: "pbkdf2",
			KDFParams: map[string]interface{}{
				"salt":  hex.EncodeToString(make([]byte, 32)),
				"dklen": 32,
				"c":     262144,
				"prf":   "hmac-sha512",
			},
		}
		_, err := getKDFKey(cj, "password")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unsupported PBKDF2 PRF")
	})

	t.Run("pbkdf2 with hmac-sha256 succeeds", func(t *testing.T) {
		salt := make([]byte, 32)
		_, err := rand.Read(salt)
		require.NoError(t, err)

		cj := CryptoJSON{
			KDF: "pbkdf2",
			KDFParams: map[string]interface{}{
				"salt":  hex.EncodeToString(salt),
				"dklen": 32,
				"c":     1024,
				"prf":   "hmac-sha256",
			},
		}
		key, err := getKDFKey(cj, "password")
		require.NoError(t, err)
		assert.Len(t, key, 32)
	})

	t.Run("scrypt KDF succeeds", func(t *testing.T) {
		salt := make([]byte, 32)
		_, err := rand.Read(salt)
		require.NoError(t, err)

		cj := CryptoJSON{
			KDF: keyHeaderKDF,
			KDFParams: map[string]interface{}{
				"salt":  hex.EncodeToString(salt),
				"dklen": 32,
				"n":     LightScryptN,
				"r":     scryptR,
				"p":     LightScryptP,
			},
		}
		key, err := getKDFKey(cj, "password")
		require.NoError(t, err)
		assert.Len(t, key, 32)
	})
}
