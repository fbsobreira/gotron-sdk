package mnemonic_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	bip39 "github.com/fbsobreira/go-bip39"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_WordCount(t *testing.T) {
	m, err := mnemonic.Generate()
	require.NoError(t, err)
	words := strings.Fields(m)
	assert.Len(t, words, 24, "mnemonic should have 24 words")
}

func TestGenerate_ValidBIP39(t *testing.T) {
	m, err := mnemonic.Generate()
	require.NoError(t, err)
	assert.True(t, bip39.IsMnemonicValid(m), "generated mnemonic should be valid BIP39")
}

func TestGenerate_RepeatedCallsRemainValid(t *testing.T) {
	m1, err := mnemonic.Generate()
	require.NoError(t, err)
	m2, err := mnemonic.Generate()
	require.NoError(t, err)
	require.NotEmpty(t, m1)
	require.NotEmpty(t, m2)
	assert.True(t, bip39.IsMnemonicValid(m1))
	assert.True(t, bip39.IsMnemonicValid(m2))
}

func TestGenerate_Uniqueness(t *testing.T) {
	seen := make(map[string]bool, 10)
	for range 10 {
		m, err := mnemonic.Generate()
		require.NoError(t, err)
		assert.False(t, seen[m], "Generate should produce unique mnemonics")
		seen[m] = true
	}
}

func TestGenerate_AllWordsFromBIP39WordList(t *testing.T) {
	m, err := mnemonic.Generate()
	require.NoError(t, err)

	wordList := bip39.GetWordList()
	wordSet := make(map[string]bool, len(wordList))
	for _, w := range wordList {
		wordSet[w] = true
	}

	for _, word := range strings.Fields(m) {
		assert.True(t, wordSet[word], "word %q should be in the BIP39 word list", word)
	}
}

func TestErrInvalidMnemonic(t *testing.T) {
	assert.NotNil(t, mnemonic.ErrInvalidMnemonic)
	assert.Equal(t, "invalid mnemonic given", mnemonic.ErrInvalidMnemonic.Error())
}

func TestGenerate_ProducesValidSeed(t *testing.T) {
	m, err := mnemonic.Generate()
	require.NoError(t, err)
	seed, err := bip39.NewSeedWithErrorChecking(m, "")
	require.NoError(t, err)
	assert.Len(t, seed, 64, "BIP39 seed should be 64 bytes")
}

func TestGenerate_EntropySizes(t *testing.T) {
	tests := []struct {
		bits      int
		wantWords int
	}{
		{128, 12},
		{160, 15},
		{192, 18},
		{224, 21},
		{256, 24},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d_bits_%d_words", tt.bits, tt.wantWords), func(t *testing.T) {
			m, err := mnemonic.Generate(tt.bits)
			require.NoError(t, err)
			words := strings.Fields(m)
			assert.Len(t, words, tt.wantWords, "entropy %d should produce %d words", tt.bits, tt.wantWords)
			assert.True(t, bip39.IsMnemonicValid(m), "mnemonic should be valid BIP39")
		})
	}
}

func TestGenerate_InvalidEntropy(t *testing.T) {
	tests := []int{0, 64, 100, 127, 129, 255, 257, 512}

	for _, bits := range tests {
		t.Run(fmt.Sprintf("%d_bits", bits), func(t *testing.T) {
			_, err := mnemonic.Generate(bits)
			require.Error(t, err, "entropy %d should be rejected", bits)
			assert.Contains(t, err.Error(), "invalid entropy size")
		})
	}
}

// Standard BIP39 test mnemonic used across derive tests.
const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func TestFromSeedAndPassphrase(t *testing.T) {
	t.Run("index 0 produces known private key", func(t *testing.T) {
		private, pub := mnemonic.FromSeedAndPassphrase(testMnemonic, "", 0)
		require.NotNil(t, private)
		require.NotNil(t, pub)

		expected := "b5a4cea271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28" // gitleaks:allow
		assert.Equal(t, expected, hex.EncodeToString(private.Serialize()))
	})

	t.Run("different indices produce different keys", func(t *testing.T) {
		pk0, _ := mnemonic.FromSeedAndPassphrase(testMnemonic, "", 0)
		pk1, _ := mnemonic.FromSeedAndPassphrase(testMnemonic, "", 1)
		require.NotNil(t, pk0)
		require.NotNil(t, pk1)

		assert.NotEqual(t, pk0.Serialize(), pk1.Serialize(),
			"different derivation indices must produce different keys")
	})

	t.Run("same mnemonic and index is deterministic", func(t *testing.T) {
		pk1, pub1 := mnemonic.FromSeedAndPassphrase(testMnemonic, "", 0)
		pk2, pub2 := mnemonic.FromSeedAndPassphrase(testMnemonic, "", 0)
		require.NotNil(t, pk1)
		require.NotNil(t, pk2)

		assert.Equal(t, pk1.Serialize(), pk2.Serialize(),
			"same mnemonic and index must produce the same private key")
		assert.Equal(t, pub1.SerializeCompressed(), pub2.SerializeCompressed(),
			"same mnemonic and index must produce the same public key")
	})

	t.Run("passphrase changes the derived key", func(t *testing.T) {
		pkNoPass, _ := mnemonic.FromSeedAndPassphrase(testMnemonic, "", 0)
		pkWithPass, _ := mnemonic.FromSeedAndPassphrase(testMnemonic, "my-secret", 0)
		require.NotNil(t, pkNoPass)
		require.NotNil(t, pkWithPass)

		assert.NotEqual(t, pkNoPass.Serialize(), pkWithPass.Serialize(),
			"a passphrase must change the derived key")
	})

	t.Run("returned public key matches private key", func(t *testing.T) {
		private, pub := mnemonic.FromSeedAndPassphrase(testMnemonic, "", 0)
		require.NotNil(t, private)
		require.NotNil(t, pub)

		assert.Equal(t, private.PubKey().SerializeCompressed(), pub.SerializeCompressed(),
			"returned public key must correspond to the returned private key")
	})

	t.Run("negative derivation index is rejected", func(t *testing.T) {
		private, pub := mnemonic.FromSeedAndPassphrase(testMnemonic, "", -1)
		assert.Nil(t, private)
		assert.Nil(t, pub)
	})
}
