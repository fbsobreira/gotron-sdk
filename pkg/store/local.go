// Package store provides local keystore management for TRON accounts.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	homedir "github.com/mitchellh/go-homedir"
)

// Store manages keystores for TRON accounts.
type Store struct {
	configDir     string
	mu            sync.Mutex
	openKeystores map[*keystore.KeyStore]struct{}
	newKeyStore   func(string) *keystore.KeyStore
}

// NewStore creates a Store rooted at the given directory.
func NewStore(configDir string) *Store {
	return &Store{
		configDir:     configDir,
		openKeystores: make(map[*keystore.KeyStore]struct{}),
		newKeyStore:   keystore.ForPath,
	}
}

// DefaultStoreInstance creates a Store using the default ~/.tronctl directory.
func DefaultStoreInstance() *Store {
	uDir, _ := homedir.Dir()
	return &Store{
		configDir:     filepath.Join(uDir, c.DefaultConfigDirName),
		openKeystores: make(map[*keystore.KeyStore]struct{}),
		newKeyStore:   keystore.ForPath,
	}
}

func (s *Store) configRoot() string {
	if filepath.IsAbs(s.configDir) {
		return filepath.Clean(s.configDir)
	}
	uDir, _ := homedir.Dir()
	return filepath.Join(uDir, s.configDir)
}

func (s *Store) configAccountsDir() string {
	return filepath.Join(s.configRoot(), c.DefaultConfigAccountAliasesDirName)
}

// InitConfigDir creates the account keystore directory if it does not exist.
func (s *Store) InitConfigDir() {
	tronCTLDir := s.configAccountsDir()
	if _, err := os.Stat(tronCTLDir); os.IsNotExist(err) {
		err = os.MkdirAll(tronCTLDir, 0700)
		if err != nil {
			fmt.Printf("create keystore dir error: %v\n", err)
		}
	}
}

// LocalAccounts returns the alias names of all locally stored accounts.
func (s *Store) LocalAccounts() []string {
	files, _ := os.ReadDir(s.configAccountsDir())
	accounts := []string{}

	for _, node := range files {
		if node.IsDir() {
			accounts = append(accounts, filepath.Base(node.Name()))
		}
	}
	return accounts
}

var (
	describe = fmt.Sprintf("%-24s\t\t%23s\n", "NAME", "ADDRESS")
	// ErrNoUnlockBadPassphrase is returned when an account cannot be unlocked with the given passphrase.
	ErrNoUnlockBadPassphrase = fmt.Errorf("could not unlock account with passphrase, perhaps need different phrase")
)

// DescribeLocalAccounts prints all account alias names and their addresses to stdout.
func (s *Store) DescribeLocalAccounts() {
	fmt.Println(describe)
	for _, name := range s.LocalAccounts() {
		ks := s.FromAccountName(name)
		allAccounts := ks.Accounts()
		for _, account := range allAccounts {
			fmt.Printf("%-48s\t%s\n", name, account.Address)
		}
		ks.Close()
	}
}

// DoesNamedAccountExist reports whether an account alias with the given name exists locally.
func (s *Store) DoesNamedAccountExist(name string) bool {
	return slices.Contains(s.LocalAccounts(), name)
}

// AddressFromAccountName returns the Base58 address for the named account, or an error if not found.
func (s *Store) AddressFromAccountName(name string) (string, error) {
	ks := s.FromAccountName(name)
	defer ks.Close()
	// FIXME: Assume 1 account per keystore for now
	for _, account := range ks.Accounts() {
		return account.Address.String(), nil
	}
	return "", fmt.Errorf("keystore not found")
}

// FromAddress will return nil if the Base58 string is not found in the imported accounts.
// Non-matching keystores are closed to prevent goroutine leaks.
func (s *Store) FromAddress(addr string) *keystore.KeyStore {
	for _, name := range s.LocalAccounts() {
		ks := s.FromAccountName(name)
		allAccounts := ks.Accounts()
		found := false
		for _, account := range allAccounts {
			if addr == account.Address.String() {
				found = true
				break
			}
		}
		if found {
			return ks
		}
		ks.Close()
	}
	return nil
}

