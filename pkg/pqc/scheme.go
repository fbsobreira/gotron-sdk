package pqc

import (
	"fmt"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// ParseScheme parses a TIP-899 scheme name typed by a person, for example a
// command-line flag. It accepts "fn-dsa-512" and "ml-dsa-44" in any case,
// with "-" or "_" as separators and surrounding spaces ignored, so the
// protobuf names "FN_DSA_512" and "ML_DSA_44" work too.
//
// Anything else, including "UNKNOWN_PQ_SCHEME", returns
// core.PQScheme_UNKNOWN_PQ_SCHEME and an error wrapping ErrUnsupportedScheme.
//
// Keystore and Toolkit files use a stricter parser: only the exact protobuf
// names are valid there.
func ParseScheme(s string) (core.PQScheme, error) {
	switch strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(s)), "-", "_") {
	case core.PQScheme_FN_DSA_512.String():
		return core.PQScheme_FN_DSA_512, nil
	case core.PQScheme_ML_DSA_44.String():
		return core.PQScheme_ML_DSA_44, nil
	}
	return core.PQScheme_UNKNOWN_PQ_SCHEME, unsupportedName(s)
}

// unsupportedName returns an ErrUnsupportedScheme error that quotes name,
// cut to 32 bytes so a long input cannot flood a message.
func unsupportedName(name string) error {
	if len(name) > 32 {
		name = name[:32] + "..."
	}
	return fmt.Errorf("%w: %q", ErrUnsupportedScheme, name)
}

// MaxAuthSigSize returns the most bytes one more pq_auth_sig entry of scheme
// adds to a serialized core.Transaction: the field tag, the length prefix and
// the encoded core.PQAuthSig holding the scheme, the public key and a
// signature of the largest size the scheme produces (FN-DSA-512 signatures
// vary in length; ML-DSA-44 signatures are fixed).
//
// It is the size to reserve for a signature that is not yet attached, for
// example when estimating bandwidth before signing. It returns an error
// wrapping ErrUnsupportedScheme for any other scheme.
func MaxAuthSigSize(scheme core.PQScheme) (int, error) {
	var pubLen, sigLen int
	switch scheme {
	case core.PQScheme_FN_DSA_512:
		pubLen, sigLen = FNDSA512PublicKeySize, FNDSA512SignatureMaxSize
	case core.PQScheme_ML_DSA_44:
		pubLen, sigLen = MLDSA44PublicKeySize, MLDSA44SignatureSize
	default:
		return 0, unsupported(scheme)
	}
	// Measure with the real message instead of adding encoding overheads by
	// hand: the size of a transaction with the entry minus one without it.
	// Only lengths matter to the encoding, so the bytes are left zero.
	empty := &core.Transaction{}
	withSig := &core.Transaction{PqAuthSig: []*core.PQAuthSig{{
		Scheme:    scheme,
		PublicKey: make([]byte, pubLen),
		Signature: make([]byte, sigLen),
	}}}
	return proto.Size(withSig) - proto.Size(empty), nil
}
