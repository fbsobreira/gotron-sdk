package pqc

import (
	"bytes"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// fnPrivateKeyHeader is the Falcon-512 reference header byte (0x50 + logn 9)
// accepted on input and stripped from 1281-byte private keys.
const fnPrivateKeyHeader = 0x59

// falconBackend performs FN-DSA-512 cryptography. Keys are in TRON canonical
// form: sk is the 1280-byte f||g||F and pk the bare 896-byte h; a backend that
// needs reference headers adds and strips them itself. sign must return a
// TRON wire signature (0x39 || nonce || compressed s2) and owns any
// resampling needed to meet the size bounds.
//
// availErr returns nil when the backend can be used, else an error wrapping
// ErrFalconUnavailable that says why (no -tags falcon, or liboqs built
// without Falcon-512, for example).
type falconBackend interface {
	availErr() error
	generate() (sk, pk []byte, err error)
	sign(sk, digest []byte) ([]byte, error)
	verify(pk, digest, sig []byte) bool
}

// fnProbeDigest is the fixed digest signed and verified to check that an
// imported FN-DSA-512 key pair belongs together.
var fnProbeDigest = []byte("gotron-sdk pqc FN-DSA-512 probe!")

// fndsaKey is an FN-DSA-512 (Falcon-512) private key.
type fndsaKey struct {
	sk  []byte // canonical f||g||F
	pub *PublicKey
}

var _ PrivateKey = (*fndsaKey)(nil)

func generateFNDSA512() (PrivateKey, error) {
	if err := falcon.availErr(); err != nil {
		return nil, err
	}
	sk, pk, err := falcon.generate()
	if err != nil {
		return nil, fmt.Errorf("pqc: FN-DSA-512 key generation: %w", err)
	}
	defer clear(sk)
	if len(sk) != FNDSA512PrivateKeySize {
		return nil, fmt.Errorf("%w: backend returned a %d-byte FN-DSA-512 private key", ErrInvalidKey, len(sk))
	}
	pub, err := ParsePublicKey(core.PQScheme_FN_DSA_512, pk)
	if err != nil {
		return nil, fmt.Errorf("pqc: backend returned an invalid FN-DSA-512 public key: %w", err)
	}
	return &fndsaKey{sk: bytes.Clone(sk), pub: pub}, nil
}

func parseFNDSA512(priv, pub []byte) (PrivateKey, error) {
	var embedded []byte
	switch {
	case len(priv) == FNDSA512PrivateKeySize:
	case len(priv) == FNDSA512PrivateKeySize+1 && priv[0] == fnPrivateKeyHeader:
		priv = priv[1:]
	case len(priv) == FNDSA512ExtendedPrivateKeySize:
		embedded = priv[FNDSA512PrivateKeySize:]
		priv = priv[:FNDSA512PrivateKeySize]
	default:
		return nil, fmt.Errorf("%w: FN-DSA-512 private key must be %d, %d (0x59 header) or %d bytes, got %d",
			ErrInvalidKey, FNDSA512PrivateKeySize, FNDSA512PrivateKeySize+1, FNDSA512ExtendedPrivateKeySize, len(priv))
	}

	var pk *PublicKey
	if len(pub) > 0 {
		p, err := ParsePublicKey(core.PQScheme_FN_DSA_512, pub)
		if err != nil {
			return nil, err
		}
		pk = p
	}
	if embedded != nil {
		e, err := ParsePublicKey(core.PQScheme_FN_DSA_512, embedded)
		if err != nil {
			return nil, err
		}
		if pk != nil && !pk.Equal(e) {
			return nil, ErrKeyMismatch
		}
		pk = e
	}
	if pk == nil {
		return nil, fmt.Errorf("%w: an FN-DSA-512 public key is required with a %d-byte private key",
			ErrInvalidKey, FNDSA512PrivateKeySize)
	}

	// Validate the pair algebraically before any backend sees it: liboqs does
	// not check an imported key, and a malformed one can stall its signer.
	if err := validateFNDSA512Pair(priv, pk.key); err != nil {
		return nil, err
	}

	// With a backend compiled in, also sign and verify a probe digest.
	k := &fndsaKey{sk: bytes.Clone(priv), pub: pk}
	if falcon.availErr() == nil {
		sig, err := k.Sign(fnProbeDigest)
		if err != nil {
			return nil, fmt.Errorf("%w: probe signature failed: %w", ErrKeyMismatch, err)
		}
		ok, err := pk.Verify(fnProbeDigest, sig)
		if err != nil {
			return nil, fmt.Errorf("%w: probe verification failed: %w", ErrKeyMismatch, err)
		}
		if !ok {
			return nil, ErrKeyMismatch
		}
	}
	return k, nil
}

// fnSignatureWellFormed reports whether sig has the TRON FN-DSA-512 wire
// shape: the 0x39 header and a length within the valid bounds.
func fnSignatureWellFormed(sig []byte) bool {
	return len(sig) >= FNDSA512SignatureMinSize &&
		len(sig) <= FNDSA512SignatureMaxSize &&
		sig[0] == FNDSA512SignatureHeader
}

// Scheme returns core.PQScheme_FN_DSA_512.
func (k *fndsaKey) Scheme() core.PQScheme { return core.PQScheme_FN_DSA_512 }

// Public returns the matching public key.
func (k *fndsaKey) Public() *PublicKey { return k.pub }

// Sign returns an FN-DSA-512 signature over the 32-byte digest. It returns an
// error wrapping ErrFalconUnavailable without -tags falcon (or when liboqs
// lacks Falcon-512), and an error (never the signature) if the backend
// produces one outside the TRON wire shape.
func (k *fndsaKey) Sign(digest []byte) ([]byte, error) {
	if err := checkDigest(digest); err != nil {
		return nil, err
	}
	if err := falcon.availErr(); err != nil {
		return nil, err
	}
	sig, err := falcon.sign(bytes.Clone(k.sk), bytes.Clone(digest))
	if err != nil {
		return nil, fmt.Errorf("pqc: FN-DSA-512 sign: %w", err)
	}
	if !fnSignatureWellFormed(sig) {
		return nil, fmt.Errorf("pqc: FN-DSA-512 backend returned a non-conforming %d-byte signature", len(sig))
	}
	return bytes.Clone(sig), nil
}

// Bytes returns a copy of the canonical 1280-byte f||g||F private key.
func (k *fndsaKey) Bytes() []byte { return bytes.Clone(k.sk) }

// Seed always returns nil: Falcon seeds are not portable between
// implementations.
func (k *fndsaKey) Seed() []byte { return nil }

// String describes the key without revealing key material.
func (k *fndsaKey) String() string { return formatPrivateKey(k) }

// GoString describes the key without revealing key material.
func (k *fndsaKey) GoString() string { return formatPrivateKey(k) }

// Format implements fmt.Formatter so that no verb prints key material.
func (k *fndsaKey) Format(f fmt.State, _ rune) { writePrivateKey(f, k) }

// isPrivateKey marks fndsaKey as a PrivateKey implementation.
func (k *fndsaKey) isPrivateKey() {}
