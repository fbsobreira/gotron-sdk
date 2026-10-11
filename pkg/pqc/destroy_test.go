package pqc

import (
	"fmt"
	"testing"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertDestroyed checks the observable contract of a destroyed key: Sign
// fails with ErrKeyDestroyed, Bytes and Seed return nil, the public key and
// formatting keep working, and a second Destroy is a no-op.
func assertDestroyed(t *testing.T, k PrivateKey, pub *PublicKey) {
	t.Helper()
	sig, err := k.Sign(testDigest(1))
	require.ErrorIs(t, err, ErrKeyDestroyed)
	assert.Nil(t, sig)
	assert.Nil(t, k.Bytes())
	assert.Nil(t, k.Seed())
	assert.True(t, pub.Equal(k.Public()))
	assert.Equal(t, pub.Address(), k.Public().Address())
	assert.Contains(t, fmt.Sprint(k), pub.Address().String())

	require.NotPanics(t, k.Destroy)
	_, err = k.Sign(testDigest(1))
	require.ErrorIs(t, err, ErrKeyDestroyed)
}

func TestMLDSA44_Destroy(t *testing.T) {
	seeded, err := NewMLDSA44FromSeed(mustHex(t, mlFixtureSeedHex))
	require.NoError(t, err)
	expanded, err := ParsePrivateKey(core.PQScheme_ML_DSA_44, seeded.Bytes(), nil)
	require.NoError(t, err)

	tests := []struct {
		name string
		key  PrivateKey
	}{
		{"from seed", seeded},
		{"expanded", expanded},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k := tc.key
			pub := k.Public()
			_, err := k.Sign(testDigest(1))
			require.NoError(t, err)

			k.Destroy()

			mk, ok := k.(*mldsaKey)
			require.True(t, ok)
			assert.Equal(t, mldsa44.PrivateKey{}, *mk.sk, "circl private key not zeroed")
			_, err = mk.signWith(testDigest(1), false)
			require.ErrorIs(t, err, ErrKeyDestroyed)
			assertDestroyed(t, k, pub)
		})
	}
}

func TestFNDSA512_Destroy(t *testing.T) {
	sk := mustHex(t, fnPairPrivKeyHex)
	pk := mustHex(t, fnPairPubKeyHex)

	t.Run("with backend", func(t *testing.T) {
		f := &fakeFalcon{sig: fnSig(FNDSA512SignatureHeader, 650), ok: true}
		useFalcon(t, f)
		k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, sk, pk)
		require.NoError(t, err)
		_, err = k.Sign(testDigest(1))
		require.NoError(t, err)

		fk, ok := k.(*fndsaKey)
		require.True(t, ok)
		secret := fk.sk // same backing array, to check it was zeroed in place
		require.Len(t, secret, FNDSA512PrivateKeySize)
		pub := k.Public()
		signCalls := f.signCalls

		k.Destroy()

		assert.Equal(t, make([]byte, FNDSA512PrivateKeySize), secret, "secret not zeroed in place")
		assertDestroyed(t, k, pub)
		assert.Equal(t, signCalls, f.signCalls, "backend reached after Destroy")
	})

	t.Run("without backend", func(t *testing.T) {
		useFalcon(t, &fakeFalcon{unavail: ErrFalconUnavailable})
		k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, sk, pk)
		require.NoError(t, err)
		pub := k.Public()

		k.Destroy()

		// ErrKeyDestroyed, not ErrFalconUnavailable: the destroyed check
		// comes first.
		_, err = k.Sign(testDigest(1))
		require.NotErrorIs(t, err, ErrFalconUnavailable)
		assertDestroyed(t, k, pub)
	})
}

func TestSchemeAvailable(t *testing.T) {
	t.Run("ML-DSA-44", func(t *testing.T) {
		require.NoError(t, SchemeAvailable(core.PQScheme_ML_DSA_44))
	})
	t.Run("FN-DSA-512 without backend", func(t *testing.T) {
		useFalcon(t, &fakeFalcon{unavail: ErrFalconUnavailable})
		require.ErrorIs(t, SchemeAvailable(core.PQScheme_FN_DSA_512), ErrFalconUnavailable)
	})
	t.Run("FN-DSA-512 with backend", func(t *testing.T) {
		useFalcon(t, &fakeFalcon{})
		require.NoError(t, SchemeAvailable(core.PQScheme_FN_DSA_512))
	})
	for _, s := range []core.PQScheme{core.PQScheme_UNKNOWN_PQ_SCHEME, core.PQScheme(99)} {
		t.Run(s.String(), func(t *testing.T) {
			require.ErrorIs(t, SchemeAvailable(s), ErrUnsupportedScheme)
		})
	}
}
