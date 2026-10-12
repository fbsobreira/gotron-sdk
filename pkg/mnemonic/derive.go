package mnemonic

import (
	"fmt"
	"math"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/go-bip39"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
)

// FromSeedAndPassphrase derives a key pair from a BIP39 mnemonic and passphrase
// at the given TRON HD path index (44'/195'/0'/0/{index}).
//
// It does NOT check the mnemonic's checksum or word list: any string is turned
// into a seed, so wallets created from an unchecked mnemonic stay recoverable.
// It returns nil, nil when index is outside [0, 2^31) or derivation fails.
// New code should call Validate first, or use DeriveKey, which validates and
// returns the reason for a failure.
func FromSeedAndPassphrase(mnemonic, passphrase string, index int) (*btcec.PrivateKey, *btcec.PublicKey) {
	sk, pk, err := deriveKey(mnemonic, passphrase, index)
	if err != nil {
		return nil, nil
	}
	return sk, pk
}

// DeriveKey derives a key pair from a BIP39 mnemonic and passphrase at the
// given TRON HD path index (44'/195'/0'/0/{index}).
//
// Unlike FromSeedAndPassphrase it validates the mnemonic (an invalid one
// returns an error matching ErrInvalidMnemonic) and the index, which must be
// in [0, 2^31), and it returns an error instead of nil keys. For a valid
// mnemonic and index both functions derive the same key.
func DeriveKey(mnemonic, passphrase string, index int) (*btcec.PrivateKey, *btcec.PublicKey, error) {
	if err := Validate(mnemonic); err != nil {
		return nil, nil, err
	}
	return deriveKey(mnemonic, passphrase, index)
}

// deriveKey derives the key at 44'/195'/0'/0/{index} without checking the
// mnemonic itself.
func deriveKey(mnemonic, passphrase string, index int) (*btcec.PrivateKey, *btcec.PublicKey, error) {
	if index < 0 || index > math.MaxInt32 {
		return nil, nil, fmt.Errorf("derivation index %d out of range: must be in [0, 2^31)", index)
	}

	seed := bip39.NewSeed(mnemonic, passphrase)
	defer common.ZeroBytes(seed)
	master, ch := hd.ComputeMastersFromSeed(seed, []byte("Bitcoin seed"))
	defer common.ZeroBytes(master[:])
	defer common.ZeroBytes(ch[:])
	private, err := hd.DerivePrivateKeyForPath(
		btcec.S256(),
		master,
		ch,
		fmt.Sprintf("44'/195'/0'/0/%d", index),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("derive key: %w", err)
	}

	sk, pk := btcec.PrivKeyFromBytes(private[:])
	common.ZeroBytes(private[:])
	return sk, pk, nil
}
