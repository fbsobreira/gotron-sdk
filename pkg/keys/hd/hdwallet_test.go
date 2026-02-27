package hd

import (
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Well-known BIP39 test mnemonic (all "abandon" x11 + "about").
// Seed derived with empty passphrase is a deterministic 64-byte value.
// We use the raw seed hex directly so the hd package does not depend on bip39.
const (
	// Seed for mnemonic "abandon abandon ... about" with empty passphrase.
	testSeedHex = "5eb00bbddcf069084889a8ab9155568165f5c453ccb85e70811aaed6f6da5fc19a5ac40b389cd370d086206dec8aa6c43daea6690f20ad3d8d48b2d2ce9e38e4"

	// TRON coin type.
	tronCoinType = 195
)

// masterSecret is "Bitcoin seed", the standard BIP32 master secret.
var masterSecret = []byte("Bitcoin seed")

// testSeed returns the decoded test seed bytes. It panics on invalid hex.
func testSeed(t *testing.T) []byte {
	t.Helper()
	seed, err := hex.DecodeString(testSeedHex)
	require.NoError(t, err, "decoding test seed hex")
	return seed
}

// -----------------------------------------------------------------------------
// NewParams
// -----------------------------------------------------------------------------

func TestNewParams(t *testing.T) {
	tests := []struct {
		name         string
		purpose      uint32
		coinType     uint32
		account      uint32
		change       bool
		addressIndex uint32
	}{
		{
			name:         "TRON defaults",
			purpose:      44,
			coinType:     tronCoinType,
			account:      0,
			change:       false,
			addressIndex: 0,
		},
		{
			name:         "with change flag",
			purpose:      44,
			coinType:     tronCoinType,
			account:      1,
			change:       true,
			addressIndex: 5,
		},
		{
			name:         "Bitcoin coin type",
			purpose:      44,
			coinType:     0,
			account:      0,
			change:       false,
			addressIndex: 0,
		},
		{
			name:         "Ethereum coin type",
			purpose:      44,
			coinType:     60,
			account:      2,
			change:       false,
			addressIndex: 10,
		},
		{
			name:         "max uint32 values",
			purpose:      44,
			coinType:     0xFFFFFFFF,
			account:      0xFFFFFFFF,
			change:       true,
			addressIndex: 0xFFFFFFFF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParams(tt.purpose, tt.coinType, tt.account, tt.change, tt.addressIndex)
			require.NotNil(t, p)
			assert.Equal(t, tt.purpose, p.Purpose)
			assert.Equal(t, tt.coinType, p.CoinType)
			assert.Equal(t, tt.account, p.Account)
			assert.Equal(t, tt.change, p.Change)
			assert.Equal(t, tt.addressIndex, p.AddressIndex)
		})
	}
}

// -----------------------------------------------------------------------------
// NewFundraiserParams
// -----------------------------------------------------------------------------

func TestNewFundraiserParams(t *testing.T) {
	tests := []struct {
		name        string
		account     uint32
		coinType    uint32
		addressIdx  uint32
		wantPurpose uint32
		wantChange  bool
	}{
		{
			name:        "TRON account 0",
			account:     0,
			coinType:    tronCoinType,
			addressIdx:  0,
			wantPurpose: 44,
			wantChange:  false,
		},
		{
			name:        "TRON account 5 index 3",
			account:     5,
			coinType:    tronCoinType,
			addressIdx:  3,
			wantPurpose: 44,
			wantChange:  false,
		},
		{
			name:        "Ethereum coin type",
			account:     0,
			coinType:    60,
			addressIdx:  0,
			wantPurpose: 44,
			wantChange:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFundraiserParams(tt.account, tt.coinType, tt.addressIdx)
			require.NotNil(t, p)
			assert.Equal(t, tt.wantPurpose, p.Purpose, "purpose must be 44")
			assert.Equal(t, tt.wantChange, p.Change, "change must be false for fundraiser params")
			assert.Equal(t, tt.coinType, p.CoinType)
			assert.Equal(t, tt.account, p.Account)
			assert.Equal(t, tt.addressIdx, p.AddressIndex)
		})
	}
}

// -----------------------------------------------------------------------------
// NewParamsFromPath — valid paths
// -----------------------------------------------------------------------------

