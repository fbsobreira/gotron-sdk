package pqc

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fnEncodeSigned is the inverse of fnDecodeSigned (codec.c trim_i8_encode):
// each coefficient as bits-bit two's complement, big-endian bitstream.
func fnEncodeSigned(p *fnPoly, bits uint) []byte {
	return fnEncodeBits(p, bits)
}

// fnEncodeModQ is the inverse of fnDecodeModQ (codec.c modq_encode). Values
// are not range-checked, so tests can build out-of-range public keys.
func fnEncodeModQ(p *fnPoly) []byte { return fnEncodeBits(p, fnHBits) }

func fnEncodeBits(p *fnPoly, bits uint) []byte {
	out := make([]byte, 0, len(p)*int(bits)/8)
	mask := uint64(1)<<bits - 1
	var acc uint64
	var accLen uint
	for _, v := range p {
		acc = acc<<bits | uint64(v)&mask
		accLen += bits
		for accLen >= 8 {
			accLen -= 8
			out = append(out, byte(acc>>accLen))
		}
	}
	return out
}

// fnSplit decodes a canonical 1280-byte sk into f, g and F.
func fnSplit(t *testing.T, sk []byte) (f, g, bigF fnPoly) {
	t.Helper()
	require.Len(t, sk, FNDSA512PrivateKeySize)
	require.True(t, fnDecodeSigned(&f, sk[:fnFGBytes], fnFGBits), "decode f")
	require.True(t, fnDecodeSigned(&g, sk[fnFGBytes:2*fnFGBytes], fnFGBits), "decode g")
	require.True(t, fnDecodeSigned(&bigF, sk[2*fnFGBytes:], fnBigFBits), "decode F")
	return f, g, bigF
}

func fnJoin(f, g, bigF *fnPoly) []byte {
	sk := fnEncodeSigned(f, fnFGBits)
	sk = append(sk, fnEncodeSigned(g, fnFGBits)...)
	return append(sk, fnEncodeSigned(bigF, fnBigFBits)...)
}

// TestFNDSA512KeyCheck_Bnorm pins the normalisation of fnBnorm on inputs with
// exact answers: for f = 1, g = 0 every |f(w)|^2 + |g(w)|^2 is 1, so the norm
// is q^2; for f = g = 1 it is q^2/2.
func TestFNDSA512KeyCheck_Bnorm(t *testing.T) {
	var zero, one, x fnPoly
	one[0] = 1
	x[1] = 1
	q2 := float64(fnQ * fnQ)
	tests := []struct {
		name string
		f, g *fnPoly
		want float64
	}{
		{"f=1 g=0", &one, &zero, q2},
		{"f=0 g=1", &zero, &one, q2},
		{"f=1 g=1", &one, &one, q2 / 2},
		{"f=x g=0", &x, &zero, q2},
		{"f=1 g=x", &one, &x, q2 / 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fnBnorm(tt.f, tt.g)
			assert.InDelta(t, tt.want, got, tt.want*1e-9)
		})
	}
	assert.True(t, math.IsInf(fnBnorm(&zero, &zero), 1), "f = g = 0 must give +Inf")
}

// TestFNDSA512KeyCheck_Norms drives both rejection branches of fnCheckNorms.
func TestFNDSA512KeyCheck_Norms(t *testing.T) {
	f, g, _ := fnSplit(t, mustHex(t, fnPairPrivKeyHex))
	require.NoError(t, fnCheckNorms(&f, &g))

	var long fnPoly
	for i := range long {
		long[i] = 6 // 512 * 36 = 18432 >= 16823
	}
	var short fnPoly
	short[0] = 1 // ||(1, 0)||^2 = 1, but its Gram-Schmidt norm is q^2
	var zero fnPoly
	tests := []struct {
		name string
		f, g *fnPoly
	}{
		{"squared norm >= 16823", &long, &zero},
		{"squared norm just over", &f, &long},
		{"bnorm q^2", &short, &zero},
		{"bnorm +Inf", &zero, &zero},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorIs(t, fnCheckNorms(tt.f, tt.g), ErrInvalidKey)
		})
	}
}

