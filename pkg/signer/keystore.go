package signer

import (
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// keystoreSigner implements Signer using a keystore account.
// The account must be unlocked before signing.
type keystoreSigner struct {
	ks   *keystore.KeyStore
	acct keystore.Account
}

// NewKeystoreSigner creates a Signer backed by a keystore account.
// The account must be unlocked (via ks.Unlock) before Sign is called.
func NewKeystoreSigner(ks *keystore.KeyStore, acct keystore.Account) Signer {
	return &keystoreSigner{ks: ks, acct: acct}
}

// Sign signs the transaction using the keystore's unlocked key.
func (s *keystoreSigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	return s.ks.SignTx(s.acct, tx)
}

// Address returns the TRON address of the keystore account.
func (s *keystoreSigner) Address() address.Address {
	return s.acct.Address
}

// keystorePassphraseSigner implements Signer using a keystore account with
// passphrase-based signing (no prior unlock required).
type keystorePassphraseSigner struct {
	ks         *keystore.KeyStore
	acct       keystore.Account
	passphrase string
}

// NewKeystorePassphraseSigner creates a Signer that decrypts the key on each
// Sign call using the provided passphrase. This is safer than keeping the key
// unlocked but slower.
func NewKeystorePassphraseSigner(ks *keystore.KeyStore, acct keystore.Account, passphrase string) Signer {
	return &keystorePassphraseSigner{ks: ks, acct: acct, passphrase: passphrase}
}

// Sign signs the transaction by decrypting the key with the passphrase.
func (s *keystorePassphraseSigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	return s.ks.SignTxWithPassphrase(s.acct, s.passphrase, tx)
}

// Address returns the TRON address of the keystore account.
func (s *keystorePassphraseSigner) Address() address.Address {
	return s.acct.Address
}
