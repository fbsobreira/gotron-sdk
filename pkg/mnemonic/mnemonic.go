// Package mnemonic provides BIP39 mnemonic phrase generation and validation.
package mnemonic

import (
	"errors"
	"fmt"

	"github.com/fbsobreira/go-bip39"
)

// ErrInvalidMnemonic is returned when a mnemonic phrase is not valid BIP39.
var ErrInvalidMnemonic = errors.New("invalid mnemonic given")

// Generate returns a new random 24-word BIP39 mnemonic phrase.
func Generate() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", fmt.Errorf("generate entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("generate mnemonic: %w", err)
	}
	return mnemonic, nil
}
