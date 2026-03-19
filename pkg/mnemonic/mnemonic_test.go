package mnemonic_test

import (
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
