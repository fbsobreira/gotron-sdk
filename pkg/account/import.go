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

// ImportFromPrivateKey imports an ECDSA private key (hex string) into the local keystore under the given name.
func ImportFromPrivateKey(privateKey, name, passphrase string) (string, error) {
	privateKey = strings.TrimPrefix(privateKey, "0x")

	if name == "" {
		base, err := generateName()
		if err != nil {
			return "", fmt.Errorf("generate account name: %w", err)
		}
		name = base + "-imported"
		for store.DoesNamedAccountExist(name) {
			base, err = generateName()
			if err != nil {
				return "", fmt.Errorf("generate account name: %w", err)
			}
			name = base + "-imported"
		}
	} else if store.DoesNamedAccountExist(name) {
		return "", fmt.Errorf("account %s already exists", name)
	}

	// private key from bytes
	sk, err := keys.GetPrivateKeyFromHex(privateKey)
	if err != nil {
		return "", err
	}

	if name == "." || name == ".." || strings.ContainsAny(name, `/\`) || filepath.IsAbs(name) {
		return "", fmt.Errorf("invalid account name %q", name)
	}

	ks := store.FromAccountName(name)
	defer ks.Close()
	defer keys.ZeroPrivateKey(sk)
	_, err = ks.ImportECDSA(sk.ToECDSA(), passphrase)
	return name, err
}

func generateName() (string, error) {
	m, err := mnemonic.Generate()
	if err != nil {
		return "", err
	}
	words := strings.Split(m, " ")
	existingAccounts := mapset.NewSet()
	for _, a := range store.LocalAccounts() {
		existingAccounts.Add(a)
	}

	i := 0
	for {
		if i >= len(words) {
			m, err = mnemonic.Generate()
			if err != nil {
				return "", err
			}
			words = strings.Split(m, " ")
			i = 0
		}
		candidate := words[i]
		if !existingAccounts.Contains(candidate) {
			return candidate, nil
		}
		i++
	}
}

func writeToFile(path string, data string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = io.WriteString(file, data)
	if err != nil {
		return err
	}
	return file.Sync()
}

// ImportKeyStore imports an encrypted keystore JSON file into the local account store.
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
		base, err := generateName()
		if err != nil {
			return "", fmt.Errorf("generate account name: %w", err)
		}
		name = base + "-imported"
		for store.DoesNamedAccountExist(name) {
			base, err = generateName()
			if err != nil {
				return "", fmt.Errorf("generate account name: %w", err)
			}
			name = base + "-imported"
		}
	} else if store.DoesNamedAccountExist(name) {
		return "", fmt.Errorf("account %s already exists", name)
	}

	// Prevent path traversal via account name.
	if name == "." || name == ".." || strings.ContainsAny(name, `/\`) || filepath.IsAbs(name) {
		return "", fmt.Errorf("invalid account name %q", name)
	}

	key, err := keystore.DecryptKey(keyJSON, passphrase)
	if err != nil {
		return "", err
	}

	if existingKs := store.FromAddress(key.Address.String()); existingKs != nil {
		existingKs.Close()
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
