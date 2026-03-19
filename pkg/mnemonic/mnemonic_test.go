package mnemonic_test

import (
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
			assert.Error(t, err, "entropy %d should be rejected", bits)
			assert.Contains(t, err.Error(), "invalid entropy size")
		})
	}
}