// SetKeystoreFactory replaces the function used to create keystores.
// Pass keystore.ForPathLight in tests for faster key derivation.
func (s *Store) SetKeystoreFactory(fn func(string) *keystore.KeyStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.newKeyStore = fn
}

// FromAccountName returns a KeyStore loaded from the named account's directory.
func (s *Store) FromAccountName(name string) *keystore.KeyStore {
	p := filepath.Join(s.configAccountsDir(), name)
	s.mu.Lock()
	ks := s.newKeyStore(p)
	s.openKeystores[ks] = struct{}{}
	s.mu.Unlock()
	return ks
}

// Forget removes a keystore from the tracked set. Call this after closing a
// keystore obtained via FromAccountName to allow garbage collection.
func (s *Store) Forget(ks *keystore.KeyStore) {
	s.mu.Lock()
	delete(s.openKeystores, ks)
	s.mu.Unlock()
}

// CloseAll closes all tracked keystores and resets the factory to production
// defaults. This is a safety net for test cleanup.
func (s *Store) CloseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ks := range s.openKeystores {
		ks.Close()
	}
	s.openKeystores = make(map[*keystore.KeyStore]struct{})
	s.newKeyStore = keystore.ForPath
}

// DefaultLocation returns the current default keystore directory path.
func (s *Store) DefaultLocation() string {
	return s.configAccountsDir()
}

// SetDefaultLocation updates the config directory and creates the account-keys
// subdirectory if needed.
func (s *Store) SetDefaultLocation(directory string) {
	s.mu.Lock()
	s.configDir = directory
	tronCTLDir := s.configAccountsDir()
	s.mu.Unlock()
	if _, err := os.Stat(tronCTLDir); os.IsNotExist(err) {
		err = os.MkdirAll(tronCTLDir, 0700)
		if err != nil {
			fmt.Printf("create keystore dir error: %v\n", err)
		}
	}
}

// UnlockedKeystore finds, unlocks, and returns the keystore and account for the given Base58 address.
func (s *Store) UnlockedKeystore(from, passphrase string) (*keystore.KeyStore, *keystore.Account, error) {
	sender, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, nil, fmt.Errorf("address not valid: %s", from)
	}
	ks := s.FromAddress(from)
	if ks == nil {
		return nil, nil, fmt.Errorf("could not open local keystore for %s", from)
	}
	account, lookupErr := ks.Find(keystore.Account{Address: sender})
	if lookupErr != nil {
		ks.Close()
		return nil, nil, fmt.Errorf("could not find %s in keystore", from)
	}
	if unlockError := ks.Unlock(account, passphrase); unlockError != nil {
		ks.Close()
		return nil, nil, fmt.Errorf("%s: %w", unlockError.Error(), ErrNoUnlockBadPassphrase)
	}
	return ks, &account, nil
}

// --- Backward-compatible package-level functions ---
// These preserve the original API so existing callers continue to work.

func configRoot() string {
	if filepath.IsAbs(c.DefaultConfigDirName) {
		return filepath.Clean(c.DefaultConfigDirName)
	}
	uDir, _ := homedir.Dir()
	return filepath.Join(uDir, c.DefaultConfigDirName)
}

func configAccountsDir() string {
	return filepath.Join(configRoot(), c.DefaultConfigAccountAliasesDirName)
}

// InitConfigDir creates the account keystore directory if it does not exist.
func InitConfigDir() {
	tronCTLDir := configAccountsDir()
	if _, err := os.Stat(tronCTLDir); os.IsNotExist(err) {
		err = os.MkdirAll(tronCTLDir, 0700)
		if err != nil {
			fmt.Printf("create keystore dir error: %v\n", err)
		}
	}
}

// LocalAccounts returns the alias names of all locally stored accounts.
func LocalAccounts() []string {
	files, _ := os.ReadDir(configAccountsDir())
	accounts := []string{}

	for _, node := range files {
		if node.IsDir() {
			accounts = append(accounts, filepath.Base(node.Name()))
		}
	}
	return accounts
}

