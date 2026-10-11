package pqc

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/sha3"
	"crypto/subtle"
	"fmt"
	"io"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// mldsaTRSize is the length of tr = SHAKE256(pk) in an expanded ML-DSA-44
// private key rho(32)||K(32)||tr(64)||s1||s2||t0.
const mldsaTRSize = 64

// mldsaKey is an ML-DSA-44 (FIPS 204) private key, used in pure mode with an
// empty context string.
type mldsaKey struct {
	sk  *mldsa44.PrivateKey
	pub *PublicKey
}

var _ PrivateKey = (*mldsaKey)(nil)

// NewMLDSA44FromSeed derives an ML-DSA-44 key from a 32-byte seed
// (FIPS 204 ML-DSA.KeyGen_internal). The seed is retained and returned by Seed.
func NewMLDSA44FromSeed(seed []byte) (PrivateKey, error) {
	if len(seed) != MLDSA44SeedSize {
		return nil, fmt.Errorf("%w: ML-DSA-44 seed must be %d bytes, got %d",
			ErrInvalidKey, MLDSA44SeedSize, len(seed))
	}
	var s [mldsa44.SeedSize]byte
	copy(s[:], seed)
	_, sk := mldsa44.NewKeyFromSeed(&s)
	clear(s[:])
	return newMLDSAKey(sk)
}

func generateMLDSA44(rand io.Reader) (PrivateKey, error) {
	if rand == nil {
		rand = cryptorand.Reader
	}
	seed := make([]byte, MLDSA44SeedSize)
	defer clear(seed)
	if _, err := io.ReadFull(rand, seed); err != nil {
		return nil, fmt.Errorf("pqc: reading ML-DSA-44 seed: %w", err)
	}
	return NewMLDSA44FromSeed(seed)
}

// parseMLDSA44 accepts a 32-byte seed or a 2560-byte expanded private key.
//
// circl unpacks an expanded key without semantic validation, so its tr
// (bytes 64..127) is checked against SHAKE256 of the derived public key. That
// does not detect every corruption: K and t0, for example, do not feed the
// public key. Pass pub to also check the key against a known one.
func parseMLDSA44(priv, pub []byte) (PrivateKey, error) {
	var (
		k   PrivateKey
		err error
	)
	switch len(priv) {
	case MLDSA44SeedSize:
		k, err = NewMLDSA44FromSeed(priv)
	case MLDSA44PrivateKeySize:
		var buf [mldsa44.PrivateKeySize]byte
		copy(buf[:], priv)
		sk := new(mldsa44.PrivateKey)
		sk.Unpack(&buf)
		clear(buf[:])
		mk, mkErr := newMLDSAKey(sk)
		if mkErr != nil {
			return nil, mkErr
		}
		if trErr := checkMLDSATr(priv, mk.pub); trErr != nil {
			return nil, trErr
		}
		k = mk
	default:
		return nil, fmt.Errorf("%w: ML-DSA-44 private key must be %d (seed) or %d bytes, got %d",
			ErrInvalidKey, MLDSA44SeedSize, MLDSA44PrivateKeySize, len(priv))
	}
	if err != nil {
		return nil, err
	}
	if len(pub) == 0 {
		return k, nil
	}
	want, err := ParsePublicKey(core.PQScheme_ML_DSA_44, pub)
	if err != nil {
		return nil, err
	}
	if !want.Equal(k.Public()) {
		return nil, ErrKeyMismatch
	}
	return k, nil
}

// checkMLDSATr reports ErrInvalidKey unless tr, bytes 64..127 of the expanded
// key priv, equals SHAKE256(pub) truncated to 64 bytes (FIPS 204).
func checkMLDSATr(priv []byte, pub *PublicKey) error {
	h := sha3.NewSHAKE256()
	if _, err := h.Write(pub.Bytes()); err != nil {
		return fmt.Errorf("pqc: hashing ML-DSA-44 public key: %w", err)
	}
	var tr [mldsaTRSize]byte
	if _, err := h.Read(tr[:]); err != nil {
		return fmt.Errorf("pqc: hashing ML-DSA-44 public key: %w", err)
	}
	if subtle.ConstantTimeCompare(tr[:], priv[64:64+mldsaTRSize]) != 1 {
		return fmt.Errorf("%w: expanded key tr does not match its public key", ErrInvalidKey)
	}
	return nil
}

// newMLDSAKey wraps sk. The public key is packed and re-parsed from its wire
// bytes so that it does not inherit tr from a (possibly inconsistent) expanded
// private key.
func newMLDSAKey(sk *mldsa44.PrivateKey) (*mldsaKey, error) {
	pk, ok := sk.Public().(*mldsa44.PublicKey)
	if !ok {
		return nil, fmt.Errorf("%w: unexpected ML-DSA-44 public key type", ErrInvalidKey)
	}
	var buf [mldsa44.PublicKeySize]byte
	pk.Pack(&buf)
	pub, err := ParsePublicKey(core.PQScheme_ML_DSA_44, buf[:])
	if err != nil {
		return nil, err
	}
	return &mldsaKey{sk: sk, pub: pub}, nil
}

// Scheme returns core.PQScheme_ML_DSA_44.
func (k *mldsaKey) Scheme() core.PQScheme { return core.PQScheme_ML_DSA_44 }

// Public returns the matching public key.
func (k *mldsaKey) Public() *PublicKey { return k.pub }

// Sign returns a hedged (randomized) ML-DSA-44 signature over the 32-byte
// digest, using an empty context string.
func (k *mldsaKey) Sign(digest []byte) ([]byte, error) {
	return k.signWith(digest, true)
}

// signWith signs digest with an empty context. randomized=false is the
// FIPS 204 deterministic variant (all-zero rnd), used only by tests.
func (k *mldsaKey) signWith(digest []byte, randomized bool) ([]byte, error) {
	if err := checkDigest(digest); err != nil {
		return nil, err
	}
	sig := make([]byte, mldsa44.SignatureSize)
	if err := mldsa44.SignTo(k.sk, bytes.Clone(digest), nil, randomized, sig); err != nil {
		return nil, fmt.Errorf("pqc: ML-DSA-44 sign: %w", err)
	}
	return sig, nil
}

// Bytes returns a copy of the 2560-byte expanded private key.
func (k *mldsaKey) Bytes() []byte {
	var buf [mldsa44.PrivateKeySize]byte
	k.sk.Pack(&buf)
	out := bytes.Clone(buf[:])
	clear(buf[:])
	return out
}

// Seed returns a copy of the 32-byte seed, or nil when the key was imported
// in expanded form.
func (k *mldsaKey) Seed() []byte {
	s := k.sk.Seed()
	if s == nil {
		return nil
	}
	return bytes.Clone(s)
}

// String describes the key without revealing key material.
func (k *mldsaKey) String() string { return formatPrivateKey(k) }

// GoString describes the key without revealing key material.
func (k *mldsaKey) GoString() string { return formatPrivateKey(k) }

// Format implements fmt.Formatter so that no verb prints key material.
func (k *mldsaKey) Format(f fmt.State, _ rune) { writePrivateKey(f, k) }

// isPrivateKey marks mldsaKey as a PrivateKey implementation.
func (k *mldsaKey) isPrivateKey() {}