// TestFNDSA512KeyCheck_PositiveControls checks that every genuine key pair at
// hand passes validateFNDSA512Pair: all 100 BouncyCastle KAT pairs and the
// tron-grpc pair. A sign, bit-order or normalisation bug would reject them.
func TestFNDSA512KeyCheck_PositiveControls(t *testing.T) {
	type pair struct {
		name   string
		sk, pk []byte
	}
	pairs := []pair{{"tron-grpc", mustHex(t, fnPairPrivKeyHex), mustHex(t, fnPairPubKeyHex)}}
	kats := loadFalconKAT(t)
	require.Len(t, kats, 100)
	for _, e := range kats {
		require.Len(t, e.sk, FNDSA512PrivateKeySize+1)
		require.Len(t, e.pk, FNDSA512PublicKeySize+1)
		pairs = append(pairs, pair{fmt.Sprintf("KAT %d", e.count), e.sk[1:], e.pk[1:]})
	}

	var maxBnorm float64
	var maxSqnorm int64
	for _, p := range pairs {
		require.NoError(t, validateFNDSA512Pair(p.sk, p.pk), p.name)

		// The codecs round-trip the reference encodings exactly.
		f, g, bigF := fnSplit(t, p.sk)
		require.Equal(t, p.sk, fnJoin(&f, &g, &bigF), "%s: f||g||F round trip", p.name)
		var h fnPoly
		require.True(t, fnDecodeModQ(&h, p.pk), p.name)
		require.Equal(t, p.pk, fnEncodeModQ(&h), "%s: h round trip", p.name)

		maxBnorm = max(maxBnorm, fnBnorm(&f, &g))
		maxSqnorm = max(maxSqnorm, fnSqnorm(&f)+fnSqnorm(&g))
	}
	t.Logf("%d genuine pairs: max ||f||^2+||g||^2 = %d (limit %d), max bnorm = %.4f (limit %.4f)",
		len(pairs), maxSqnorm, fnSqnormMax, maxBnorm, fnBnormMax)
	assert.Less(t, maxSqnorm, int64(fnSqnormMax))
	assert.Less(t, maxBnorm, fnBnormMax)
	assert.Greater(t, maxBnorm, 1000.0, "bnorm of genuine keys is of the order of q")
}

// fnMalformedPair is an FN-DSA-512 private/public key pair that must be
// rejected before any backend sees it. A nil want accepts any error.
type fnMalformedPair struct {
	name   string
	sk, pk []byte
	want   error
}

// fnMalformedPairs builds malformed-but-right-length pairs around the
// tron-grpc key, including the shapes that make liboqs signing loop forever
// (all-0xff, F = f).
func fnMalformedPairs(t *testing.T) []fnMalformedPair {
	t.Helper()
	sk := mustHex(t, fnPairPrivKeyHex)
	pk := mustHex(t, fnPairPubKeyHex)
	f, g, _ := fnSplit(t, sk)
	var zero fnPoly

	var hOne fnPoly // h = 1 makes h*f = g hold for f = g
	hOne[0] = 1
	pkOne := fnEncodeModQ(&hOne)

	var h fnPoly
	require.True(t, fnDecodeModQ(&h, pk))
	pkOutOfRange := func(v int64) []byte {
		bad := h
		bad[7] = v
		return fnEncodeModQ(&bad)
	}

	random := make([]byte, 0, FNDSA512PrivateKeySize)
	for block := sha256.Sum256([]byte("fn malformed random sk")); len(random) < FNDSA512PrivateKeySize; block = sha256.Sum256(block[:]) {
		random = append(random, block[:]...)
	}
	random = random[:FNDSA512PrivateKeySize]

	withByte := func(off int, b byte) []byte {
		c := bytes.Clone(sk)
		c[off] = b
		return c
	}
	kats := loadFalconKAT(t)

	out := []fnMalformedPair{
		{"all 0xff sk", bytes.Repeat([]byte{0xff}, FNDSA512PrivateKeySize), pk, ErrKeyMismatch},
		{"all 0xff sk with h = 1", bytes.Repeat([]byte{0xff}, FNDSA512PrivateKeySize), pkOne, ErrInvalidKey},
		{"F = f", fnJoin(&f, &g, &f), pk, ErrInvalidKey},
		{"F = 0", fnJoin(&f, &g, &zero), pk, ErrInvalidKey},
		{"all-zero sk", make([]byte, FNDSA512PrivateKeySize), pk, ErrInvalidKey},
		{"random sk", random, pk, nil},
		{"f coefficient -32", withByte(0, 0x80|sk[0]&0x03), pk, ErrInvalidKey},
		{"g coefficient -32", withByte(fnFGBytes, 0x80|sk[fnFGBytes]&0x03), pk, ErrInvalidKey},
		{"F coefficient -128", withByte(2*fnFGBytes, 0x80), pk, ErrInvalidKey},
		{"F last coefficient -128", withByte(FNDSA512PrivateKeySize-1, 0x80), pk, ErrInvalidKey},
		{"h coefficient q", sk, pkOutOfRange(fnQ), ErrInvalidKey},
		{"h coefficient 2^14-1", sk, pkOutOfRange(1<<fnHBits - 1), ErrInvalidKey},
		{"pk of KAT 0", sk, kats[0].pk[1:], ErrKeyMismatch},
		{"KAT 1 sk with KAT 0 pk", kats[1].sk[1:], kats[0].pk[1:], ErrKeyMismatch},
	}
	for _, off := range []int{0, 100, fnFGBytes - 1, fnFGBytes, 500, 2*fnFGBytes - 1, 2 * fnFGBytes, 1000, FNDSA512PrivateKeySize - 1} {
		for _, bit := range []byte{0x01, 0x10} {
			c := bytes.Clone(sk)
			c[off] ^= bit
			out = append(out, fnMalformedPair{fmt.Sprintf("sk[%d] ^= 0x%02x", off, bit), c, pk, nil})
		}
	}
	return out
}