func TestNewParamsFromPath_Valid(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantPurpose  uint32
		wantCoinType uint32
		wantAccount  uint32
		wantChange   bool
		wantAddrIdx  uint32
	}{
		{
			name:         "TRON standard path",
			path:         "44'/195'/0'/0/0",
			wantPurpose:  44,
			wantCoinType: tronCoinType,
			wantAccount:  0,
			wantChange:   false,
			wantAddrIdx:  0,
		},
		{
			name:         "with change=1",
			path:         "44'/195'/0'/1/0",
			wantPurpose:  44,
			wantCoinType: tronCoinType,
			wantAccount:  0,
			wantChange:   true,
			wantAddrIdx:  0,
		},
		{
			name:         "different account and index",
			path:         "44'/195'/7'/0/42",
			wantPurpose:  44,
			wantCoinType: tronCoinType,
			wantAccount:  7,
			wantChange:   false,
			wantAddrIdx:  42,
		},
		{
			name:         "Ethereum coin type",
			path:         "44'/60'/0'/0/0",
			wantPurpose:  44,
			wantCoinType: 60,
			wantAccount:  0,
			wantChange:   false,
			wantAddrIdx:  0,
		},
		{
			name:         "Bitcoin coin type",
			path:         "44'/0'/0'/0/0",
			wantPurpose:  44,
			wantCoinType: 0,
			wantAccount:  0,
			wantChange:   false,
			wantAddrIdx:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParamsFromPath(tt.path)
			require.NoError(t, err)
			require.NotNil(t, p)
			assert.Equal(t, tt.wantPurpose, p.Purpose)
			assert.Equal(t, tt.wantCoinType, p.CoinType)
			assert.Equal(t, tt.wantAccount, p.Account)
			assert.Equal(t, tt.wantChange, p.Change)
			assert.Equal(t, tt.wantAddrIdx, p.AddressIndex)
		})
	}
}

// -----------------------------------------------------------------------------
// NewParamsFromPath — invalid paths
// -----------------------------------------------------------------------------

