package pqc

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeFalcon is a test falconBackend with scripted results. It records a copy
// of the arguments of the last sign and verify calls, and counts every call
// to generate, sign and verify.
type fakeFalcon struct {
	sig     []byte
	signErr error
	ok      bool
	unavail error // returned by availErr

	gotSK     []byte
	gotPK     []byte
	gotDigest []byte
	gotSig    []byte

	generateCalls int
	signCalls     int
	verifyCalls   int
}

func (f *fakeFalcon) availErr() error { return f.unavail }
func (f *fakeFalcon) generate() ([]byte, []byte, error) {
	f.generateCalls++
	return nil, nil, errors.New("fake: no keygen")
}

// sign returns f.sig itself, not a copy, so tests can check that the key type
// copies the backend output before handing it to the caller.
func (f *fakeFalcon) sign(sk, digest []byte) ([]byte, error) {
	f.signCalls++
	f.gotSK, f.gotDigest = bytes.Clone(sk), bytes.Clone(digest)
	return f.sig, f.signErr
}

func (f *fakeFalcon) verify(pk, digest, sig []byte) bool {
	f.verifyCalls++
	f.gotPK, f.gotDigest, f.gotSig = bytes.Clone(pk), bytes.Clone(digest), bytes.Clone(sig)
	return f.ok
}

// backendCalls is the number of generate, sign and verify calls made so far.
func (f *fakeFalcon) backendCalls() int { return f.generateCalls + f.signCalls + f.verifyCalls }

func useFalcon(t *testing.T, b falconBackend) {
	t.Helper()
	prev := falcon
	falcon = b
	t.Cleanup(func() { falcon = prev })
}

func fnSig(header byte, n int) []byte {
	s := make([]byte, n)
	s[0] = header
	return s
}

func TestFNDSA512_ParsePublicKey(t *testing.T) {
	pk := mustHex(t, fnFixturePubKeyHex)
	ref := mustHex(t, fnFixtureRefPubKeyHex)
	require.Equal(t, mustHex(t, fnPairPubKeyHex), pk)
	require.Equal(t, append([]byte{0x09}, pk...), ref)

	badHeader := bytes.Clone(ref)
	badHeader[0] = 0x0a

	tests := []struct {
		name string
		in   []byte
		err  error
	}{
		{"bare 896", pk, nil},
		{"reference 897", ref, nil},
		{"897 wrong header", badHeader, ErrInvalidKey},
		{"895", pk[:895], ErrInvalidKey},
		{"898", append(bytes.Clone(ref), 0), ErrInvalidKey},
		{"empty", nil, ErrInvalidKey},
		{"ML size", make([]byte, MLDSA44PublicKeySize), ErrInvalidKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := ParsePublicKey(core.PQScheme_FN_DSA_512, tt.in)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				assert.Nil(t, k)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, core.PQScheme_FN_DSA_512, k.Scheme())
			assert.Equal(t, pk, k.Bytes())
			assert.Equal(t, fnFixtureAddress, k.Address().String())
			assert.Equal(t, fnFixtureAddressHex, hex.EncodeToString(k.Address()))
		})
	}

	t.Run("input buffer is copied", func(t *testing.T) {
		for _, in := range [][]byte{pk, ref} {
			buf := bytes.Clone(in)
			k, err := ParsePublicKey(core.PQScheme_FN_DSA_512, buf)
			require.NoError(t, err)
			clear(buf)
			assert.Equal(t, pk, k.Bytes(), "input len %d", len(in))
			assert.Equal(t, fnFixtureAddress, k.Address().String(), "input len %d", len(in))
		}
	})

	a, err := ParsePublicKey(core.PQScheme_FN_DSA_512, pk)
	require.NoError(t, err)
	b, err := ParsePublicKey(core.PQScheme_FN_DSA_512, ref)
	require.NoError(t, err)
	assert.True(t, a.Equal(b))

	// Same bytes under a different scheme are a different key.
	ml, err := ParsePublicKey(core.PQScheme_ML_DSA_44, make([]byte, MLDSA44PublicKeySize))
	require.NoError(t, err)
	assert.False(t, a.Equal(ml))
	assert.False(t, a.Equal(nil))
}

