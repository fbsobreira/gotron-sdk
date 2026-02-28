package account

import (
	"fmt"
	"path/filepath"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
)

// ExportPrivateKey from account
func ExportPrivateKey(address, passphrase string) error {
	ks := store.FromAddress(address)
	if ks == nil {
		return fmt.Errorf("keystore not found for address %s", address)
	}
	allAccounts := ks.Accounts()
	if len(allAccounts) == 0 {
		return fmt.Errorf("no account found for address %s", address)
	}
	for _, account := range allAccounts {
		_, key, err := ks.GetDecryptedKey(keystore.Account{Address: account.Address}, passphrase)
		if err != nil {
			return err
		}
		fmt.Printf("%064x\n", key.PrivateKey.D)
	}
	return nil
}

// ExportKeystore to file
func ExportKeystore(address, path, passphrase string) (string, error) {
	ks := store.FromAddress(address)
	if ks == nil {
		return "", fmt.Errorf("keystore not found for address %s", address)
	}
	allAccounts := ks.Accounts()
	if len(allAccounts) == 0 {
		return "", fmt.Errorf("no account found for address %s", address)
	}
	dirPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	outFile := filepath.Join(dirPath, fmt.Sprintf("%s.key", address))
	for _, account := range allAccounts {
		keyFile, err := ks.Export(keystore.Account{Address: account.Address}, passphrase, passphrase)
		if err != nil {
			return "", err
		}
		e := writeToFile(outFile, string(keyFile))
		if e != nil {
			return "", e
		}
	}
	return outFile, nil
}
