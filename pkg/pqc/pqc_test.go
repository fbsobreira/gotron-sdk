package pqc

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func testDigest(fill byte) []byte {
	d := make([]byte, DigestSize)
	for i := range d {
		d[i] = fill + byte(i)
	}
	return d
}

func TestUnsupportedScheme(t *testing.T) {
	schemes := []core.PQScheme{core.PQScheme_UNKNOWN_PQ_SCHEME, core.PQScheme(99)}
	for _, s := range schemes {
		t.Run(s.String(), func(t *testing.T) {
			_, err := ParsePublicKey(s, make([]byte, MLDSA44PublicKeySize))
			require.ErrorIs(t, err, ErrUnsupportedScheme)

			_, err = ParsePrivateKey(s, make([]byte, MLDSA44SeedSize), nil)
			require.ErrorIs(t, err, ErrUnsupportedScheme)

			_, err = GenerateKey(s, nil)
			require.ErrorIs(t, err, ErrUnsupportedScheme)
		})
	}

	t.Run("zero PublicKey", func(t *testing.T) {
		var k PublicKey
		ok, err := k.Verify(testDigest(0), make([]byte, MLDSA44SignatureSize))
		require.ErrorIs(t, err, ErrUnsupportedScheme)
		assert.False(t, ok)
	})
}

func TestPrivateKeyFormattingDoesNotLeak(t *testing.T) {
	seed := mustHex(t, mlFixtureSeedHex)
	mlSeeded, err := NewMLDSA44FromSeed(seed)
	require.NoError(t, err)
	mlExpanded, err := ParsePrivateKey(core.PQScheme_ML_DSA_44, mlSeeded.Bytes(), nil)
	require.NoError(t, err)
	fn, err := ParsePrivateKey(core.PQScheme_FN_DSA_512,
		mustHex(t, fnPairPrivKeyHex), mustHex(t, fnPairPubKeyHex))
	require.NoError(t, err)

	tests := []struct {
		name    string
		key     PrivateKey
		secrets [][]byte
		prefix  string
	}{
		{"ML seeded", mlSeeded, [][]byte{seed, mlSeeded.Bytes()}, "pqc.PrivateKey(ML_DSA_44, " + mlFixtureAddress + ")"},
		{"ML expanded", mlExpanded, [][]byte{mlExpanded.Bytes()}, "pqc.PrivateKey(ML_DSA_44, " + mlFixtureAddress + ")"},
		{"FN", fn, [][]byte{fn.Bytes()}, "pqc.PrivateKey(FN_DSA_512, " + fnFixtureAddress + ")"},
	}
	verbs := []string{"%v", "%+v", "%#v", "%s", "%x", "%X", "%d", "%q"}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.prefix, fmt.Sprintf("%v", tt.key))
			assert.Equal(t, tt.prefix, fmt.Sprintf("%#v", tt.key))
			for _, verb := range verbs {
				out := strings.ToLower(fmt.Sprintf(verb, tt.key))
				for _, secret := range tt.secrets {
					// Any 16-byte window of the secret would already be a leak.
					assert.NotContains(t, out, hex.EncodeToString(secret[:16]), "verb %s", verb)
					assert.NotContains(t, out, hex.EncodeToString(secret[len(secret)-16:]), "verb %s", verb)
					assert.NotContains(t, out, strings.Trim(fmt.Sprintf("%d", secret[:8]), "[]"), "verb %s", verb)
				}
			}
		})
	}
}

// The unexported marker method keeps PrivateKey closed to outside packages.
var _ interface{ isPrivateKey() } = PrivateKey(nil)
