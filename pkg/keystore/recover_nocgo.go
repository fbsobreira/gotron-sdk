//go:build !windows && !cgo
// +build !windows,!cgo

package keystore

import (
	"errors"

	"github.com/btcsuite/btcd/btcec/v2"
	btc_ecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

// SignatureLength indicates the byte length required to carry a signature with recovery id.
const SignatureLength = 64 + 1 // 64 bytes ECDSA signature + 1 byte recovery id

// RecoveryIDOffset points to the byte offset within the signature that contains the recovery id.
const RecoveryIDOffset = 64

// DigestLength sets the signature digest exact length
const DigestLength = 32

func RecoverPubkey(hash []byte, signature []byte) (address.Address, error) {

	if signature[64] >= 27 {
		signature[64] -= 27
	}

	sigPublicKey, err := nocgoRecoverPubkey(hash, signature)
	if err != nil {
		return nil, err
	}

	pubKey, err := UnmarshalPublic(sigPublicKey)
	if err != nil {
		return nil, err
	}

	addr := address.PubkeyToAddress(*pubKey)
	return addr, nil
}

func nocgoRecoverPubkey(hash, sig []byte) ([]byte, error) {
	if len(sig) != SignatureLength {
		return nil, errors.New("invalid signature")
	}
	// Convert to btcec input format with 'recovery id' v at the beginning.
	btcsig := make([]byte, SignatureLength)
	btcsig[0] = sig[RecoveryIDOffset] + 27
	copy(btcsig[1:], sig)

	pub, _, err := btc_ecdsa.RecoverCompact(btcsig, hash)
	//covert pub to []byte
	bytes := (*btcec.PublicKey)(pub).SerializeUncompressed()
	return bytes, err
}
