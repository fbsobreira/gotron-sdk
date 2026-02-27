package account

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	mapset "github.com/deckarep/golang-set"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
)

// ImportFromPrivateKey allows import of an ECDSA private key
func ImportFromPrivateKey(privateKey, name, passphrase string) (string, error) {
	privateKey = strings.TrimPrefix(privateKey, "0x")

	if name == "" {
		name = generateName() + "-imported"
		for store.DoesNamedAccountExist(name) {
			name = generateName() + "-imported"
		}
	} else if store.DoesNamedAccountExist(name) {
		return "", fmt.Errorf("account %s already exists", name)
	}

	// private key from bytes
	sk, err := keys.GetPrivateKeyFromHex(privateKey)
	if err != nil {
		return "", err
	}

	ks := store.FromAccountName(name)
	_, err = ks.ImportECDSA(sk.ToECDSA(), passphrase)
	return name, err
}

func generateName() string {
	words := strings.Split(mnemonic.Generate(), " ")
	existingAccounts := mapset.NewSet()
	for a := range store.LocalAccounts() {
		existingAccounts.Add(a)
	}

	i := 0
	for {
		if i >= len(words) {
			words = strings.Split(mnemonic.Generate(), " ")
			i = 0
		}
		candidate := words[i]
		if !existingAccounts.Contains(candidate) {
			return candidate
		}
		i++
	}
}

func writeToFile(path string, data string) error {
	currDir, _ := os.Getwd()
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}
	err = os.Chdir(filepath.Dir(path))
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Base(path))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		return err
	}
	err = os.Chdir(currDir)
	if err != nil {
		return err
	}
	return file.Sync()
}

// ImportKeyStore imports a keystore along with a password
func ImportKeyStore(keyPath, name, passphrase string) (string, error) {
	keyPath, err := filepath.Abs(keyPath)
	if err != nil {
		return "", err
	}
	keyJSON, readError := os.ReadFile(keyPath)
	if readError != nil {
		return "", readError
	}
	if name == "" {
		name = generateName() + "-imported"
		for store.DoesNamedAccountExist(name) {
			name = generateName() + "-imported"
		}
	} else if store.DoesNamedAccountExist(name) {
		return "", fmt.Errorf("account %s already exists", name)
	}
	key, err := keystore.DecryptKey(keyJSON, passphrase)
	if err != nil {
		return "", err
	}

	hasAddress := store.FromAddress(key.Address.String()) != nil
	if hasAddress {
		return "", fmt.Errorf("address %s already exists in keystore", key.Address.String())
	}
	// create home dir if it doesn't exist
	store.InitConfigDir()
	newPath := filepath.Join(store.DefaultLocation(), name, filepath.Base(keyPath))
	err = writeToFile(newPath, string(keyJSON))
	if err != nil {
		return "", err
	}
	return name, nil
}
