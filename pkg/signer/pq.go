package signer

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "google.golang.org/protobuf/proto"
)

// MaxTotalSignatures is the most signatures a TRON transaction may carry:
// ECDSA signatures plus TIP-899 pq_auth_sig entries together. It mirrors the
// node's TOTAL_SIGN_NUM, a genesis constant that proposals cannot change.
// Every signer in this package refuses to add a signature beyond it.
const MaxTotalSignatures = keystore.MaxTotalSignatures

// ErrTooManySignatures is returned (possibly wrapped) by every signer in this
// package, ECDSA and post-quantum, when the transaction already carries
// MaxTotalSignatures signatures, so the node would reject another. It is
// keystore.ErrTooManySignatures.
var ErrTooManySignatures = keystore.ErrTooManySignatures

// Errors returned (possibly wrapped) by SignTransactionPQ.
//
// Experimental: TIP-899 is a draft and is live only on the Nile testnet.
var (
	// ErrPQAlreadySigned is returned when the transaction already carries a
	// pq_auth_sig entry for the same scheme and public key; the node rejects
	// a key that signs twice.
	ErrPQAlreadySigned = errors.New("signer: post-quantum key has already signed the transaction")
)

// SignTransactionPQ signs tx with a TIP-899 post-quantum key and appends a
// pq_auth_sig entry (scheme, public key, signature). The signed message is
// the raw 32-byte transaction ID, SHA-256(raw_data), the same digest the
// ECDSA path signs. tx.Signature is never modified, so ECDSA and
// post-quantum signatures can be mixed on one transaction.
//
// Before signing it refuses a transaction that already carries
// MaxTotalSignatures signatures (ErrTooManySignatures) or a pq_auth_sig
// entry from the same key (ErrPQAlreadySigned). The new signature is verified
// against the key's public key before it is appended. On any error tx is left
// unchanged. It returns tx itself; it is not safe for concurrent use on the
// same transaction.
//
// Whether the chain has the scheme enabled, and whether the account's
// permission has room for another post-quantum key, are not checked here:
// signing does not ask the node. Use txcore.Send, which does, or call
// txcore.CheckPQSchemes before broadcasting.
//
// Experimental: TIP-899 is a draft and is live only on the Nile testnet.
func SignTransactionPQ(tx *core.Transaction, key pqc.PrivateKey) (*core.Transaction, error) {
	return signTransactionPQ(tx, key, (*pqc.PublicKey).Verify)
}

// pqVerifyFunc checks a fresh signature before it is attached.
type pqVerifyFunc func(pub *pqc.PublicKey, digest, sig []byte) (bool, error)

// signTransactionPQ implements SignTransactionPQ with an injectable verifier,
// so tests can simulate a faulty backend (the closed pqc.PrivateKey interface
// does not allow a fake key).
func signTransactionPQ(tx *core.Transaction, key pqc.PrivateKey, verify pqVerifyFunc) (*core.Transaction, error) {
	if tx == nil {
		return nil, errors.New("signer: nil transaction")
	}
	if tx.GetRawData() == nil {
		return nil, errors.New("signer: transaction has no raw_data")
	}
	if key == nil {
		return nil, errors.New("signer: nil post-quantum key")
	}
	pub := key.Public()
	if pub == nil {
		return nil, fmt.Errorf("signer: %s key has no public key", key.Scheme())
	}
	if err := keystore.CheckSignatureRoom(tx); err != nil {
		return nil, err
	}
	pubBytes := pub.Bytes()
	for _, e := range tx.GetPqAuthSig() {
		if e.GetScheme() == key.Scheme() && bytes.Equal(e.GetPublicKey(), pubBytes) {
			return nil, fmt.Errorf("%w: %s %s", ErrPQAlreadySigned, key.Scheme(), pub.Address())
		}
	}

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return nil, fmt.Errorf("signer: marshal raw_data: %w", err)
	}
	digest := sha256.Sum256(rawData)

	sig, err := key.Sign(digest[:])
	if err != nil {
		return nil, fmt.Errorf("signer: %s sign: %w", key.Scheme(), err)
	}
	ok, err := verify(pub, digest[:], sig)
	if err != nil {
		return nil, fmt.Errorf("signer: %s verify own signature: %w", key.Scheme(), err)
	}
	if !ok {
		return nil, fmt.Errorf("signer: %s signature does not verify against its own public key", key.Scheme())
	}

	tx.PqAuthSig = append(tx.PqAuthSig, &core.PQAuthSig{
		Scheme:    key.Scheme(),
		PublicKey: pubBytes,
		Signature: sig,
	})
	return tx, nil
}

// pqSigner implements Signer with a TIP-899 post-quantum key.
type pqSigner struct {
	key  pqc.PrivateKey
	addr address.Address
}

// NewPQSigner creates a Signer from a TIP-899 post-quantum key. Sign appends
// a pq_auth_sig entry through SignTransactionPQ and leaves the ECDSA
// signatures alone. The returned value also implements txcore.PQSchemeSigner
// (a PQScheme() core.PQScheme method), so txcore.Send can check that the node
// has the scheme enabled before signing.
//
// An FN-DSA-512 key is rejected with an error wrapping
// pqc.ErrFalconUnavailable when this build has no Falcon backend, rather than
// failing on the first Sign. A key that has already been destroyed is
// rejected with an error wrapping pqc.ErrKeyDestroyed. The signer does not
// copy the key: destroying the key later also disables the signer.
//
// Experimental: TIP-899 is a draft and is live only on the Nile testnet.
func NewPQSigner(key pqc.PrivateKey) (Signer, error) {
	if key == nil {
		return nil, errors.New("signer: nil post-quantum key")
	}
	if key.Destroyed() {
		return nil, fmt.Errorf("signer: %s: %w", key.Scheme(), pqc.ErrKeyDestroyed)
	}
	if err := pqc.SchemeAvailable(key.Scheme()); err != nil {
		return nil, fmt.Errorf("signer: %s: %w", key.Scheme(), err)
	}
	addr := key.Public().Address()
	if addr == nil {
		return nil, fmt.Errorf("signer: %s key has no address", key.Scheme())
	}
	return &pqSigner{key: key, addr: addr}, nil
}

// Sign appends a post-quantum signature to the transaction; see
// SignTransactionPQ.
func (s *pqSigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	return SignTransactionPQ(tx, s.key)
}

// Address returns the TRON address derived from the post-quantum public key.
// Returns a copy to prevent callers from mutating the cached address.
func (s *pqSigner) Address() address.Address {
	return append(address.Address(nil), s.addr...)
}

// PQScheme reports the post-quantum signature scheme of the signing key.
//
// Experimental: TIP-899 is a draft and is live only on the Nile testnet.
func (s *pqSigner) PQScheme() core.PQScheme {
	return s.key.Scheme()
}
