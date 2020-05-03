package keys

import (
	"fmt"

	secp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/hd"
	"github.com/tyler-smith/go-bip39"
)

// FromMnemonicSeedAndPassphrase derive form mnemonic and passphrase at index
func FromMnemonicSeedAndPassphrase(mnemonic, passphrase string, index int) (*secp256k1.PrivateKey, *secp256k1.PublicKey) {
	seed := bip39.NewSeed(mnemonic, passphrase)
	master, ch := hd.ComputeMastersFromSeed(seed)
	private, _ := hd.DerivePrivateKeyForPath(
		master,
		ch,
		fmt.Sprintf("44'/195'/0'/0/%d", index),
	)

	return secp256k1.PrivKeyFromBytes(secp256k1.S256(), private[:])
}
