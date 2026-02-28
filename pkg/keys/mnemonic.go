package keys

import (
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/go-bip39"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
)

// FromMnemonicSeedAndPassphrase derive form mnemonic and passphrase at index
func FromMnemonicSeedAndPassphrase(mnemonic, passphrase string, index int) (*btcec.PrivateKey, *btcec.PublicKey) {
	if index < 0 {
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

	return btcec.PrivKeyFromBytes(private[:])
}