func TestFNDSA512KeyCheck_Malformed(t *testing.T) {
	for _, tt := range fnMalformedPairs(t) {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFNDSA512Pair(tt.sk, tt.pk)
			require.Error(t, err)
			if tt.want != nil {
				require.ErrorIs(t, err, tt.want)
			}
			if !errorsIsAny(err, ErrInvalidKey, ErrKeyMismatch) {
				t.Fatalf("error %v is neither ErrInvalidKey nor ErrKeyMismatch", err)
			}
		})
	}
	t.Run("wrong lengths", func(t *testing.T) {
		sk, pk := mustHex(t, fnPairPrivKeyHex), mustHex(t, fnPairPubKeyHex)
		require.ErrorIs(t, validateFNDSA512Pair(sk[1:], pk), ErrInvalidKey)
		require.ErrorIs(t, validateFNDSA512Pair(sk, pk[1:]), ErrInvalidKey)
		require.ErrorIs(t, validateFNDSA512Pair(nil, nil), ErrInvalidKey)
	})
}

// TestFNDSA512_ParsePrivateKeyRejectsMalformedBeforeBackend runs the
// malformed pairs through ParsePrivateKey, in every accepted encoding, with a
// fake backend that would accept anything: the pure-Go check must reject them
// without a single backend call.
func TestFNDSA512_ParsePrivateKeyRejectsMalformedBeforeBackend(t *testing.T) {
	for _, tt := range fnMalformedPairs(t) {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeFalcon{sig: fnSig(FNDSA512SignatureHeader, 650), ok: true}
			useFalcon(t, fake)
			for enc, in := range map[string][2][]byte{
				"1280 + 896": {tt.sk, tt.pk},
				"1281 + 897": {append([]byte{fnPrivateKeyHeader}, tt.sk...), append([]byte{fnPublicKeyHeader}, tt.pk...)},
				"2176":       {append(bytes.Clone(tt.sk), tt.pk...), nil},
			} {
				k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, in[0], in[1])
				require.Error(t, err, enc)
				assert.Nil(t, k, enc)
				if tt.want != nil {
					require.ErrorIs(t, err, tt.want, enc)
				}
			}
			assert.Zero(t, fake.backendCalls(), "backend reached (sign %d, verify %d)", fake.signCalls, fake.verifyCalls)
		})
	}
}

// TestFNDSA512_BackendUnavailableDetail checks that the backend's own reason
// for being unavailable (for example liboqs built without Falcon) reaches the
// caller of Sign, Verify and GenerateKey, still matching ErrFalconUnavailable,
// and that ParsePrivateKey still validates the pair in pure Go.
func TestFNDSA512_BackendUnavailableDetail(t *testing.T) {
	const detail = "liboqs 0.16.0 was built without Falcon-512"
	fake := &fakeFalcon{unavail: fmt.Errorf("%w: %s", ErrFalconUnavailable, detail), sig: fnSig(FNDSA512SignatureHeader, 650), ok: true}
	useFalcon(t, fake)

	k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, mustHex(t, fnPairPrivKeyHex), mustHex(t, fnPairPubKeyHex))
	require.NoError(t, err)
	_, err = ParsePrivateKey(core.PQScheme_FN_DSA_512, bytes.Repeat([]byte{0xff}, FNDSA512PrivateKeySize), mustHex(t, fnPairPubKeyHex))
	require.ErrorIs(t, err, ErrKeyMismatch)

	_, err = k.Sign(testDigest(0))
	require.ErrorIs(t, err, ErrFalconUnavailable)
	assert.Contains(t, err.Error(), detail, "Sign")

	ok, err := k.Public().Verify(testDigest(0), fnSig(FNDSA512SignatureHeader, 650))
	require.ErrorIs(t, err, ErrFalconUnavailable)
	assert.Contains(t, err.Error(), detail, "Verify")
	assert.False(t, ok)

	gen, err := GenerateKey(core.PQScheme_FN_DSA_512, nil)
	require.ErrorIs(t, err, ErrFalconUnavailable)
	assert.Contains(t, err.Error(), detail, "GenerateKey")
	assert.Nil(t, gen)

	assert.Zero(t, fake.backendCalls())
}

func errorsIsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
