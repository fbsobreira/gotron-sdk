// Package account provides TRON account creation, import, and export operations.
package account

import (
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
)

// Creation holds the parameters needed to create a new local TRON account.
type Creation struct {
	Name               string
	Passphrase         string
	Mnemonic           string
	MnemonicPassphrase string
	HdAccountNumber    *uint32
	HdIndexNumber      *uint32
}

// New returns the default name for a new account.
func New() string {
	return "New Account"
}

// IsValidPassphrase is a placeholder that currently always returns true.
// TODO: implement actual strength validation.
func IsValidPassphrase(pass string) bool {
	// TODO: force strong password
	return true
}

// CreateNewLocalAccount creates a new account in the local keystore from the given Creation params.
func CreateNewLocalAccount(candidate *Creation) error {
	ks := store.FromAccountName(candidate.Name)
	defer ks.Close()
	if candidate.Mnemonic == "" {
		candidate.Mnemonic = mnemonic.Generate()
	}
	// Hardcoded index of 0 for brandnew account.
	private, _ := keys.FromMnemonicSeedAndPassphrase(candidate.Mnemonic, candidate.MnemonicPassphrase, 0)
	_, err := ks.ImportECDSA(private.ToECDSA(), candidate.Passphrase)
	if err != nil {
		return err
	}
	return nil
}
