//go:build falcon

package pqc

// FN-DSA-512 backend on liboqs (https://github.com/open-quantum-safe/liboqs),
// linked through cgo. It uses the "Falcon-512" algorithm: compressed,
// variable-length signatures with the 0x39 header, which is what TRON nodes
// (BouncyCastle 1.84) verify. "Falcon-padded-512" is a different wire format
// and must never be used here.

/*
#cgo pkg-config: liboqs
#include <stdlib.h>
#include <oqs/oqs.h>
*/
import "C"

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

const (
	// oqsFalconAlg is the liboqs algorithm name (OQS_SIG_alg_falcon_512).
	oqsFalconAlg = "Falcon-512"
	// fnMaxSignAttempts bounds how many times sign resamples a signature
	// that falls outside the TRON wire bounds.
	fnMaxSignAttempts = 16
)

// falcon is the FN-DSA-512 backend: liboqs in builds with the "falcon" tag.
var falcon falconBackend = &liboqsFalcon{}

// oqsEmpty backs zero-length messages so that liboqs is never handed a NULL
// message pointer.
var oqsEmpty [1]byte

// liboqsFalcon implements falconBackend with one OQS_SIG object, created on
// first use and kept for the life of the process. The object is read-only
// after creation and liboqs Falcon keeps no per-call state in it, so sign and
// verify are safe for concurrent use.
type liboqsFalcon struct {
	once sync.Once
	sig  *C.OQS_SIG
	err  error
}

var _ falconBackend = (*liboqsFalcon)(nil)

// init creates the shared OQS_SIG object once and records why it could not
// be, if so.
func (f *liboqsFalcon) init() error {
	f.once.Do(func() {
		C.OQS_init()
		name := C.CString(oqsFalconAlg)
		defer C.free(unsafe.Pointer(name))
		version := C.GoString(C.OQS_version())
		if C.OQS_SIG_alg_is_enabled(name) != 1 {
			f.err = fmt.Errorf("%w: liboqs %s was built without %s", ErrFalconUnavailable, version, oqsFalconAlg)
			return
		}
		s := C.OQS_SIG_new(name)
		if s == nil {
			f.err = fmt.Errorf("%w: liboqs %s could not create %s", ErrFalconUnavailable, version, oqsFalconAlg)
			return
		}
		if s.length_public_key != FNDSA512PublicKeySize+1 ||
			s.length_secret_key != FNDSA512PrivateKeySize+1 ||
			s.length_signature < FNDSA512SignatureMaxSize {
			f.err = fmt.Errorf("%w: liboqs %s %s has unexpected sizes (public key %d, secret key %d, signature %d; want %d, %d, >= %d)",
				ErrFalconUnavailable, version, oqsFalconAlg,
				uint64(s.length_public_key), uint64(s.length_secret_key), uint64(s.length_signature),
				FNDSA512PublicKeySize+1, FNDSA512PrivateKeySize+1, FNDSA512SignatureMaxSize)
			C.OQS_SIG_free(s)
			return
		}
		f.sig = s
	})
	return f.err
}

// availErr returns nil, or why liboqs cannot provide Falcon-512 (wrapping
// ErrFalconUnavailable).
func (f *liboqsFalcon) availErr() error { return f.init() }

// generate returns a fresh key pair in TRON canonical form: the 1280-byte
// f||g||F and the bare 896-byte h, with the liboqs 0x59 and 0x09 headers
// checked and stripped.
func (f *liboqsFalcon) generate() ([]byte, []byte, error) {
	if err := f.init(); err != nil {
		return nil, nil, err
	}
	const skLen = FNDSA512PrivateKeySize + 1
	pk := make([]byte, FNDSA512PublicKeySize+1)
	skC := C.OQS_MEM_malloc(skLen)
	if skC == nil {
		return nil, nil, errors.New("liboqs: out of memory")
	}
	defer C.OQS_MEM_secure_free(skC, skLen)

	if C.OQS_SIG_keypair(f.sig, (*C.uint8_t)(unsafe.Pointer(&pk[0])), (*C.uint8_t)(skC)) != C.OQS_SUCCESS {
		return nil, nil, errors.New("liboqs: Falcon-512 key generation failed")
	}
	skRef := unsafe.Slice((*byte)(skC), skLen)
	if pk[0] != fnPublicKeyHeader || skRef[0] != fnPrivateKeyHeader {
		return nil, nil, fmt.Errorf("liboqs: Falcon-512 key headers 0x%02x/0x%02x, want 0x%02x/0x%02x",
			pk[0], skRef[0], fnPublicKeyHeader, fnPrivateKeyHeader)
	}
	sk := make([]byte, FNDSA512PrivateKeySize)
	copy(sk, skRef[1:])
	return sk, bytes.Clone(pk[1:]), nil
}

