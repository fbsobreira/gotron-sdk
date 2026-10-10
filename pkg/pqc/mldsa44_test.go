package pqc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMLDSA44_NileKAT(t *testing.T) {
	for _, kat := range mlKATs {
		t.Run(kat.name, func(t *testing.T) {
			seed := make([]byte, MLDSA44SeedSize)
			for i := range seed {
				seed[i] = kat.seed(i)
			}
			k, err := NewMLDSA44FromSeed(seed)
			require.NoError(t, err)

			pk := k.Public().Bytes()
			sk := k.Bytes()
			require.Len(t, pk, MLDSA44PublicKeySize)
			require.Len(t, sk, MLDSA44PrivateKeySize)

			pkSum := sha256.Sum256(pk)
			skSum := sha256.Sum256(sk)
			assert.Equal(t, kat.pkSHA256, hex.EncodeToString(pkSum[:]))
			assert.Equal(t, kat.skSHA256, hex.EncodeToString(skSum[:]))
			assert.Equal(t, seed, k.Seed())
		})
	}
}

func TestMLDSA44_Fixture(t *testing.T) {
	k, err := NewMLDSA44FromSeed(mustHex(t, mlFixtureSeedHex))
	require.NoError(t, err)
	assert.Equal(t, core.PQScheme_ML_DSA_44, k.Scheme())

	pub := k.Public()
	assert.Equal(t, mlFixturePubKeyHex, hex.EncodeToString(pub.Bytes()))
	assert.Equal(t, mlFixtureAddress, pub.Address().String())
	assert.Equal(t, mlFixtureAddressHex, hex.EncodeToString(pub.Address()))

	msg := mustHex(t, mlFixtureMessageHex)
	sig := mustHex(t, mlFixtureSigHex)
	ok, err := pub.Verify(msg, sig)
	require.NoError(t, err)
	assert.True(t, ok)

	parsed, err := ParsePublicKey(core.PQScheme_ML_DSA_44, mustHex(t, mlFixturePubKeyHex))
	require.NoError(t, err)
	assert.True(t, parsed.Equal(pub))

	// Signing entropy in the fixture is all-zero: FIPS 204 deterministic mode.
	det, err := k.(*mldsaKey).signWith(msg, false)
	require.NoError(t, err)
	assert.Equal(t, mlFixtureSigHex, hex.EncodeToString(det))
}

func TestMLDSA44_SignVerify(t *testing.T) {
	k, err := GenerateKey(core.PQScheme_ML_DSA_44, nil)
	require.NoError(t, err)
	other, err := GenerateKey(core.PQScheme_ML_DSA_44, nil)
	require.NoError(t, err)
	require.Len(t, k.Seed(), MLDSA44SeedSize)

	digest := testDigest(0x10)
	sig1, err := k.Sign(digest)
	require.NoError(t, err)
	require.Len(t, sig1, MLDSA44SignatureSize)
	sig2, err := k.Sign(digest)
	require.NoError(t, err)
	assert.NotEqual(t, sig1, sig2, "hedged signatures must differ")

	tamperedSig := bytes.Clone(sig1)
	tamperedSig[100] ^= 0x01
	tamperedDigest := bytes.Clone(digest)
	tamperedDigest[0] ^= 0x01

	tests := []struct {
		name   string
		pub    *PublicKey
		digest []byte
		sig    []byte
		want   bool
		err    error
	}{
		{"valid 1", k.Public(), digest, sig1, true, nil},
		{"valid 2", k.Public(), digest, sig2, true, nil},
		{"tampered sig", k.Public(), digest, tamperedSig, false, nil},
		{"tampered digest", k.Public(), tamperedDigest, sig1, false, nil},
		{"cross key", other.Public(), digest, sig1, false, nil},
		{"short sig", k.Public(), digest, sig1[:MLDSA44SignatureSize-1], false, nil},
		{"long sig", k.Public(), digest, append(bytes.Clone(sig1), 0), false, nil},
		{"empty sig", k.Public(), digest, nil, false, nil},
		{"digest 31", k.Public(), digest[:31], sig1, false, ErrInvalidDigest},
		{"digest 33", k.Public(), append(bytes.Clone(digest), 0), sig1, false, ErrInvalidDigest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := tt.pub.Verify(tt.digest, tt.sig)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, ok)
		})
	}

	for _, n := range []int{0, 31, 33, 64} {
		_, err := k.Sign(make([]byte, n))
		require.ErrorIs(t, err, ErrInvalidDigest, "sign digest len %d", n)
	}
}

func TestMLDSA44_GenerateKeyFromReader(t *testing.T) {
	seed := mustHex(t, mlFixtureSeedHex)
	k, err := GenerateKey(core.PQScheme_ML_DSA_44, bytes.NewReader(seed))
	require.NoError(t, err)
	assert.Equal(t, mlFixturePubKeyHex, hex.EncodeToString(k.Public().Bytes()))

	_, err = GenerateKey(core.PQScheme_ML_DSA_44, bytes.NewReader(seed[:10]))
	require.Error(t, err)
}

