package account

import (
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
)

// Creation struct for account
type Creation struct {
	Name               string
	Passphrase         string
	Mnemonic           string
	MnemonicPassphrase string
	HdAccountNumber    *uint32
	HdIndexNumber      *uint32
}

// New create new name
func New() string {
	return "New Account"
}

// IsValidPassphrase check if strong
func IsValidPassphrase(pass string) bool {
	// TODO: force strong password
	return true
}

// CreateNewLocalAccount assumes all the inputs are valid, legitmate
func CreateNewLocalAccount(candidate *Creation) error {
	ks := store.FromAccountName(candidate.Name)
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
