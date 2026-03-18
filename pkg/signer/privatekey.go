package signer

import (
	"crypto/ecdsa"
	"crypto/sha256"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "google.golang.org/protobuf/proto"
)

// privateKeySigner implements Signer using an ECDSA private key.
type privateKeySigner struct {
	key  *ecdsa.PrivateKey
	addr address.Address
}

// NewPrivateKeySigner creates a Signer from an ECDSA private key.
func NewPrivateKeySigner(key *ecdsa.PrivateKey) (Signer, error) {
	// Re-derive through go-ethereum to ensure the correct secp256k1 curve
	// instance is used (required for non-CGO platforms).
	canonical, err := crypto.ToECDSA(crypto.FromECDSA(key))
	if err != nil {
		return nil, err
	}
	return &privateKeySigner{
		key:  canonical,
		addr: address.PubkeyToAddress(canonical.PublicKey),
	}, nil
}

// NewPrivateKeySignerFromBTCEC creates a Signer from a btcec private key.
func NewPrivateKeySignerFromBTCEC(key *btcec.PrivateKey) (Signer, error) {
	return NewPrivateKeySigner(key.ToECDSA())
}

// Sign appends a signature to the transaction.
func (s *privateKeySigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return nil, err
	}

	h := sha256.Sum256(rawData)

	sig, err := crypto.Sign(h[:], s.key)
	if err != nil {
		return nil, err
	}
	tx.Signature = append(tx.Signature, sig)
	return tx, nil
}

// Address returns the TRON address derived from the signing key.
func (s *privateKeySigner) Address() address.Address {
	return s.addr
}