// DescribeLocalAccounts prints all account alias names and their addresses to stdout.
func DescribeLocalAccounts() {
	fmt.Println(describe)
	for _, name := range LocalAccounts() {
		ks := FromAccountName(name)
		allAccounts := ks.Accounts()
		for _, account := range allAccounts {
			fmt.Printf("%-48s\t%s\n", name, account.Address)
		}
		ks.Close()
	}
}

// DoesNamedAccountExist reports whether an account alias with the given name exists locally.
func DoesNamedAccountExist(name string) bool {
	return slices.Contains(LocalAccounts(), name)
}

// AddressFromAccountName returns the Base58 address for the named account, or an error if not found.
func AddressFromAccountName(name string) (string, error) {
	ks := FromAccountName(name)
	defer ks.Close()
	// FIXME: Assume 1 account per keystore for now
	for _, account := range ks.Accounts() {
		return account.Address.String(), nil
	}
	return "", fmt.Errorf("keystore not found")
}

// FromAddress will return nil if the Base58 string is not found in the imported accounts.
// Non-matching keystores are closed to prevent goroutine leaks.
func FromAddress(addr string) *keystore.KeyStore {
	for _, name := range LocalAccounts() {
		ks := FromAccountName(name)
		allAccounts := ks.Accounts()
		found := false
		for _, account := range allAccounts {
			if addr == account.Address.String() {
				found = true
				break
			}
		}
		if found {
			return ks
		}
		ks.Close()
	}
	return nil
}

var (
	keystoreMu    sync.Mutex
	openKeystores []*keystore.KeyStore
	newKeyStore   = keystore.ForPath
)

// SetKeystoreFactory replaces the function used to create keystores.
// Pass keystore.ForPathLight in tests for faster key derivation.
func SetKeystoreFactory(fn func(string) *keystore.KeyStore) {
	keystoreMu.Lock()
	defer keystoreMu.Unlock()
	newKeyStore = fn
}

// FromAccountName returns a KeyStore loaded from the named account's directory.
func FromAccountName(name string) *keystore.KeyStore {
	p := filepath.Join(configAccountsDir(), name)
	keystoreMu.Lock()
	ks := newKeyStore(p)
	openKeystores = append(openKeystores, ks)
	keystoreMu.Unlock()
	return ks
}

// CloseAll closes all tracked keystores and resets the factory to production
// defaults. This is a safety net for test cleanup.
func CloseAll() {
	keystoreMu.Lock()
	defer keystoreMu.Unlock()
	for _, ks := range openKeystores {
		ks.Close()
	}
	openKeystores = nil
	newKeyStore = keystore.ForPath
}

// DefaultLocation returns the current default keystore directory path.
func DefaultLocation() string {
	return configAccountsDir()
}

// SetDefaultLocation updates the default keystore directory and creates it if needed.
func SetDefaultLocation(directory string) {
	keystoreMu.Lock()
	c.DefaultConfigDirName = directory
	keystoreMu.Unlock()
	tronCTLDir := configAccountsDir()
	if _, err := os.Stat(tronCTLDir); os.IsNotExist(err) {
		err = os.MkdirAll(tronCTLDir, 0700)
		if err != nil {
			fmt.Printf("create keystore dir error: %v\n", err)
		}
	}
}

// UnlockedKeystore finds, unlocks, and returns the keystore and account for the given Base58 address.
func UnlockedKeystore(from, passphrase string) (*keystore.KeyStore, *keystore.Account, error) {
	sender, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, nil, fmt.Errorf("address not valid: %s", from)
	}
	ks := FromAddress(from)
	if ks == nil {
		return nil, nil, fmt.Errorf("could not open local keystore for %s", from)
	}
	account, lookupErr := ks.Find(keystore.Account{Address: sender})
	if lookupErr != nil {
		ks.Close()
		return nil, nil, fmt.Errorf("could not find %s in keystore", from)
	}
	if unlockError := ks.Unlock(account, passphrase); unlockError != nil {
		ks.Close()
		return nil, nil, fmt.Errorf("%s: %w", unlockError.Error(), ErrNoUnlockBadPassphrase)
	}
	return ks, &account, nil
}
