package keys

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
)

// Deprecated: Use mnemonic.FromSeedAndPassphrase instead.
func FromMnemonicSeedAndPassphrase(m, passphrase string, index int) (*btcec.PrivateKey, *btcec.PublicKey) {
	return mnemonic.FromSeedAndPassphrase(m, passphrase, index)
}
