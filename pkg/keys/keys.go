// Package keys provides key management utilities for TRON accounts.
package keys

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"

	homedir "github.com/mitchellh/go-homedir"
)

func checkAndMakeKeyDirIfNeeded() string {
	userDir, _ := homedir.Dir()
	tronCTLDir := path.Join(userDir, ".tronctl", "keystore")
	if _, err := os.Stat(tronCTLDir); os.IsNotExist(err) {
		// Double check with Leo what is right file persmission
		err := os.Mkdir(tronCTLDir, 0700)
		if err != nil {
			fmt.Printf("create keystore dir error: %v\n", err)
			return ""
		}
	}

	return tronCTLDir
}

// ListKeys prints all accounts in the keystore directory to stdout.
func ListKeys(keystoreDir string) {
	tronCTLDir := checkAndMakeKeyDirIfNeeded()
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	ks := keystore.NewKeyStore(tronCTLDir, scryptN, scryptP)
	// keystore.KeyStore
	allAccounts := ks.Accounts()
	fmt.Printf("Tron Address:%s File URL:\n", strings.Repeat(" ", address.AddressLengthBase58))
	for _, account := range allAccounts {
		fmt.Printf("%s\t\t %s\n", account.Address, account.URL)
	}
}

// AddNewKey creates a new account in the default keystore directory.
func AddNewKey(password string) {
	tronCTLDir := checkAndMakeKeyDirIfNeeded()
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	ks := keystore.NewKeyStore(tronCTLDir, scryptN, scryptP)
	account, err := ks.NewAccount(password)
	if err != nil {
		fmt.Printf("new account error: %v\n", err)
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

// GetPrivateKeyFromBytes creates a private key from raw 32-byte key material.
func GetPrivateKeyFromBytes(privateKeyBytes []byte) (*btcec.PrivateKey, error) {
	if len(privateKeyBytes) != common.Secp256k1PrivateKeyBytesLength {
		return nil, common.ErrBadKeyLength
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
