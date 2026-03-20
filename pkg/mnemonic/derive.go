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
func FromSeedAndPassphrase(mnemonic, passphrase string, index int) (*btcec.PrivateKey, *btcec.PublicKey) {
	if index < 0 || index > math.MaxUint32 {
		return nil, nil
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
		return nil, nil
	}

	sk, pk := btcec.PrivKeyFromBytes(private[:])
	common.ZeroBytes(private[:])
	return sk, pk
}
