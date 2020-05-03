package mnemonic

import (
	"fmt"

	"github.com/tyler-smith/go-bip39"
)

var (
	// ErrInvalidMnemonic error
	ErrInvalidMnemonic = fmt.Errorf("invalid mnemonic given")
)

// Generate with 24 words deafult
func Generate() string {
	entropy, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropy)
	return mnemonic
}
