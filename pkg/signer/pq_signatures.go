package signer

import (
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// PQSignatureInfo describes one pq_auth_sig entry of a transaction: the
// scheme it claims, and the public key and TRON address that scheme and key
// bytes resolve to. It carries no signature bytes.
type PQSignatureInfo struct {
	// Scheme is the signature scheme the entry declares.
	Scheme core.PQScheme
	// Address is the TRON address of PublicKey.
	Address address.Address
	// PublicKey is the entry's public key, parsed for Scheme.
	PublicKey *pqc.PublicKey
}

// PQSignatures lists who the pq_auth_sig entries of tx claim to be, one
// PQSignatureInfo per entry and in transaction order, so a caller can tell
// which accounts have already signed before adding another signature.
//
// It does NOT verify the signatures: FN-DSA-512 verification needs the falcon
// build (see pqc.ErrFalconUnavailable), and the node verifies every entry when
// the transaction is broadcast. A listed address means only that an entry
// names that key, not that the entry is valid.
//
// A malformed entry (a nil entry, an unsupported scheme or a public key that
// does not parse for its scheme) makes the whole call fail with an error that
// names its index as pq_auth_sig[i]; entries are never skipped, because a
// skipped entry would misreport who has signed. A nil tx is an error; a
// transaction with no pq_auth_sig entries returns an empty list.
func PQSignatures(tx *core.Transaction) ([]PQSignatureInfo, error) {
	if tx == nil {
		return nil, errors.New("signer: nil transaction")
	}
	entries := tx.GetPqAuthSig()
	infos := make([]PQSignatureInfo, 0, len(entries))
	for i, e := range entries {
		if e == nil {
			return nil, fmt.Errorf("pq_auth_sig[%d]: nil entry", i)
		}
		pub, err := pqc.ParsePublicKey(e.GetScheme(), e.GetPublicKey())
		if err != nil {
			return nil, fmt.Errorf("pq_auth_sig[%d]: %w", i, err)
		}
		infos = append(infos, PQSignatureInfo{
			Scheme:    pub.Scheme(),
			Address:   pub.Address(),
			PublicKey: pub,
		})
	}
	return infos, nil
}
