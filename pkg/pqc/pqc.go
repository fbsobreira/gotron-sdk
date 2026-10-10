// Package pqc implements the post-quantum account keys of TIP-899: the
// FN-DSA-512 (Falcon-512) and ML-DSA-44 (FIPS 204) signature schemes, TRON
// address derivation from a post-quantum public key, and parsing of raw key
// material.
//
// Every Sign and Verify operates on a 32-byte digest: for a transaction that
// is the raw 32-byte transaction ID, exactly as TRON nodes verify it.
//
// A post-quantum address is 0x41 || Keccak-256(public key)[12:32], where the
// public key is in its canonical wire form (896 bytes for FN-DSA-512, 1312
// bytes for ML-DSA-44).
//
// ML-DSA-44 is implemented in pure Go. FN-DSA-512 cryptography needs a Falcon
// backend that is only compiled in with the "falcon" build tag; without it,
// FN-DSA-512 keys can be parsed (and the key pair validated in pure Go) and
// their addresses derived, but Sign, Verify and GenerateKey return
// ErrFalconUnavailable. The same error, wrapped with the reason, is returned
// when the backend is compiled in but the installed liboqs was built without
// Falcon-512.
//
// To enable FN-DSA-512, build with CGO_ENABLED=1 and -tags falcon, with liboqs
// installed and discoverable through pkg-config (pkg-config --cflags --libs
// liboqs). Build liboqs as a shared library (-DBUILD_SHARED_LIBS=ON); a static
// liboqs lists OpenSSL only in Requires.private, so also set
// CGO_LDFLAGS="-lcrypto" (or PKG_CONFIG="pkg-config --static") to avoid
// undefined OpenSSL symbols. The backend links liboqs through cgo and uses
// its "Falcon-512" algorithm (compressed signatures), byte-compatible with
// the BouncyCastle verifier in TRON nodes. liboqs lists Falcon as Tier 3
// ("upstream maintenance TBD"); a pure-Go verifier is tracked as GOT-77. The
// default build stays free of cgo.
//
// Experimental: TIP-899 is a draft and is live only on the Nile testnet. The
// APIs and encodings in this package may change before mainnet activation.
package pqc

