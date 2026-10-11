package pqc

import (
	"bytes"
	"fmt"

	"github.com/cloudflare/circl/sign/mldsa/mldsa44"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// fnPublicKeyHeader is the Falcon-512 reference header byte (0x00 + logn 9)
// accepted on input and stripped from 897-byte public keys.
const fnPublicKeyHeader = 0x09

// PublicKey is a TIP-899 post-quantum public key in its canonical wire form.
// It is immutable and safe for concurrent use.
type PublicKey struct {
	scheme core.PQScheme
	key    []byte
	ml     *mldsa44.PublicKey // unpacked key, set for ML_DSA_44
}

// ParsePublicKey parses a public key for scheme.
//
// ML-DSA-44 keys are 1312 bytes. FN-DSA-512 keys are the bare 896-byte h, or
// the 897-byte reference form with a leading 0x09 header, which is stripped.
// The key bytes are copied.
func ParsePublicKey(scheme core.PQScheme, b []byte) (*PublicKey, error) {
	switch scheme {
	case core.PQScheme_ML_DSA_44:
		if len(b) != MLDSA44PublicKeySize {
			return nil, fmt.Errorf("%w: ML-DSA-44 public key must be %d bytes, got %d",
				ErrInvalidKey, MLDSA44PublicKeySize, len(b))
		}
		var buf [mldsa44.PublicKeySize]byte
		copy(buf[:], b)
		ml := new(mldsa44.PublicKey)
		ml.Unpack(&buf)
		return &PublicKey{scheme: scheme, key: buf[:], ml: ml}, nil
	case core.PQScheme_FN_DSA_512:
		switch {
		case len(b) == FNDSA512PublicKeySize:
		case len(b) == FNDSA512PublicKeySize+1 && b[0] == fnPublicKeyHeader:
			b = b[1:]
		default:
			return nil, fmt.Errorf("%w: FN-DSA-512 public key must be %d bytes (or %d with a 0x09 header), got %d",
				ErrInvalidKey, FNDSA512PublicKeySize, FNDSA512PublicKeySize+1, len(b))
		}
		return &PublicKey{scheme: scheme, key: bytes.Clone(b)}, nil
	default:
		return nil, unsupported(scheme)
	}
}

// Scheme reports the signature scheme of the key.
func (k *PublicKey) Scheme() core.PQScheme {
	if k == nil {
		return core.PQScheme_UNKNOWN_PQ_SCHEME
	}
	return k.scheme
}

// Bytes returns a copy of the canonical wire encoding of the key.
func (k *PublicKey) Bytes() []byte {
	if k == nil {
		return nil
	}
	return bytes.Clone(k.key)
}

// Address returns the TRON address of the key:
// 0x41 || Keccak-256(wire public key)[12:32]. It returns nil for an empty key.
func (k *PublicKey) Address() address.Address {
	if k == nil || len(k.key) == 0 {
		return nil
	}
	return address.BytesToAddress(common.Keccak256(k.key)[12:])
}

// Equal reports whether k and o are the same key of the same scheme.
func (k *PublicKey) Equal(o *PublicKey) bool {
	if k == nil || o == nil {
		return k == o
	}
	return k.scheme == o.scheme && bytes.Equal(k.key, o.key)
}

// Verify reports whether sig is a valid signature by k over the 32-byte
// digest.
//
// A digest of any other length returns ErrInvalidDigest. A signature with the
// wrong length (or, for FN-DSA-512, the wrong header byte) returns
// (false, nil) before any backend is consulted. A well-formed FN-DSA-512
// signature in a build without -tags falcon, or with a liboqs that lacks
// Falcon-512, returns false and an error wrapping ErrFalconUnavailable.
func (k *PublicKey) Verify(digest, sig []byte) (bool, error) {
	if err := checkDigest(digest); err != nil {
		return false, err
	}
	switch k.Scheme() {
	case core.PQScheme_ML_DSA_44:
		if len(sig) != MLDSA44SignatureSize || k.ml == nil {
			return false, nil
		}
		return mldsa44.Verify(k.ml, digest, nil, sig), nil
	case core.PQScheme_FN_DSA_512:
		if !fnSignatureWellFormed(sig) || len(k.key) != FNDSA512PublicKeySize {
			return false, nil
		}
		if err := falcon.availErr(); err != nil {
			return false, err
		}
		return falcon.verify(bytes.Clone(k.key), bytes.Clone(digest), bytes.Clone(sig)), nil
	default:
		return false, unsupported(k.Scheme())
	}
}
