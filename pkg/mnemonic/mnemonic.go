// Package mnemonic provides BIP39 mnemonic phrase generation and validation.
package mnemonic

import (
	"errors"
	"fmt"

	"github.com/fbsobreira/go-bip39"
)

// ErrInvalidMnemonic is returned when a mnemonic phrase is not valid BIP39.
var ErrInvalidMnemonic = errors.New("invalid mnemonic given")

// Generate returns a new random BIP39 mnemonic phrase.
// An optional entropy size in bits can be provided (128, 160, 192, 224, 256).
// Default: 256 (24 words).
func Generate(entropyBits ...int) (string, error) {
	bits := 256
	if len(entropyBits) > 0 {
		bits = entropyBits[0]
	}

	switch bits {
	case 128, 160, 192, 224, 256:
	default:
		return "", fmt.Errorf("invalid entropy size %d: must be 128, 160, 192, 224, or 256", bits)
	}

	entropy, err := bip39.NewEntropy(bits)
	if err != nil {
		return "", fmt.Errorf("generate entropy: %w", err)
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("generate mnemonic: %w", err)
	}
	return mnemonic, nil
}
