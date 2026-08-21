//go:build !windows
// +build !windows

package keystore

import (
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

// RecoverPubkey recovers the TRON address from a message hash and its ECDSA signature.
// The caller's signature slice is never modified; V-byte normalization
// (Ethereum-style v >= 27), if needed, is applied to an internal copy.
func RecoverPubkey(hash []byte, signature []byte) (address.Address, error) {
	if len(signature) != 65 {
		return nil, fmt.Errorf("invalid signature length: %d/65", len(signature))
	}
	// Always copy so callers can re-verify, serialize, or broadcast the original
	// signature without observing a mutated V byte.
	sig := make([]byte, 65)
	copy(sig, signature)
	if sig[64] >= 27 {
		sig[64] -= 27
	}

	sigPublicKey, err := crypto.Ecrecover(hash, sig)
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
