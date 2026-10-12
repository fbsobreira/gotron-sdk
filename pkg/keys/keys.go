// Package keys provides key management utilities for TRON accounts.
package keys

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
)

func checkAndMakeKeyDirIfNeeded() (string, error) {
	userDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	tronCTLDir := path.Join(userDir, ".tronctl", "keystore")
	if _, err := os.Stat(tronCTLDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tronCTLDir, 0700); err != nil {
			return "", fmt.Errorf("create keystore dir: %w", err)
		}
	}
	return tronCTLDir, nil
}

// ListKeys prints all accounts in keystoreDir to stdout. An empty keystoreDir
// means the default ~/.tronctl/keystore, which is created if missing.
//
// Deprecated: use keystore.NewKeyStore(dir, ...).Accounts(), or the account
// and store packages, which return the accounts instead of printing them.
func ListKeys(keystoreDir string) {
	tronCTLDir := keystoreDir
	if tronCTLDir == "" {
		var err error
		tronCTLDir, err = checkAndMakeKeyDirIfNeeded()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return
		}
	}
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	ks := keystore.NewKeyStore(tronCTLDir, scryptN, scryptP)
	defer ks.Close()
	allAccounts := ks.Accounts()
	fmt.Printf("Tron Address:%s File URL:\n", strings.Repeat(" ", address.AddressLengthBase58))
	for _, account := range allAccounts {
		fmt.Printf("%s\t\t %s\n", account.Address, account.URL)
	}
}

// AddNewKey creates a new account in the default keystore directory
// (~/.tronctl/keystore) and prints its address and file. On failure it prints
// the error to stderr and nothing else.
//
// Deprecated: use account.CreateNewLocalAccount, or keystore.NewKeyStore and
// KeyStore.NewAccount, which return errors instead of printing them.
func AddNewKey(password string) {
	tronCTLDir, err := checkAndMakeKeyDirIfNeeded()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	ks := keystore.NewKeyStore(tronCTLDir, scryptN, scryptP)
	defer ks.Close()
	account, err := ks.NewAccount(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "new account error: %v\n", err)
		return
	}
	fmt.Printf("account: %s\n", account.Address)
	fmt.Printf("URL: %s\n", account.URL)
}

// GenerateKey generates a new random secp256k1 private key.
func GenerateKey() (*btcec.PrivateKey, error) {
	return btcec.NewPrivateKey()
}

// GetPrivateKeyFromHex parses a hex-encoded private key string.
func GetPrivateKeyFromHex(privateKeyHex string) (*btcec.PrivateKey, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key hex: %w", err)
	}

	return GetPrivateKeyFromBytes(privateKeyBytes)
}

// ErrInvalidPrivateKey is returned when key material is zero or not below the
// secp256k1 curve order n, so it is not a valid private key.
var ErrInvalidPrivateKey = errors.New("invalid private key: out of range for secp256k1")

// GetPrivateKeyFromBytes creates a private key from raw 32-byte key material.
// The value must be in [1, n-1] for the secp256k1 order n; zero or any value
// >= n returns ErrInvalidPrivateKey rather than being reduced mod n.
func GetPrivateKeyFromBytes(privateKeyBytes []byte) (*btcec.PrivateKey, error) {
	if len(privateKeyBytes) != common.Secp256k1PrivateKeyBytesLength {
		return nil, common.ErrBadKeyLength
	}

	var scalar btcec.ModNScalar
	overflow := scalar.SetByteSlice(privateKeyBytes)
	invalid := overflow || scalar.IsZero()
	scalar.Zero()
	if invalid {
		return nil, ErrInvalidPrivateKey
	}

	// btcec.PrivKeyFromBytes only returns a secret key and public key
	private, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	if private == nil {
		return nil, fmt.Errorf("failed to create private key from bytes")
	}

	return private, nil
}

// ZeroPrivateKey overwrites the private key bytes with zeros.
func ZeroPrivateKey(key *btcec.PrivateKey) {
	if key != nil {
		key.Zero()
	}
}

// ZeroECDSAKey overwrites the backing memory of an ECDSA private key's D value.
// Unlike big.Int.SetInt64(0) which only changes the logical value, this zeros
// the actual backing array to prevent key material from lingering in memory.
func ZeroECDSAKey(key *ecdsa.PrivateKey) {
	if key != nil && key.D != nil {
		b := key.D.Bits()
		for i := range b {
			b[i] = 0
		}
	}
}
