package pqc

import (
	"fmt"
	"math"
)

// Pure-Go algebraic validation of an FN-DSA-512 (Falcon-512) key pair. It runs
// in every build, before any backend sees the key: liboqs does not re-check an
// imported private key, and a basis that Falcon key generation would never
// produce can make its signing sampler loop forever.
//
// The checks follow the Falcon reference implementation (PQClean
// crypto_sign/falcon-512/clean): codec.c trim_i8_decode and modq_decode for the
// encodings, and the two norm rejections of keygen.c keygen().

const (
	fnN         = 512                // ring degree: Z[x]/(x^512 + 1)
	fnQ         = 12289              // modulus q
	fnFGBits    = 6                  // bits per coefficient of f and g (max_fg_bits[9])
	fnBigFBits  = 8                  // bits per coefficient of F (max_FG_bits[9])
	fnHBits     = 14                 // bits per coefficient of h (modq_encode)
	fnMaxBigG   = 127                // largest |G| coefficient that encodes in 8 bits
	fnSqnormMax = 16823              // keygen.c rejects ||f||^2 + ||g||^2 >= 16823
	fnBnormMax  = 16822.4121         // keygen.c fpr_bnorm_max: 1.17^2 * q
	fnFGBytes   = fnN * fnFGBits / 8 // 384
)

// fnPoly is a polynomial of Z[x]/(x^512 + 1), one coefficient per entry.
type fnPoly [fnN]int64

// validateFNDSA512Pair checks that sk (the canonical 1280-byte f||g||F) and pk
// (the bare 896-byte h) form a Falcon-512 key pair that Falcon key generation
// could have produced:
//
//  1. f, g and F decode as trim_i8 (6, 6 and 8 bits per coefficient);
//  2. h decodes as modq (14 bits per coefficient, each < q);
//  3. h*f = g mod q, else ErrKeyMismatch;
//  4. G = F*h mod q, centred, has every |G_i| <= 127;
//  5. f*G - g*F = q over Z[x]/(x^512 + 1);
//  6. ||f||^2 + ||g||^2 < 16823, and the Gram-Schmidt norm of (g, -f) is below
//     16822.4121.
//
// Every failure other than step 3 is ErrInvalidKey. Errors never contain key
// material.
func validateFNDSA512Pair(sk, pk []byte) error {
	if len(sk) != FNDSA512PrivateKeySize || len(pk) != FNDSA512PublicKeySize {
		return fmt.Errorf("%w: FN-DSA-512 key pair must be %d and %d bytes, got %d and %d",
			ErrInvalidKey, FNDSA512PrivateKeySize, FNDSA512PublicKeySize, len(sk), len(pk))
	}
	var f, g, bigF, h fnPoly
	defer func() { clear(f[:]); clear(g[:]); clear(bigF[:]) }()

	if !fnDecodeSigned(&f, sk[:fnFGBytes], fnFGBits) ||
		!fnDecodeSigned(&g, sk[fnFGBytes:2*fnFGBytes], fnFGBits) ||
		!fnDecodeSigned(&bigF, sk[2*fnFGBytes:], fnBigFBits) {
		return fmt.Errorf("%w: FN-DSA-512 private key is not a valid f||g||F encoding", ErrInvalidKey)
	}
	if !fnDecodeModQ(&h, pk) {
		return fmt.Errorf("%w: FN-DSA-512 public key has a coefficient >= %d", ErrInvalidKey, fnQ)
	}

	// 3. The public key must be g/f: h*f = g (mod q).
	hf := fnMulModQ(&h, &f)
	for i := range hf {
		if hf[i] != fnModQ(g[i]) {
			return ErrKeyMismatch
		}
	}

	// 4. G = F*h = F*g/f (mod q) must be small.
	bigG := fnMulModQ(&bigF, &h)
	defer clear(bigG[:])
	for i, v := range bigG {
		if v > fnQ/2 {
			v -= fnQ
		}
		if v < -fnMaxBigG || v > fnMaxBigG {
			return fmt.Errorf("%w: FN-DSA-512 private key: G is out of range", ErrInvalidKey)
		}
		bigG[i] = v
	}

	// 5. The NTRU equation f*G - g*F = q must hold over the integers.
	fG := fnMul(&f, &bigG)
	gF := fnMul(&g, &bigF)
	for i := range fG {
		want := int64(0)
		if i == 0 {
			want = fnQ
		}
		if fG[i]-gF[i] != want {
			return fmt.Errorf("%w: FN-DSA-512 private key does not satisfy the NTRU equation", ErrInvalidKey)
		}
	}

	// 6. The norm bounds of keygen.c.
	return fnCheckNorms(&f, &g)
}

