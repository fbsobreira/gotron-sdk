package mnemonic_test

import (
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Public BIP39 test vectors (https://github.com/trezor/python-mnemonic/blob/master/vectors.json).
var (
	validMnemonic12 = testMnemonic
	validMnemonic24 = strings.Repeat("abandon ", 23) + "art"
	validLegal      = "legal winner thank year wave sausage worth useful legal winner thank yellow"

	// Every word is in the list, but the checksum does not match.
	badChecksum12 = strings.TrimSpace(strings.Repeat("abandon ", 12))
	badChecksum24 = strings.TrimSpace(strings.Repeat("abandon ", 24))
)

// errAny marks a table row that expects some error, of any kind.
var errAny = errors.New("any error")

// tooLargeIndex is 2^31, computed at run time so the file builds where int is 32 bits.
func tooLargeIndex() int {
	v := int64(math.MaxInt32)
	return int(v + 1)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name     string
		mnemonic string
		wantErr  bool
	}{
		{"valid 12 words", validMnemonic12, false},
		{"valid 24 words", validMnemonic24, false},
		{"valid legal winner", validLegal, false},
		{"bad checksum 12 words", badChecksum12, true},
		{"bad checksum 24 words", badChecksum24, true},
		{"wrong word count", "abandon abandon abandon", true},
		{"unknown word", strings.Repeat("abandon ", 11) + "notaword", true},
		{"empty", "", true},
		// Valid words with other spacing would derive a different seed.
		{"double space", strings.Replace(validMnemonic12, " ", "  ", 1), true},
		{"leading space", " " + validMnemonic12, true},
		{"trailing space", validMnemonic12 + " ", true},
		{"tab separator", strings.Replace(validMnemonic12, " ", "\t", 1), true},
		{"trailing newline", validMnemonic12 + "\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mnemonic.Validate(tt.mnemonic)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.True(t, errors.Is(err, mnemonic.ErrInvalidMnemonic),
				"error must match ErrInvalidMnemonic, got %v", err)
		})
	}
}

func TestDeriveKey(t *testing.T) {
	tests := []struct {
		name       string
		mnemonic   string
		passphrase string
		index      int
		wantErr    error // nil means success; errAny means any error
	}{
		{name: "index 0", mnemonic: validMnemonic12, index: 0},
		{name: "index 1 with passphrase", mnemonic: validMnemonic12, passphrase: "TREZOR", index: 1},
		{name: "24 words", mnemonic: validMnemonic24, index: 7},
		{name: "max index 2^31-1", mnemonic: validMnemonic12, index: math.MaxInt32},
		{name: "bad checksum", mnemonic: badChecksum12, index: 0, wantErr: mnemonic.ErrInvalidMnemonic},
		{name: "wrong word count", mnemonic: "abandon abandon", index: 0, wantErr: mnemonic.ErrInvalidMnemonic},
		{name: "negative index", mnemonic: validMnemonic12, index: -1, wantErr: errAny},
		{name: "index 2^31", mnemonic: validMnemonic12, index: tooLargeIndex(), wantErr: errAny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sk, pk, err := mnemonic.DeriveKey(tt.mnemonic, tt.passphrase, tt.index)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.Nil(t, sk)
				assert.Nil(t, pk)
				if tt.wantErr != errAny {
					assert.True(t, errors.Is(err, tt.wantErr), "want %v, got %v", tt.wantErr, err)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, sk)
			require.NotNil(t, pk)

			// DeriveKey must derive exactly what FromSeedAndPassphrase derives.
			wantSK, wantPK := mnemonic.FromSeedAndPassphrase(tt.mnemonic, tt.passphrase, tt.index)
			require.NotNil(t, wantSK)
			assert.Equal(t, wantSK.Serialize(), sk.Serialize())
			assert.Equal(t, wantPK.SerializeCompressed(), pk.SerializeCompressed())
		})
	}
}

func TestDeriveKey_KnownVector(t *testing.T) {
	sk, _, err := mnemonic.DeriveKey(testMnemonic, "", 0)
	require.NoError(t, err)
	assert.Equal(t, "b5a4cea271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28", // gitleaks:allow
		hex.EncodeToString(sk.Serialize()))
}

func TestFromSeedAndPassphrase_Bounds(t *testing.T) {
	tests := []struct {
		name     string
		mnemonic string
		index    int
		wantNil  bool
	}{
		{"max index 2^31-1 is accepted", testMnemonic, math.MaxInt32, false},
		{"index 2^31 is rejected", testMnemonic, tooLargeIndex(), true},
		{"negative index is rejected", testMnemonic, -1, true},
		// The checksum is not checked here, so wallets made from an
		// unchecked mnemonic stay recoverable.
		{"bad checksum still derives", badChecksum12, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sk, pk := mnemonic.FromSeedAndPassphrase(tt.mnemonic, "", tt.index)
			if tt.wantNil {
				assert.Nil(t, sk)
				assert.Nil(t, pk)
				return
			}
			assert.NotNil(t, sk)
			assert.NotNil(t, pk)
		})
	}
}