// sign signs digest (any length; Sign enforces 32 bytes) with the 1280-byte
// canonical sk. The key is copied, with its 0x59 header, into C memory that
// is wiped and freed before returning, and sk itself is cleared. Signatures
// outside the TRON wire bounds are resampled up to fnMaxSignAttempts times.
func (f *liboqsFalcon) sign(sk, digest []byte) ([]byte, error) {
	defer clear(sk)
	if err := f.init(); err != nil {
		return nil, err
	}
	if len(sk) != FNDSA512PrivateKeySize {
		return nil, fmt.Errorf("%w: FN-DSA-512 private key must be %d bytes, got %d",
			ErrInvalidKey, FNDSA512PrivateKeySize, len(sk))
	}
	const skLen = FNDSA512PrivateKeySize + 1
	skC := C.OQS_MEM_malloc(skLen)
	if skC == nil {
		return nil, errors.New("liboqs: out of memory")
	}
	defer C.OQS_MEM_secure_free(skC, skLen)
	skRef := unsafe.Slice((*byte)(skC), skLen)
	skRef[0] = fnPrivateKeyHeader
	copy(skRef[1:], sk)

	msg, msgLen := cMessage(digest)
	buf := make([]byte, int(f.sig.length_signature))
	return signUntilWellFormed(func() ([]byte, error) {
		var n C.size_t
		rc := C.OQS_SIG_sign(f.sig, (*C.uint8_t)(unsafe.Pointer(&buf[0])), &n, msg, msgLen, (*C.uint8_t)(skC))
		if rc != C.OQS_SUCCESS {
			return nil, errors.New("liboqs: Falcon-512 signing failed")
		}
		if uint64(n) > uint64(len(buf)) {
			return nil, fmt.Errorf("liboqs: Falcon-512 reported a %d-byte signature for a %d-byte buffer", uint64(n), len(buf))
		}
		return bytes.Clone(buf[:n]), nil
	})
}

// verify reports whether sig is a valid signature over digest (any length)
// by the bare 896-byte pk. Malformed input of any length yields false.
func (f *liboqsFalcon) verify(pk, digest, sig []byte) bool {
	if f.init() != nil || len(pk) != FNDSA512PublicKeySize || len(sig) == 0 {
		return false
	}
	ref := make([]byte, 0, FNDSA512PublicKeySize+1)
	ref = append(ref, fnPublicKeyHeader)
	ref = append(ref, pk...)
	msg, msgLen := cMessage(digest)
	return C.OQS_SIG_verify(f.sig, msg, msgLen,
		(*C.uint8_t)(unsafe.Pointer(&sig[0])), C.size_t(len(sig)),
		(*C.uint8_t)(unsafe.Pointer(&ref[0]))) == C.OQS_SUCCESS
}

// cMessage returns a liboqs message pointer and length for m, using a
// non-NULL pointer for an empty message.
func cMessage(m []byte) (*C.uint8_t, C.size_t) {
	if len(m) == 0 {
		return (*C.uint8_t)(unsafe.Pointer(&oqsEmpty[0])), 0
	}
	return (*C.uint8_t)(unsafe.Pointer(&m[0])), C.size_t(len(m))
}

// signUntilWellFormed calls try until it returns a signature with the TRON
// FN-DSA-512 wire shape (0x39 header, 617..667 bytes), at most
// fnMaxSignAttempts times. An error from try is returned at once.
func signUntilWellFormed(try func() ([]byte, error)) ([]byte, error) {
	for range fnMaxSignAttempts {
		sig, err := try()
		if err != nil {
			return nil, err
		}
		if fnSignatureWellFormed(sig) {
			return sig, nil
		}
	}
	return nil, fmt.Errorf("pqc: FN-DSA-512 signature outside %d..%d bytes after %d attempts",
		FNDSA512SignatureMinSize, FNDSA512SignatureMaxSize, fnMaxSignAttempts)
}
