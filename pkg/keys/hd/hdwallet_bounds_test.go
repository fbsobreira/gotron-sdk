package hd

import (
	"math/big"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// Path index bounds: every index must be < 2^31 before the hardened bit.
// -----------------------------------------------------------------------------

func TestHardenedInt_Bounds(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint32
		wantErr bool
	}{
		{"max plain", "2147483647", 2147483647, false},
		{"max hardened", "2147483647'", 2147483647, false},
		{"2^31 plain", "2147483648", 0, true},
		{"2^31 hardened", "2147483648'", 0, true},
		{"2^32 plain", "4294967296", 0, true},
		{"2^32 hardened", "4294967296'", 0, true},
		{"overflows int64", "99999999999999999999", 0, true},
		{"overflows int64 hardened", "99999999999999999999'", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hardenedInt(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewParamsFromPath_IndexBounds(t *testing.T) {
	t.Run("max indices are accepted", func(t *testing.T) {
		p, err := NewParamsFromPath("44'/195'/2147483647'/0/2147483647")
		require.NoError(t, err)
		assert.Equal(t, uint32(2147483647), p.Account)
		assert.Equal(t, uint32(2147483647), p.AddressIndex)
	})

	tests := []struct {
		name string
		path string
	}{
		{"account 2^31", "44'/195'/2147483648'/0/0"},
		{"coin type 2^32", "44'/4294967296'/0'/0/0"},
		{"address index 2^31", "44'/195'/0'/0/2147483648"},
		{"address index 2^32", "44'/195'/0'/0/4294967296"},
		{"address index overflows int64", "44'/195'/0'/0/99999999999999999999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParamsFromPath(tt.path)
			require.Error(t, err)
			assert.Nil(t, p)
			assert.Contains(t, err.Error(), "out of range")
		})
	}
}

func TestDerivePrivateKeyForPath_IndexBounds(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"max plain", "2147483647", false},
		{"max hardened", "2147483647'", false},
		{"max in TRON path", "44'/195'/0'/0/2147483647", false},
		{"plain 2^31", "0/2147483648", true},
		{"hardened 2^31", "2147483648'", true},
		{"plain 2^32", "4294967296", true},
		{"hardened 2^32", "4294967296'", true},
		{"overflows int64", "99999999999999999999", true},
		{"TRON path address 2^32", "44'/195'/0'/0/4294967296", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := DerivePrivateKeyForPath(curve, master, ch, tt.path)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid BIP 32 path")
				assert.Equal(t, [32]byte{}, key)
				return
			}
			require.NoError(t, err)
			assert.NotEqual(t, [32]byte{}, key)
		})
	}
}

// Indices that Atoi accepted before ("+1", "-0", "01") must still derive the
// same key, so no existing path changes meaning.
func TestDerivePrivateKeyForPath_LegacyIndexSyntax(t *testing.T) {
	seed := testSeed(t)
	curve := btcec.S256()
	master, ch := ComputeMastersFromSeed(seed, masterSecret)

	tests := []struct {
		name      string
		path      string
		canonical string
	}{
		{"plus sign", "44'/195'/0'/0/+1", "44'/195'/0'/0/1"},
		{"plus sign hardened", "44'/195'/+0'/0/0", "44'/195'/0'/0/0"},
		{"minus zero", "44'/195'/0'/0/-0", "44'/195'/0'/0/0"},
		{"leading zero", "44'/195'/0'/0/01", "44'/195'/0'/0/1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DerivePrivateKeyForPath(curve, master, ch, tt.path)
			require.NoError(t, err)
			want, err := DerivePrivateKeyForPath(curve, master, ch, tt.canonical)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

// -----------------------------------------------------------------------------
// BIP32 key validity checks
// -----------------------------------------------------------------------------

func be32(v *big.Int) []byte {
	return v.FillBytes(make([]byte, 32))
}

func TestCheckPrivateScalar(t *testing.T) {
	curve := btcec.S256()
	n := curve.Params().N
	one := big.NewInt(1)

	tests := []struct {
		name    string
		value   *big.Int
		wantErr bool
	}{
		{"zero", big.NewInt(0), true},
		{"one", big.NewInt(1), false},
		{"n-1", new(big.Int).Sub(n, one), false},
		{"n", new(big.Int).Set(n), true},
		{"n+1", new(big.Int).Add(n, one), true},
		{"2^256-1", new(big.Int).Sub(new(big.Int).Lsh(one, 256), one), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkPrivateScalar(curve, be32(tt.value))
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestChildPrivateScalar(t *testing.T) {
	curve := btcec.S256()
	n := curve.Params().N
	one := big.NewInt(1)

	tests := []struct {
		name    string
		parent  *big.Int
		il      *big.Int
		want    *big.Int
		wantErr bool
	}{
		{"IL = 0 keeps the parent", big.NewInt(5), big.NewInt(0), big.NewInt(5), false},
		{"small sum", big.NewInt(10), big.NewInt(20), big.NewInt(30), false},
		{"IL = n-1 wraps to parent-1", big.NewInt(2), new(big.Int).Sub(n, one), big.NewInt(1), false},
		{"IL = n is rejected", big.NewInt(1), new(big.Int).Set(n), nil, true},
		{"IL = n+1 is rejected", big.NewInt(1), new(big.Int).Add(n, one), nil, true},
		{"IL = 2^256-1 is rejected", big.NewInt(1), new(big.Int).Sub(new(big.Int).Lsh(one, 256), one), nil, true},
		{"child = 0 is rejected", big.NewInt(1), new(big.Int).Sub(n, one), nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := childPrivateScalar(curve, be32(tt.parent), be32(tt.il))
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, [32]byte{}, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, be32(tt.want), got[:])
		})
	}
}

func TestDerivePrivateKeyForPath_InvalidMasterKey(t *testing.T) {
	curve := btcec.S256()
	n := curve.Params().N
	one := big.NewInt(1)
	var ch [32]byte
	ch[0] = 1

	tests := []struct {
		name   string
		master *big.Int
	}{
		{"zero", big.NewInt(0)},
		{"n", new(big.Int).Set(n)},
		{"n+1", new(big.Int).Add(n, one)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var master [32]byte
			copy(master[:], be32(tt.master))
			key, err := DerivePrivateKeyForPath(curve, master, ch, "44'/195'/0'/0/0")
			require.Error(t, err)
			assert.Equal(t, [32]byte{}, key)
		})
	}
}