func TestFNDSA512_ParsePrivateKey(t *testing.T) {
	sk := mustHex(t, fnPairPrivKeyHex)
	pk := mustHex(t, fnPairPubKeyHex)
	require.Len(t, sk, FNDSA512PrivateKeySize)
	refPK := append([]byte{0x09}, pk...)
	headered := append([]byte{0x59}, sk...)
	badHeadered := append([]byte{0x5a}, sk...)
	extended := append(bytes.Clone(sk), pk...)
	otherPK := bytes.Clone(pk)
	otherPK[10] ^= 0x01

	tests := []struct {
		name string
		priv []byte
		pub  []byte
		err  error
	}{
		{"1280 + 896", sk, pk, nil},
		{"1280 + 897", sk, refPK, nil},
		{"1281 + 896", headered, pk, nil},
		{"2176", extended, nil, nil},
		{"2176 + matching pub", extended, pk, nil},
		{"2176 + matching ref pub", extended, refPK, nil},
		{"1280 without pub", sk, nil, ErrInvalidKey},
		{"1281 without pub", headered, nil, ErrInvalidKey},
		{"1281 wrong header", badHeadered, pk, ErrInvalidKey},
		{"2176 mismatching pub", extended, otherPK, ErrKeyMismatch},
		{"1280 bad pub length", sk, pk[:800], ErrInvalidKey},
		{"1279", sk[:1279], pk, ErrInvalidKey},
		{"2175", extended[:2175], nil, ErrInvalidKey},
		{"empty", nil, pk, ErrInvalidKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, tt.priv, tt.pub)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				assert.Nil(t, k)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, core.PQScheme_FN_DSA_512, k.Scheme())
			assert.Equal(t, sk, k.Bytes())
			assert.Equal(t, pk, k.Public().Bytes())
			assert.Equal(t, fnPairAddress, k.Public().Address().String())
			assert.Nil(t, k.Seed())
		})
	}

	t.Run("defensive copies", func(t *testing.T) {
		in := bytes.Clone(sk)
		k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, in, pk)
		require.NoError(t, err)
		in[0] ^= 0xff
		out := k.Bytes()
		assert.Equal(t, sk, out)
		out[0] ^= 0xff
		assert.Equal(t, sk, k.Bytes())
	})

	t.Run("2176 input buffer is copied", func(t *testing.T) {
		in := bytes.Clone(extended)
		k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, in, nil)
		require.NoError(t, err)
		clear(in)
		assert.Equal(t, sk, k.Bytes())
		assert.Equal(t, pk, k.Public().Bytes())
		assert.Equal(t, fnPairAddress, k.Public().Address().String())
	})
}

// TestFNDSA512_SignRejectsNonConformingBackendOutput checks that the key type
// never returns a signature outside the TRON wire shape, whatever the backend.
func TestFNDSA512_SignRejectsNonConformingBackendOutput(t *testing.T) {
	// Parse before swapping the backend so no pair probe runs against the fake.
	k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512,
		mustHex(t, fnPairPrivKeyHex), mustHex(t, fnPairPubKeyHex))
	require.NoError(t, err)

	tests := []struct {
		name    string
		backend *fakeFalcon
		wantErr bool
	}{
		{"too long 668", &fakeFalcon{sig: fnSig(FNDSA512SignatureHeader, 668)}, true},
		{"too short 616", &fakeFalcon{sig: fnSig(FNDSA512SignatureHeader, 616)}, true},
		{"wrong header", &fakeFalcon{sig: fnSig(0x3a, 650)}, true},
		{"empty", &fakeFalcon{sig: nil}, true},
		{"backend error", &fakeFalcon{signErr: errors.New("boom")}, true},
		{"min 617", &fakeFalcon{sig: fnSig(FNDSA512SignatureHeader, 617)}, false},
		{"max 667", &fakeFalcon{sig: fnSig(FNDSA512SignatureHeader, 667)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useFalcon(t, tt.backend)
			sig, err := k.Sign(testDigest(0))
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, sig)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.backend.sig, sig)
		})
	}

	t.Run("bad digest", func(t *testing.T) {
		useFalcon(t, &fakeFalcon{sig: fnSig(FNDSA512SignatureHeader, 650)})
		_, err := k.Sign(make([]byte, 31))
		require.ErrorIs(t, err, ErrInvalidDigest)
	})
}

func TestFNDSA512_ParsePrivateKeyProbe(t *testing.T) {
	sk := mustHex(t, fnPairPrivKeyHex)
	pk := mustHex(t, fnPairPubKeyHex)

	t.Run("probe verifies", func(t *testing.T) {
		useFalcon(t, &fakeFalcon{sig: fnSig(FNDSA512SignatureHeader, 650), ok: true})
		_, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, sk, pk)
		require.NoError(t, err)
	})
	t.Run("probe fails verify", func(t *testing.T) {
		useFalcon(t, &fakeFalcon{sig: fnSig(FNDSA512SignatureHeader, 650), ok: false})
		_, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, sk, pk)
		require.ErrorIs(t, err, ErrKeyMismatch)
	})
	t.Run("probe sign error", func(t *testing.T) {
		useFalcon(t, &fakeFalcon{signErr: errors.New("boom")})
		_, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, sk, pk)
		require.ErrorIs(t, err, ErrKeyMismatch)
	})
}