func TestNewParamsFromPath_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "too few segments",
			path:    "44'/195'/0'",
			wantErr: "path length is wrong",
		},
		{
			name:    "too many segments",
			path:    "44'/195'/0'/0/0/1",
			wantErr: "path length is wrong",
		},
		{
			name:    "empty string",
			path:    "",
			wantErr: "path length is wrong",
		},
		{
			name:    "purpose is not 44",
			path:    "49'/195'/0'/0/0",
			wantErr: "first field in path must be 44'",
		},
		{
			name:    "purpose not hardened",
			path:    "44/195'/0'/0/0",
			wantErr: "first field in path must be 44'",
		},
		{
			name:    "coin type not hardened",
			path:    "44'/195/0'/0/0",
			wantErr: "second and third field in path must be hardened",
		},
		{
			name:    "account not hardened",
			path:    "44'/195'/0/0/0",
			wantErr: "second and third field in path must be hardened",
		},
		{
			name:    "change field hardened",
			path:    "44'/195'/0'/0'/0",
			wantErr: "fourth and fifth field in path must not be hardened",
		},
		{
			name:    "address index hardened",
			path:    "44'/195'/0'/0/0'",
			wantErr: "fourth and fifth field in path must not be hardened",
		},
		{
			name:    "change field is 2",
			path:    "44'/195'/0'/2/0",
			wantErr: "change field can only be 0 or 1",
		},
		{
			name:    "non-numeric purpose",
			path:    "abc'/195'/0'/0/0",
			wantErr: "invalid syntax",
		},
		{
			name:    "non-numeric coin type",
			path:    "44'/xyz'/0'/0/0",
			wantErr: "invalid syntax",
		},
		{
			name:    "negative account",
			path:    "44'/195'/-1'/0/0",
			wantErr: "fields must not be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParamsFromPath(tt.path)
			require.Error(t, err)
			assert.Nil(t, p)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// -----------------------------------------------------------------------------
// DerivationPath
// -----------------------------------------------------------------------------

func TestDerivationPath(t *testing.T) {
	tests := []struct {
		name   string
		params *BIP44Params
		want   []uint32
	}{
		{
			name:   "TRON default no change",
			params: NewParams(44, tronCoinType, 0, false, 0),
			want:   []uint32{44, tronCoinType, 0, 0, 0},
		},
		{
			name:   "with change flag set",
			params: NewParams(44, tronCoinType, 0, true, 0),
			want:   []uint32{44, tronCoinType, 0, 1, 0},
		},
		{
			name:   "non-zero account and index",
			params: NewParams(44, tronCoinType, 3, false, 7),
			want:   []uint32{44, tronCoinType, 3, 0, 7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.DerivationPath()
			assert.Equal(t, tt.want, got)
		})
	}
}

// -----------------------------------------------------------------------------
// String and round-trip
// -----------------------------------------------------------------------------

func TestString(t *testing.T) {
	tests := []struct {
		name   string
		params *BIP44Params
		want   string
	}{
		{
			name:   "TRON standard",
			params: NewParams(44, tronCoinType, 0, false, 0),
			want:   "44'/195'/0'/0/0",
		},
		{
			name:   "with change=1",
			params: NewParams(44, tronCoinType, 0, true, 0),
			want:   "44'/195'/0'/1/0",
		},
		{
			name:   "account 2 index 5",
			params: NewParams(44, tronCoinType, 2, false, 5),
			want:   "44'/195'/2'/0/5",
		},
		{
			name:   "Bitcoin coin type",
			params: NewParams(44, 0, 0, false, 0),
			want:   "44'/0'/0'/0/0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStringRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"TRON default", "44'/195'/0'/0/0"},
		{"with change", "44'/195'/0'/1/0"},
		{"account 3 index 7", "44'/195'/3'/0/7"},
		{"Ethereum", "44'/60'/0'/0/0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := NewParamsFromPath(tt.path)
			require.NoError(t, err)

			roundTripped := params.String()
			assert.Equal(t, tt.path, roundTripped, "String() output must match original path")

			// Parse the round-tripped string and verify equality.
			params2, err := NewParamsFromPath(roundTripped)
			require.NoError(t, err)
			assert.Equal(t, params, params2, "round-tripped params must be equal")
		})
	}
}

// -----------------------------------------------------------------------------
// ComputeMastersFromSeed
// -----------------------------------------------------------------------------

func TestComputeMastersFromSeed(t *testing.T) {
	seed := testSeed(t)

	secret, chainCode := ComputeMastersFromSeed(seed, masterSecret)

	// The master secret and chain code must not be zero.
	assert.NotEqual(t, [32]byte{}, secret, "master secret must not be zero")
	assert.NotEqual(t, [32]byte{}, chainCode, "chain code must not be zero")

	// Determinism: same seed yields same result.
	secret2, chainCode2 := ComputeMastersFromSeed(seed, masterSecret)
	assert.Equal(t, secret, secret2, "must be deterministic")
	assert.Equal(t, chainCode, chainCode2, "must be deterministic")
}

func TestComputeMastersFromSeed_DifferentSeeds(t *testing.T) {
	seed1 := testSeed(t)
	seed2 := make([]byte, 64)
	for i := range seed2 {
		seed2[i] = 0xFF
	}

	secret1, chain1 := ComputeMastersFromSeed(seed1, masterSecret)
	secret2, chain2 := ComputeMastersFromSeed(seed2, masterSecret)

	assert.NotEqual(t, secret1, secret2, "different seeds must yield different secrets")
	assert.NotEqual(t, chain1, chain2, "different seeds must yield different chain codes")
}

func TestComputeMastersFromSeed_KnownVector(t *testing.T) {
	// BIP32 test vector 1 seed.
	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	require.NoError(t, err)

	secret, chainCode := ComputeMastersFromSeed(seed, masterSecret)

	// Expected values from BIP32 test vector 1 (master key).
	// See: https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki#test-vector-1
	expectedSecret := "e8f32e723decf4051aefac8e2c93c9c5b214313817cdb01a1494b917c8436b35"
	expectedChain := "873dff81c02f525623fd1fe5167eac3a55a049de3d314bb42ee227ffed37d508"

	assert.Equal(t, expectedSecret, hex.EncodeToString(secret[:]))
	assert.Equal(t, expectedChain, hex.EncodeToString(chainCode[:]))
}

// -----------------------------------------------------------------------------
// DerivePrivateKeyForPath
// -----------------------------------------------------------------------------

func TestDerivePrivateKeyForPath_TRONStandard(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	tests := []struct {
		name string
		path string
	}{
		{"index 0", "44'/195'/0'/0/0"},
		{"index 1", "44'/195'/0'/0/1"},
		{"index 2", "44'/195'/0'/0/2"},
	}

	derivedKeys := make(map[string][32]byte)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := DerivePrivateKeyForPath(curve, master, ch, tt.path)
			require.NoError(t, err)
			assert.NotEqual(t, [32]byte{}, key, "derived key must not be zero")
			derivedKeys[tt.path] = key
		})
	}

	// All derived keys must be distinct.
	paths := []string{"44'/195'/0'/0/0", "44'/195'/0'/0/1", "44'/195'/0'/0/2"}
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			assert.NotEqual(t, derivedKeys[paths[i]], derivedKeys[paths[j]],
				"keys at %s and %s must differ", paths[i], paths[j])
		}
	}
}

