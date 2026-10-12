package keys

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
)

// FromMnemonicSeedAndPassphrase derives a key pair at the TRON HD path index
// (44'/195'/0'/0/{index}). Like mnemonic.FromSeedAndPassphrase, it does NOT
// check the mnemonic's checksum, and it returns nil, nil on failure.
//
// Deprecated: Use mnemonic.DeriveKey, which validates the mnemonic and returns
// an error, or mnemonic.Validate followed by mnemonic.FromSeedAndPassphrase.
func FromMnemonicSeedAndPassphrase(m, passphrase string, index int) (*btcec.PrivateKey, *btcec.PublicKey) {
	return mnemonic.FromSeedAndPassphrase(m, passphrase, index)
}
