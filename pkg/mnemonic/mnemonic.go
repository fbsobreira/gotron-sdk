// Package mnemonic provides BIP39 mnemonic phrase generation and validation.
package mnemonic

import (
	"fmt"

	"github.com/fbsobreira/go-bip39"
)

// ErrInvalidMnemonic is returned when a mnemonic phrase is not valid BIP39.
var ErrInvalidMnemonic = fmt.Errorf("invalid mnemonic given")

// Generate returns a new random 24-word BIP39 mnemonic phrase.
func Generate() string {
	entropy, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropy)
	return mnemonic
}