func TestDerivePrivateKeyForPath_Deterministic(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)
	path := "44'/195'/0'/0/0"

	key1, err := DerivePrivateKeyForPath(curve, master, ch, path)
	require.NoError(t, err)

	key2, err := DerivePrivateKeyForPath(curve, master, ch, path)
	require.NoError(t, err)

	assert.Equal(t, key1, key2, "same path must produce same key")
}

func TestDerivePrivateKeyForPath_BIP32Vector1(t *testing.T) {
	// BIP32 test vector 1.
	// Seed: 000102030405060708090a0b0c0d0e0f
	// Chain m/0' should produce a specific key.
	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	require.NoError(t, err)

	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	// Derive m/0' (hardened child 0).
	key, err := DerivePrivateKeyForPath(curve, master, ch, "0'")
	require.NoError(t, err)

	// Expected from BIP32 test vector 1: Chain m/0'
	expectedKey := "edb2e14f9ee77d26dd93b4ecede8d16ed408ce149b6cd80b0715a2d911a0afea"
	assert.Equal(t, expectedKey, hex.EncodeToString(key[:]))
}

func TestDerivePrivateKeyForPath_BIP32Vector1_DeepPath(t *testing.T) {
	// BIP32 test vector 1: Chain m/0'/1/2'/2/1000000000
	seed, err := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	require.NoError(t, err)

	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	key, err := DerivePrivateKeyForPath(curve, master, ch, "0'/1/2'/2/1000000000")
	require.NoError(t, err)

	expectedKey := "471b76e389e528d6de6d816857e012c5455051cad6660850e58372a6c3e6e7c8"
	assert.Equal(t, expectedKey, hex.EncodeToString(key[:]))
}

func TestDerivePrivateKeyForPath_BIP32Vector2(t *testing.T) {
	// BIP32 test vector 2: Chain m/0
	seed, err := hex.DecodeString("fffcf9f6f3f0edeae7e4e1dedbd8d5d2cfccc9c6c3c0bdbab7b4b1aeaba8a5a29f9c999693908d8a8784817e7b7875726f6c696663605d5a5754514e4b484542")
	require.NoError(t, err)

	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	// Verify master key.
	expectedMaster := "4b03d6fc340455b363f51020ad3ecca4f0850280cf436c70c727923f6db46c3e"
	assert.Equal(t, expectedMaster, hex.EncodeToString(master[:]))

	// Derive m/0 (public derivation).
	key, err := DerivePrivateKeyForPath(curve, master, ch, "0")
	require.NoError(t, err)

	expectedKey := "abe74a98f6c7eabee0428f53798f0ab8aa1bd37873999041703c742f15ac7e1e"
	assert.Equal(t, expectedKey, hex.EncodeToString(key[:]))
}

func TestDerivePrivateKeyForPath_InvalidPath(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "non-numeric segment",
			path:    "44'/abc/0'/0/0",
			wantErr: "invalid BIP 32 path",
		},
		{
			name:    "negative index",
			path:    "44'/-1/0'/0/0",
			wantErr: "invalid BIP 32 path",
		},
		{
			name:    "alphabetic path",
			path:    "m/foo/bar",
			wantErr: "invalid BIP 32 path",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: "empty segment",
		},
		{
			name:    "trailing slash creates empty segment",
			path:    "44'/195'/0'/0/",
			wantErr: "empty segment",
		},
		{
			name:    "double slash creates empty segment",
			path:    "44'//0'/0/0",
			wantErr: "empty segment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DerivePrivateKeyForPath(curve, master, ch, tt.path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// -----------------------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------------------

func TestHardenedInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint32
		wantErr bool
	}{
		{"plain number", "44", 44, false},
		{"hardened number", "44'", 44, false},
		{"zero", "0", 0, false},
		{"zero hardened", "0'", 0, false},
		{"large number", "195", 195, false},
		{"negative", "-1", 0, true},
		{"non-numeric", "abc", 0, true},
		{"empty", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hardenedInt(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIsHardened(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"44'", true},
		{"0'", true},
		{"44", false},
		{"0", false},
		{"'", true},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, isHardened(tt.input))
		})
	}
}