// fnCheckNorms applies the two norm rejections of Falcon keygen.c keygen():
// ||f||^2 + ||g||^2 must be below 16823 and fnBnorm below fpr_bnorm_max. The
// second comparison is written as !(b < max), like keygen.c !fpr_lt, so that
// +Inf and NaN are rejected.
func fnCheckNorms(f, g *fnPoly) error {
	if fnSqnorm(f)+fnSqnorm(g) >= fnSqnormMax {
		return fmt.Errorf("%w: FN-DSA-512 private key: (f, g) is too long", ErrInvalidKey)
	}
	if b := fnBnorm(f, g); !(b < fnBnormMax) {
		return fmt.Errorf("%w: FN-DSA-512 private key: orthogonalised (f, g) is too long", ErrInvalidKey)
	}
	return nil
}

// fnDecodeSigned decodes len(dst) two's-complement coefficients of bits bits
// each from the big-endian bitstream src (codec.c trim_i8_decode). It rejects
// the value -2^(bits-1) and a src whose length is not exactly
// len(dst)*bits/8 bytes.
func fnDecodeSigned(dst *fnPoly, src []byte, bits uint) bool {
	if len(src)*8 != len(dst)*int(bits) {
		return false
	}
	mask := uint32(1)<<bits - 1
	sign := uint32(1) << (bits - 1)
	var acc uint32
	var accLen uint
	u := 0
	for _, b := range src {
		acc = acc<<8 | uint32(b)
		accLen += 8
		for accLen >= bits {
			accLen -= bits
			w := (acc >> accLen) & mask
			if w == sign {
				return false
			}
			v := int64(w)
			if w&sign != 0 {
				v -= int64(1) << bits
			}
			dst[u] = v
			u++
		}
	}
	return u == len(dst)
}

// fnDecodeModQ decodes the 14-bit big-endian bitstream src into dst (codec.c
// modq_decode), rejecting any value >= q.
func fnDecodeModQ(dst *fnPoly, src []byte) bool {
	if len(src)*8 != len(dst)*fnHBits {
		return false
	}
	var acc uint32
	var accLen uint
	u := 0
	for _, b := range src {
		acc = acc<<8 | uint32(b)
		accLen += 8
		if accLen >= fnHBits {
			accLen -= fnHBits
			w := (acc >> accLen) & (1<<fnHBits - 1)
			if w >= fnQ {
				return false
			}
			dst[u] = int64(w)
			u++
		}
	}
	return u == len(dst)
}

// fnModQ reduces v into [0, q).
func fnModQ(v int64) int64 {
	v %= fnQ
	if v < 0 {
		v += fnQ
	}
	return v
}

// fnMul returns a*b in Z[x]/(x^512 + 1), where x^512 = -1. It is meant for
// small coefficients; inputs reduced mod q keep every sum below 2^37.
func fnMul(a, b *fnPoly) fnPoly {
	var c fnPoly
	for i, ai := range a {
		if ai == 0 {
			continue
		}
		for j, bj := range b {
			if k := i + j; k < fnN {
				c[k] += ai * bj
			} else {
				c[k-fnN] -= ai * bj
			}
		}
	}
	return c
}

// fnMulModQ returns a*b in Z_q[x]/(x^512 + 1), coefficients in [0, q).
func fnMulModQ(a, b *fnPoly) fnPoly {
	var ar, br fnPoly
	for i := range a {
		ar[i], br[i] = fnModQ(a[i]), fnModQ(b[i])
	}
	c := fnMul(&ar, &br)
	clear(ar[:])
	clear(br[:])
	for i := range c {
		c[i] = fnModQ(c[i])
	}
	return c
}

// fnSqnorm returns the squared Euclidean norm of the coefficients of a.
func fnSqnorm(a *fnPoly) int64 {
	var s int64
	for _, v := range a {
		s += v * v
	}
	return s
}

// fnBnorm returns the squared norm of (q*adj(f)/(f*adj(f)+g*adj(g)),
// q*adj(g)/(f*adj(f)+g*adj(g))), the vector keygen.c compares against
// fpr_bnorm_max. Pointwise over the evaluations at the n roots
// w_k = e^(i*pi*(2k+1)/n) of x^n + 1, each component has squared modulus
// q^2*|f(w_k)|^2/D_k^2 (resp. g) with D_k = |f(w_k)|^2 + |g(w_k)|^2, so the
// pair contributes q^2/D_k. Parseval for this transform,
// sum_k |a(w_k)|^2 = n*||a||^2, gives the coefficient-domain norm
// (1/n) * sum_k q^2/D_k. A zero D_k yields +Inf.
func fnBnorm(f, g *fnPoly) float64 {
	var tw [2 * fnN]complex128 // tw[m] = e^(i*pi*m/n)
	for m := range tw {
		s, c := math.Sincos(math.Pi * float64(m) / fnN)
		tw[m] = complex(c, s)
	}
	var sum float64
	for k := range fnN {
		var fh, gh complex128
		step, idx := 2*k+1, 0
		for j := range fnN {
			fh += complex(float64(f[j]), 0) * tw[idx]
			gh += complex(float64(g[j]), 0) * tw[idx]
			idx = (idx + step) % (2 * fnN)
		}
		d := real(fh)*real(fh) + imag(fh)*imag(fh) + real(gh)*real(gh) + imag(gh)*imag(gh)
		sum += fnQ * fnQ / d
	}
	return sum / fnN
}
