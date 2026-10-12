package keys_test

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scalar32 returns v as 32 big-endian bytes.
func scalar32(v *big.Int) []byte {
	return v.FillBytes(make([]byte, 32))
}

// rangeCases covers the secp256k1 private key bounds [1, n-1].
func rangeCases() []struct {
	name    string
	value   *big.Int
	wantErr bool
} {
	n := btcec.S256().N
	one := big.NewInt(1)
	max256 := new(big.Int).Sub(new(big.Int).Lsh(one, 256), one)
	return []struct {
		name    string
		value   *big.Int
		wantErr bool
	}{
		{"zero", big.NewInt(0), true},
		{"one", big.NewInt(1), false},
		{"n-1", new(big.Int).Sub(n, one), false},
		{"n", new(big.Int).Set(n), true},
		{"n+1", new(big.Int).Add(n, one), true},
		{"2^256-1", max256, true},
	}
}

func TestGetPrivateKeyFromBytes_Range(t *testing.T) {
	for _, tt := range rangeCases() {
		t.Run(tt.name, func(t *testing.T) {
			b := scalar32(tt.value)
			pk, err := keys.GetPrivateKeyFromBytes(b)
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, keys.ErrInvalidPrivateKey)
				assert.Nil(t, pk)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, pk)
			assert.Equal(t, b, pk.Serialize(), "key must not be reduced or altered")
		})
	}
}

func TestGetPrivateKeyFromHex_Range(t *testing.T) {
	for _, tt := range rangeCases() {
		t.Run(tt.name, func(t *testing.T) {
			b := scalar32(tt.value)
			pk, err := keys.GetPrivateKeyFromHex(hex.EncodeToString(b))
			if tt.wantErr {
				require.Error(t, err)
				assert.ErrorIs(t, err, keys.ErrInvalidPrivateKey)
				assert.Nil(t, pk)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, pk)
			assert.Equal(t, b, pk.Serialize(), "key must not be reduced or altered")
		})
	}
}