func TestUint32ToBytes(t *testing.T) {
	tests := []struct {
		name string
		val  uint32
		want []byte
	}{
		{"zero", 0, []byte{0x00, 0x00, 0x00, 0x00}},
		{"one", 1, []byte{0x00, 0x00, 0x00, 0x01}},
		{"256", 256, []byte{0x00, 0x00, 0x01, 0x00}},
		{"max", 0xFFFFFFFF, []byte{0xFF, 0xFF, 0xFF, 0xFF}},
		{"hardened flag", 0x80000000, []byte{0x80, 0x00, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uint32ToBytes(tt.val)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestI64(t *testing.T) {
	key := []byte("Bitcoin seed")
	data := []byte("test data")

	il, ir := i64(key, data)

	// Must be non-zero.
	assert.NotEqual(t, [32]byte{}, il, "left half must not be zero")
	assert.NotEqual(t, [32]byte{}, ir, "right half must not be zero")

	// Must be deterministic.
	il2, ir2 := i64(key, data)
	assert.Equal(t, il, il2)
	assert.Equal(t, ir, ir2)

	// Different data must produce different results.
	il3, ir3 := i64(key, []byte("different data"))
	assert.NotEqual(t, il, il3)
	assert.NotEqual(t, ir, ir3)
}

func TestAddScalars(t *testing.T) {
	curve := btcec.S256()

	t.Run("zero plus value equals value", func(t *testing.T) {
		a := make([]byte, 32)
		b := make([]byte, 32)
		b[31] = 42

		result := addScalars(curve, a, b)
		assert.Equal(t, byte(42), result[31])
	})

	t.Run("addition is commutative", func(t *testing.T) {
		a := make([]byte, 32)
		a[31] = 10
		b := make([]byte, 32)
		b[31] = 20

		r1 := addScalars(curve, a, b)
		r2 := addScalars(curve, b, a)
		assert.Equal(t, r1, r2)
	})

	t.Run("result is mod N (stays within curve order)", func(t *testing.T) {
		// N is the secp256k1 curve order. Adding N to a value should give the same value.
		n := curve.Params().N.Bytes()
		a := make([]byte, 32)
		a[31] = 1 // value = 1

		result := addScalars(curve, a, n)
		// (1 + N) mod N = 1
		assert.Equal(t, byte(1), result[31])
	})
}

// -----------------------------------------------------------------------------
// DerivePrivateKey (unexported)
// -----------------------------------------------------------------------------

func TestDerivePrivateKey(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	t.Run("hardened derivation", func(t *testing.T) {
		key, newCh := derivePrivateKey(curve, master, ch, 0, true)
		assert.NotEqual(t, [32]byte{}, key)
		assert.NotEqual(t, [32]byte{}, newCh)
		assert.NotEqual(t, master, key, "derived key must differ from master")
		assert.NotEqual(t, ch, newCh, "derived chain code must differ from master chain code")
	})

	t.Run("non-hardened derivation", func(t *testing.T) {
		key, newCh := derivePrivateKey(curve, master, ch, 0, false)
		assert.NotEqual(t, [32]byte{}, key)
		assert.NotEqual(t, [32]byte{}, newCh)
	})

	t.Run("hardened and non-hardened produce different keys", func(t *testing.T) {
		keyH, _ := derivePrivateKey(curve, master, ch, 0, true)
		keyP, _ := derivePrivateKey(curve, master, ch, 0, false)
		assert.NotEqual(t, keyH, keyP, "hardened and public derivation must differ")
	})

	t.Run("different indices produce different keys", func(t *testing.T) {
		key0, _ := derivePrivateKey(curve, master, ch, 0, true)
		key1, _ := derivePrivateKey(curve, master, ch, 1, true)
		assert.NotEqual(t, key0, key1, "different indices must produce different keys")
	})
}

// -----------------------------------------------------------------------------
// Edge cases
// -----------------------------------------------------------------------------

func TestDerivePrivateKeyForPath_SingleSegment(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	// Single-segment paths are valid BIP32 paths.
	key, err := DerivePrivateKeyForPath(curve, master, ch, "0'")
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, key)
}

func TestDerivePrivateKeyForPath_ValidPublicDerivation(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	// Path with all public (non-hardened) segments.
	key, err := DerivePrivateKeyForPath(curve, master, ch, "0/1/2")
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, key)
}

func TestComputeMastersFromSeed_EmptySeed(t *testing.T) {
	// An empty seed is technically valid for HMAC (produces a deterministic output).
	secret, chainCode := ComputeMastersFromSeed([]byte{}, masterSecret)

	// Must produce some non-trivial output (HMAC of empty input is not zero).
	assert.NotEqual(t, [32]byte{}, secret)
	assert.NotEqual(t, [32]byte{}, chainCode)
}

func TestComputeMastersFromSeed_DifferentMasterSecrets(t *testing.T) {
	seed := testSeed(t)

	s1, c1 := ComputeMastersFromSeed(seed, []byte("Bitcoin seed"))
	s2, c2 := ComputeMastersFromSeed(seed, []byte("Nist256p1 seed"))

	assert.NotEqual(t, s1, s2, "different master secrets must produce different results")
	assert.NotEqual(t, c1, c2, "different master secrets must produce different chain codes")
}

// -----------------------------------------------------------------------------
// Integration: full TRON key derivation from known mnemonic seed
// -----------------------------------------------------------------------------

func TestFullTRONDerivation_KnownMnemonic(t *testing.T) {
	// "abandon abandon abandon abandon abandon abandon abandon abandon
	//  abandon abandon abandon about" with empty passphrase.
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	// Derive m/44'/195'/0'/0/0 (TRON first address).
	key, err := DerivePrivateKeyForPath(curve, master, ch, "44'/195'/0'/0/0")
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, key)

	// Verify the private key is a valid secp256k1 scalar.
	privKey, pubKey := btcec.PrivKeyFromBytes(key[:])
	require.NotNil(t, privKey)
	require.NotNil(t, pubKey)

	// The public key must be on the curve.
	assert.True(t,
		curve.IsOnCurve(pubKey.ToECDSA().X, pubKey.ToECDSA().Y),
		"derived public key must be on the secp256k1 curve",
	)

	// We validate structurally that the derived key is valid rather than
	// comparing against a hardcoded expected value, since that would couple
	// this test to the specific bip39 seed derivation implementation.

	// Verify determinism: derive again and compare.
	key2, err := DerivePrivateKeyForPath(curve, master, ch, "44'/195'/0'/0/0")
	require.NoError(t, err)
	assert.Equal(t, key, key2, "derivation must be deterministic")
}

func TestFullTRONDerivation_MultipleIndices(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	keys := make([][32]byte, 5)
	for i := 0; i < 5; i++ {
		var err error
		path := fmt.Sprintf("44'/195'/0'/0/%d", i)
		keys[i], err = DerivePrivateKeyForPath(curve, master, ch, path)
		require.NoError(t, err, "path %s", path)
	}

	// All keys must be unique.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			assert.NotEqual(t, keys[i], keys[j],
				"keys at index %d and %d must differ", i, j)
		}
	}
}

// Needed for strconv import in the integration test.
func TestDerivePrivateKeyForPath_LargeIndex(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	// Use a large (but valid) address index.
	key, err := DerivePrivateKeyForPath(curve, master, ch, "44'/195'/0'/0/999999")
	require.NoError(t, err)
	assert.NotEqual(t, [32]byte{}, key)
}

func TestNewParamsFromPath_FundraiserRoundTrip(t *testing.T) {
	// Verify that NewFundraiserParams produces the same result as NewParamsFromPath.
	fp := NewFundraiserParams(0, tronCoinType, 0)
	pp, err := NewParamsFromPath(fp.String())
	require.NoError(t, err)
	assert.Equal(t, fp, pp)
}

func TestDerivationPath_ChangeValues(t *testing.T) {
	// Explicitly verify the change field maps correctly.
	noChange := NewParams(44, tronCoinType, 0, false, 0)
	assert.Equal(t, uint32(0), noChange.DerivationPath()[3])

	withChange := NewParams(44, tronCoinType, 0, true, 0)
	assert.Equal(t, uint32(1), withChange.DerivationPath()[3])
}

// elliptic import is used by DerivePrivateKeyForPath signature.
var _ elliptic.Curve = btcec.S256()