func TestFNDSA512_VerifyShape(t *testing.T) {
	pub, err := ParsePublicKey(core.PQScheme_FN_DSA_512, mustHex(t, fnFixturePubKeyHex))
	require.NoError(t, err)
	useFalcon(t, &fakeFalcon{ok: true})

	tests := []struct {
		name   string
		digest []byte
		sig    []byte
		want   bool
		err    error
	}{
		{"well-formed", testDigest(0), fnSig(FNDSA512SignatureHeader, 650), true, nil},
		{"min 617", testDigest(0), fnSig(FNDSA512SignatureHeader, 617), true, nil},
		{"max 667", testDigest(0), fnSig(FNDSA512SignatureHeader, 667), true, nil},
		{"616", testDigest(0), fnSig(FNDSA512SignatureHeader, 616), false, nil},
		{"668", testDigest(0), fnSig(FNDSA512SignatureHeader, 668), false, nil},
		{"wrong header", testDigest(0), fnSig(0x00, 650), false, nil},
		{"empty", testDigest(0), nil, false, nil},
		{"digest 31", testDigest(0)[:31], fnSig(FNDSA512SignatureHeader, 650), false, ErrInvalidDigest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := pub.Verify(tt.digest, tt.sig)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, ok)
		})
	}
}

// TestFNDSA512_BackendInputs checks that the key types hand the backend the
// canonical 1280-byte private key, the bare 896-byte public key, and the
// caller's digest and signature, whatever encoding the keys were parsed from.
func TestFNDSA512_BackendInputs(t *testing.T) {
	sk := mustHex(t, fnPairPrivKeyHex)
	pk := mustHex(t, fnPairPubKeyHex)
	refPK := append([]byte{0x09}, pk...)
	headered := append([]byte{0x59}, sk...)
	extended := append(bytes.Clone(sk), pk...)

	tests := []struct {
		name string
		priv []byte
		pub  []byte
	}{
		{"1280 + 896", sk, pk},
		{"1280 + 897", sk, refPK},
		{"1281 + 896", headered, pk},
		{"1281 + 897", headered, refPK},
		{"2176", extended, nil},
		{"2176 + 897", extended, refPK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantSig := fnSig(FNDSA512SignatureHeader, 650)
			f := &fakeFalcon{sig: bytes.Clone(wantSig), ok: true}
			useFalcon(t, f)

			// The pair probe signs and verifies through the backend.
			k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, tt.priv, tt.pub)
			require.NoError(t, err)
			assert.Equal(t, sk, f.gotSK, "probe sk")
			assert.Equal(t, pk, f.gotPK, "probe pk")
			assert.Equal(t, fnProbeDigest, f.gotDigest, "probe digest")
			assert.Equal(t, wantSig, f.gotSig, "probe sig")

			digest := testDigest(0x20)
			sig, err := k.Sign(digest)
			require.NoError(t, err)
			require.Len(t, f.gotSK, FNDSA512PrivateKeySize)
			assert.Equal(t, k.Bytes(), f.gotSK)
			assert.Equal(t, digest, f.gotDigest)
			assert.Equal(t, wantSig, sig)

			// Sign must return a copy, not the backend's buffer.
			sig[1] ^= 0xff
			assert.Equal(t, wantSig, f.sig)

			vDigest := testDigest(0x30)
			vSig := fnSig(FNDSA512SignatureHeader, 640)
			vSig[5] = 0xaa
			ok, err := k.Public().Verify(vDigest, vSig)
			require.NoError(t, err)
			assert.True(t, ok)
			require.Len(t, f.gotPK, FNDSA512PublicKeySize)
			assert.Equal(t, k.Public().Bytes(), f.gotPK)
			assert.Equal(t, vDigest, f.gotDigest)
			assert.Equal(t, vSig, f.gotSig)
		})
	}

	t.Run("public key parsed from 897", func(t *testing.T) {
		f := &fakeFalcon{ok: true}
		useFalcon(t, f)
		pub, err := ParsePublicKey(core.PQScheme_FN_DSA_512, refPK)
		require.NoError(t, err)
		digest := testDigest(0x40)
		sig := fnSig(FNDSA512SignatureHeader, 620)
		ok, err := pub.Verify(digest, sig)
		require.NoError(t, err)
		assert.True(t, ok)
		require.Len(t, f.gotPK, FNDSA512PublicKeySize)
		assert.Equal(t, pk, f.gotPK)
		assert.Equal(t, pub.Bytes(), f.gotPK)
		assert.Equal(t, digest, f.gotDigest)
		assert.Equal(t, sig, f.gotSig)
	})
}