func TestMLDSA44_ParsePrivateKey(t *testing.T) {
	seed := mustHex(t, mlFixtureSeedHex)
	ref, err := NewMLDSA44FromSeed(seed)
	require.NoError(t, err)
	pub := ref.Public().Bytes()
	expanded := ref.Bytes()
	other, err := GenerateKey(core.PQScheme_ML_DSA_44, nil)
	require.NoError(t, err)

	tests := []struct {
		name     string
		priv     []byte
		pub      []byte
		err      error
		wantSeed bool
	}{
		{"seed", seed, nil, nil, true},
		{"seed with pub", seed, pub, nil, true},
		{"expanded", expanded, nil, nil, false},
		{"expanded with pub", expanded, pub, nil, false},
		{"seed mismatched pub", seed, other.Public().Bytes(), ErrKeyMismatch, false},
		{"expanded mismatched pub", expanded, other.Public().Bytes(), ErrKeyMismatch, false},
		{"bad pub length", seed, pub[:100], ErrInvalidKey, false},
		{"empty priv", nil, nil, ErrInvalidKey, false},
		{"priv 31", seed[:31], nil, ErrInvalidKey, false},
		{"priv 2559", expanded[:2559], nil, ErrInvalidKey, false},
		{"priv 2561", append(bytes.Clone(expanded), 0), nil, ErrInvalidKey, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := ParsePrivateKey(core.PQScheme_ML_DSA_44, tt.priv, tt.pub)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				assert.Nil(t, k)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, pub, k.Public().Bytes())
			assert.Equal(t, expanded, k.Bytes())
			if tt.wantSeed {
				assert.Equal(t, seed, k.Seed())
			} else {
				assert.Nil(t, k.Seed())
			}

			digest := testDigest(0x40)
			sig, err := k.Sign(digest)
			require.NoError(t, err)
			ok, err := ref.Public().Verify(digest, sig)
			require.NoError(t, err)
			assert.True(t, ok)
		})
	}

	t.Run("bad seed length", func(t *testing.T) {
		_, err := NewMLDSA44FromSeed(seed[:16])
		require.ErrorIs(t, err, ErrInvalidKey)
	})
}

func TestMLDSA44_DefensiveCopies(t *testing.T) {
	seed := mustHex(t, mlFixtureSeedHex)
	k, err := NewMLDSA44FromSeed(seed)
	require.NoError(t, err)
	seed[0] ^= 0xff
	assert.Equal(t, mlFixturePubKeyHex, hex.EncodeToString(k.Public().Bytes()))

	b := k.Bytes()
	b[0] ^= 0xff
	assert.NotEqual(t, b, k.Bytes())
	s := k.Seed()
	s[0] ^= 0xff
	assert.NotEqual(t, s, k.Seed())

	raw := mustHex(t, mlFixturePubKeyHex)
	pub, err := ParsePublicKey(core.PQScheme_ML_DSA_44, raw)
	require.NoError(t, err)
	raw[0] ^= 0xff
	assert.Equal(t, mlFixturePubKeyHex, hex.EncodeToString(pub.Bytes()))
	out := pub.Bytes()
	out[0] ^= 0xff
	assert.Equal(t, mlFixturePubKeyHex, hex.EncodeToString(pub.Bytes()))

	_, err = ParsePublicKey(core.PQScheme_ML_DSA_44, raw[:MLDSA44PublicKeySize-1])
	require.ErrorIs(t, err, ErrInvalidKey)
}

// TestMLDSA44_ExpandedKeyTrCheck checks that an expanded key whose tr
// (bytes 64..127) is not SHAKE256 of its own public key is rejected, with or
// without pub, and pins the documented limitation that corruption elsewhere
// (K, t0) that leaves the public key unchanged is not detected.
func TestMLDSA44_ExpandedKeyTrCheck(t *testing.T) {
	ref, err := NewMLDSA44FromSeed(mustHex(t, mlFixtureSeedHex))
	require.NoError(t, err)
	pub := ref.Public().Bytes()
	expanded := ref.Bytes()

	flip := func(i int) []byte {
		b := bytes.Clone(expanded)
		b[i] ^= 0x01
		return b
	}

	for _, i := range []int{64, 127} {
		for _, withPub := range []bool{false, true} {
			var p []byte
			if withPub {
				p = pub
			}
			k, err := ParsePrivateKey(core.PQScheme_ML_DSA_44, flip(i), p)
			require.ErrorIs(t, err, ErrInvalidKey, "byte %d, pub %v", i, withPub)
			assert.Nil(t, k)
		}
	}

	// Documented limitation: K (byte 40) and t0 (byte 2559) do not feed the
	// derived public key, so flipping them keeps tr consistent and parses.
	for _, i := range []int{40, MLDSA44PrivateKeySize - 1} {
		for _, p := range [][]byte{nil, pub} {
			k, err := ParsePrivateKey(core.PQScheme_ML_DSA_44, flip(i), p)
			require.NoError(t, err, "byte %d", i)
			assert.Equal(t, pub, k.Public().Bytes())
		}
	}
}