import (
	"errors"
	"fmt"
	"io"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// Sentinel errors returned (possibly wrapped) by this package.
var (
	// ErrUnsupportedScheme is returned for a PQScheme this package does not implement.
	ErrUnsupportedScheme = errors.New("pqc: unsupported scheme")
	// ErrInvalidKey is returned for key material of the wrong length or encoding.
	ErrInvalidKey = errors.New("pqc: invalid key")
	// ErrInvalidDigest is returned when a digest is not exactly DigestSize bytes.
	ErrInvalidDigest = errors.New("pqc: digest must be 32 bytes")
	// ErrKeyMismatch is returned when a supplied public key does not belong to the private key.
	ErrKeyMismatch = errors.New("pqc: public key does not match private key")
	// ErrFalconUnavailable is returned for FN-DSA-512 cryptographic operations
	// in a build without a Falcon backend, and wrapped with the reason when the
	// backend is compiled in but unusable (for example liboqs built without
	// Falcon-512). Enable it by building with
	// CGO_ENABLED=1 and -tags falcon, with liboqs installed and discoverable
	// through pkg-config.
	ErrFalconUnavailable = errors.New("pqc: FN-DSA-512 backend unavailable (build with CGO_ENABLED=1 -tags falcon and liboqs installed, discoverable via pkg-config)")
)

// Sizes, in bytes, of the TIP-899 wire encodings.
const (
	// DigestSize is the length of the message every Sign and Verify takes.
	DigestSize = 32

	// FNDSA512PublicKeySize is the bare Falcon-512 public key h, without the
	// 0x09 reference header.
	FNDSA512PublicKeySize = 896
	// FNDSA512PrivateKeySize is the canonical Falcon-512 private key f||g||F,
	// without the 0x59 reference header.
	FNDSA512PrivateKeySize = 1280
	// FNDSA512ExtendedPrivateKeySize is f||g||F||h: the private key followed by
	// the bare public key.
	FNDSA512ExtendedPrivateKeySize = FNDSA512PrivateKeySize + FNDSA512PublicKeySize
	// FNDSA512SignatureMinSize is the shortest valid FN-DSA-512 signature.
	FNDSA512SignatureMinSize = 617
	// FNDSA512SignatureMaxSize is the longest valid FN-DSA-512 signature.
	FNDSA512SignatureMaxSize = 667
	// FNDSA512SignatureHeader is the first byte of every FN-DSA-512 signature
	// (0x39 || 40-byte nonce || compressed s2).
	FNDSA512SignatureHeader = 0x39

	// MLDSA44PublicKeySize is the FIPS 204 ML-DSA-44 public key rho||t1.
	MLDSA44PublicKeySize = 1312
	// MLDSA44PrivateKeySize is the FIPS 204 expanded ML-DSA-44 private key
	// rho||K||tr||s1||s2||t0.
	MLDSA44PrivateKeySize = 2560
	// MLDSA44SeedSize is the ML-DSA-44 key-generation seed.
	MLDSA44SeedSize = 32
	// MLDSA44SignatureSize is the fixed ML-DSA-44 signature length.
	MLDSA44SignatureSize = 2420
)

// PrivateKey is a TIP-899 post-quantum private key.
//
// The interface is closed: it can only be implemented inside this package, so
// keys are obtained only through GenerateKey, NewMLDSA44FromSeed or
// ParsePrivateKey, and methods may be added without breaking callers.
// Implementations never print key material through the fmt package.
type PrivateKey interface {
	// Scheme reports the signature scheme of the key.
	Scheme() core.PQScheme
	// Public returns the matching public key.
	Public() *PublicKey
	// Sign signs a 32-byte digest. ML-DSA-44 signatures are hedged
	// (randomized); FN-DSA-512 signatures need the Falcon backend.
	Sign(digest []byte) ([]byte, error)
	// Bytes returns a copy of the canonical TRON private key encoding:
	// 1280-byte f||g||F for FN-DSA-512, 2560-byte expanded key for ML-DSA-44.
	Bytes() []byte
	// Seed returns a copy of the 32-byte ML-DSA-44 seed when known, otherwise
	// nil. FN-DSA-512 keys always return nil.
	Seed() []byte

	// isPrivateKey restricts implementations to this package.
	isPrivateKey()
}

// GenerateKey creates a new private key for scheme.
//
// For ML-DSA-44 a 32-byte seed is read from rand (crypto/rand when rand is
// nil) and expanded with NewMLDSA44FromSeed. For FN-DSA-512 the key pair comes
// from the Falcon backend, which uses its own random number generator: rand is
// ignored, and without -tags falcon ErrFalconUnavailable is returned.
func GenerateKey(scheme core.PQScheme, rand io.Reader) (PrivateKey, error) {
	switch scheme {
	case core.PQScheme_ML_DSA_44:
		return generateMLDSA44(rand)
	case core.PQScheme_FN_DSA_512:
		return generateFNDSA512()
	default:
		return nil, unsupported(scheme)
	}
}

// ParsePrivateKey parses raw private key material for scheme. pub is optional
// for some encodings; an empty pub is treated as absent.
//
// ML-DSA-44: priv is a 32-byte seed or a 2560-byte expanded key. If pub is
// given it must equal the derived public key, else ErrKeyMismatch. An expanded
// key is checked for internal consistency: its tr must equal SHAKE256 of the
// public key derived from it, else ErrInvalidKey. That check does not detect
// every possible corruption (a damaged K or t0, for example, goes unnoticed).
//
// FN-DSA-512: priv is the 1280-byte f||g||F, the same with a leading 0x59
// header (1281 bytes), or the 2176-byte extended form f||g||F||h whose
// trailing 896 bytes are the public key. A Falcon public key cannot be derived
// from f||g||F, so pub (896 bytes, or 897 with a leading 0x09) is required for
// the 1280/1281-byte forms; for the extended form it is optional but must
// match the embedded key. In every build the pair is validated algebraically
// in pure Go before any backend sees it: f, g, F and h must decode, h*f = g
// (mod q), F must complete an NTRU basis with a small G, and (f, g) must meet
// the norm bounds of Falcon key generation. A pk that does not belong to the
// sk (h*f != g) returns ErrKeyMismatch; any other malformed key returns
// ErrInvalidKey. When a Falcon backend is available, the pair is additionally
// checked by signing and verifying a fixed digest (ErrKeyMismatch on failure).
func ParsePrivateKey(scheme core.PQScheme, priv, pub []byte) (PrivateKey, error) {
	switch scheme {
	case core.PQScheme_ML_DSA_44:
		return parseMLDSA44(priv, pub)
	case core.PQScheme_FN_DSA_512:
		return parseFNDSA512(priv, pub)
	default:
		return nil, unsupported(scheme)
	}
}

func unsupported(scheme core.PQScheme) error {
	return fmt.Errorf("%w: %d", ErrUnsupportedScheme, int32(scheme))
}

func checkDigest(digest []byte) error {
	if len(digest) != DigestSize {
		return fmt.Errorf("%w: got %d bytes", ErrInvalidDigest, len(digest))
	}
	return nil
}

// formatPrivateKey renders a private key without any key material.
func formatPrivateKey(k PrivateKey) string {
	return fmt.Sprintf("pqc.PrivateKey(%s, %s)", k.Scheme(), k.Public().Address())
}

// writePrivateKey implements fmt.Formatter for private keys so that no verb
// (including %x and %d) can reach the underlying key bytes.
func writePrivateKey(f fmt.State, k PrivateKey) {
	_, _ = io.WriteString(f, formatPrivateKey(k))
}
