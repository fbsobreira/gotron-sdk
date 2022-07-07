package keys

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	secp256k1 "github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
	"github.com/tyler-smith/go-bip39"
)

// FromMnemonicSeedAndPassphrase derive form mnemonic and passphrase at index
func FromMnemonicSeedAndPassphrase(mnemonic, passphrase string, index int) (*secp256k1.PrivateKey, *secp256k1.PublicKey) {
	seed := bip39.NewSeed(mnemonic, passphrase)
	master, ch := hd.ComputeMastersFromSeed(seed, []byte("Bitcoin seed"))
	private, _ := hd.DerivePrivateKeyForPath(
		btcec.S256(),
		master,
		ch,
		fmt.Sprintf("44'/195'/0'/0/%d", index),
	)

	return secp256k1.PrivKeyFromBytes(private[:])
}
