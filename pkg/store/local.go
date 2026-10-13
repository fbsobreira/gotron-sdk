// Package store provides local keystore management for TRON accounts.
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
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
// When the home directory cannot be determined, the Store keeps the relative
// default and every operation that needs the directory reports that error (or,
// where it cannot return one, does nothing); see the methods of Store.
func DefaultStoreInstance() *Store {
	root, err := configRoot()
	if err != nil {
		return NewStore(getDefaultConfigDir())
	}
	return NewStore(root)
}

// configDir must be read under s.mu. These helpers snapshot it for callers.
func (s *Store) getConfigDir() string {
	s.mu.Lock()
	dir := s.configDir
	s.mu.Unlock()
	return dir
}

func configRootFromDir(dir string) (string, error) {
	if filepath.IsAbs(dir) {
		return filepath.Clean(dir), nil
	}
	uDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(uDir, dir), nil
}

// resolveConfigRoot calls configRootFromDir and adds the directory to the
// error. It fails only for a relative dir when the home directory is unknown
// ($HOME unset on Unix, %USERPROFILE% on Windows). It never falls back to the
// current directory: a caller that cannot resolve the root must not touch the
// filesystem.
func resolveConfigRoot(dir string) (string, error) {
	root, err := configRootFromDir(dir)
	if err != nil {
		return "", fmt.Errorf("resolve config directory %q: %w: %w", dir, ErrNoConfigDir, err)
	}
	return root, nil
}

// accountsDir returns the account-keys folder of the Store.
func (s *Store) accountsDir() (string, error) {
	root, err := resolveConfigRoot(s.getConfigDir())
	if err != nil {
		return "", err
	}
	return filepath.Join(root, c.DefaultConfigAccountAliasesDirName), nil
}

// ensureDir creates dir (mode 0700) if it does not exist and prints a failure
// to stdout, as the void InitConfigDir and SetDefaultLocation always have.
func ensureDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			fmt.Printf("create keystore dir error: %v\n", err)
		}
	}
}

// InitConfigDir creates the account keystore directory if it does not exist.
// If the config directory cannot be resolved (a relative directory without a
// home directory) it creates nothing and prints the error.
func (s *Store) InitConfigDir() {
	dir, err := s.accountsDir()
	if err != nil {
		fmt.Printf("create keystore dir error: %v\n", err)
		return
	}
	ensureDir(dir)
}

// LocalAccounts returns the alias names of all locally stored accounts. It
// returns an empty list when the config directory cannot be resolved.
func (s *Store) LocalAccounts() []string {
	accounts := []string{}
	dir, err := s.accountsDir()
	if err != nil {
		return accounts
	}
	files, _ := os.ReadDir(dir)

	for _, node := range files {
		if node.IsDir() {
			accounts = append(accounts, filepath.Base(node.Name()))
		}
	}
	return accounts
}

var (
	describe = fmt.Sprintf("%-24s\t\t%23s\n", "NAME", "ADDRESS")
	// ErrNoConfigDir is wrapped by every error caused by a config folder that
	// cannot be resolved: a relative config dir with no home directory.
	ErrNoConfigDir = errors.New("the keystore folder cannot be resolved (is $HOME set?)")
	// ErrKeystoreNotFound is wrapped by UnlockedKeystore when the address has no
	// ECDSA keystore in the local store.
	ErrKeystoreNotFound = errors.New("account not found in the local keystore")
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
		s.Forget(ks)
	}
}

// DoesNamedAccountExist reports whether an account alias with the given name exists locally.
// It reports false when the config directory cannot be resolved.
func (s *Store) DoesNamedAccountExist(name string) bool {
	return slices.Contains(s.LocalAccounts(), name)
}

// AddressFromAccountName returns the Base58 address for the named account, or an error if not found.
// A name that is not a single path element on this OS is rejected before any
// path is built, so it cannot reach a keystore outside the accounts folder.
func (s *Store) AddressFromAccountName(name string) (string, error) {
	if err := checkLocalAccountName(name); err != nil {
		return "", err
	}
	ks := s.FromAccountName(name)
	if ks == nil {
		_, err := s.accountsDir()
		return "", err
	}
	defer func() {
		ks.Close()
		s.Forget(ks)
	}()
	// FIXME: Assume 1 account per keystore for now
	for _, account := range ks.Accounts() {
		return account.Address.String(), nil
	}
	return "", fmt.Errorf("no accounts found in keystore for %s", name)
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
		s.Forget(ks)
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

// checkLocalAccountName rejects a name that is not a single path element
// inside the accounts folder on this OS: empty, ".", "..", absolute, or
// containing a separator. Unlike ValidateAccountName, which new names must
// pass, it accepts "\" on Unix, where it is an ordinary file name character,
// so accounts created under such names before v0.26.0 still resolve.
func checkLocalAccountName(name string) error {
	if name == "" || name == "." || !filepath.IsLocal(name) || filepath.Base(name) != name {
		return fmt.Errorf("%w %q", ErrInvalidAccountName, name)
	}
	return nil
}

// FromAccountName returns a KeyStore loaded from the named account's directory.
// It does not check name: the caller must validate it with ValidateAccountName
// first, or a name such as "../x" addresses a directory outside the store.
// It returns nil when the config directory cannot be resolved (a relative
// directory without a home directory), so check the result before use.
func (s *Store) FromAccountName(name string) *keystore.KeyStore {
	dir, err := s.accountsDir()
	if err != nil {
		return nil
	}
	p := filepath.Join(dir, name)
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

// DefaultLocation returns the current default keystore directory path. It
// returns "" when the config directory cannot be resolved; do not join the
// empty string with a name and write to it, which would be relative to the
// current directory.
func (s *Store) DefaultLocation() string {
	dir, err := s.accountsDir()
	if err != nil {
		return ""
	}
	return dir
}

// SetDefaultLocation updates the config directory and creates the account-keys
// subdirectory if needed. A relative directory is resolved against the home
// directory. If that cannot be resolved the directory is still recorded, so
// later operations fail the same way instead of using the previous one; nothing
// is created and the error is printed.
func (s *Store) SetDefaultLocation(directory string) {
	s.mu.Lock()
	s.configDir = directory
	s.mu.Unlock()
	dir, err := s.accountsDir()
	if err != nil {
		fmt.Printf("create keystore dir error: %v\n", err)
		return
	}
	ensureDir(dir)
}

// findError reports a failed keystore lookup. It wraps ErrKeystoreNotFound
// only when no key matches, not when several do (keystore.AmbiguousAddrError).
func findError(from string, err error) error {
	if errors.Is(err, keystore.ErrNoMatch) {
		return fmt.Errorf("could not find %s in keystore: %w: %w", from, err, ErrKeystoreNotFound)
	}
	return fmt.Errorf("could not find %s in keystore: %w", from, err)
}

// UnlockedKeystore finds, unlocks, and returns the keystore and account for the given Base58 address.
func (s *Store) UnlockedKeystore(from, passphrase string) (*keystore.KeyStore, *keystore.Account, error) {
	sender, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, nil, fmt.Errorf("address not valid: %s", from)
	}
	if _, err := s.accountsDir(); err != nil {
		return nil, nil, err
	}
	ks := s.FromAddress(from)
	if ks == nil {
		return nil, nil, fmt.Errorf("could not open local keystore for %s: %w", from, ErrKeystoreNotFound)
	}
	account, lookupErr := ks.Find(keystore.Account{Address: sender})
	if lookupErr != nil {
		ks.Close()
		s.Forget(ks)
		return nil, nil, findError(from, lookupErr)
	}
	if unlockError := ks.Unlock(account, passphrase); unlockError != nil {
		ks.Close()
		s.Forget(ks)
		return nil, nil, fmt.Errorf("%w: %w", unlockError, ErrNoUnlockBadPassphrase)
	}
	return ks, &account, nil
}

// --- Backward-compatible package-level functions ---
// These preserve the original API so existing callers continue to work.

// getDefaultConfigDir snapshots c.DefaultConfigDirName under keystoreMu.
func getDefaultConfigDir() string {
	keystoreMu.Lock()
	dir := c.DefaultConfigDirName
	keystoreMu.Unlock()
	return dir
}

// configRoot resolves the default config root; see resolveConfigRoot.
func configRoot() (string, error) {
	return resolveConfigRoot(getDefaultConfigDir())
}

func accountsDir() (string, error) {
	root, err := configRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, c.DefaultConfigAccountAliasesDirName), nil
}

// InitConfigDir creates the account keystore directory if it does not exist.
// If the config directory cannot be resolved (a relative directory without a
// home directory) it creates nothing and prints the error.
func InitConfigDir() {
	dir, err := accountsDir()
	if err != nil {
		fmt.Printf("create keystore dir error: %v\n", err)
		return
	}
	ensureDir(dir)
}

// LocalAccounts returns the alias names of all locally stored accounts. It
// returns an empty list when the config directory cannot be resolved.
func LocalAccounts() []string {
	accounts := []string{}
	dir, err := accountsDir()
	if err != nil {
		return accounts
	}
	files, _ := os.ReadDir(dir)

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
// It reports false when the config directory cannot be resolved.
func DoesNamedAccountExist(name string) bool {
	return slices.Contains(LocalAccounts(), name)
}

// AddressFromAccountName returns the Base58 address for the named account, or an error if not found.
// A name that is not a single path element on this OS is rejected before any
// path is built, so it cannot reach a keystore outside the accounts folder.
func AddressFromAccountName(name string) (string, error) {
	if err := checkLocalAccountName(name); err != nil {
		return "", err
	}
	ks := FromAccountName(name)
	if ks == nil {
		_, err := accountsDir()
		return "", err
	}
	defer ks.Close()
	// FIXME: Assume 1 account per keystore for now
	for _, account := range ks.Accounts() {
		return account.Address.String(), nil
	}
	return "", fmt.Errorf("no accounts found in keystore for %s", name)
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
// It does not check name: the caller must validate it with ValidateAccountName
// first, or a name such as "../x" addresses a directory outside the store.
// It returns nil when the config directory cannot be resolved (a relative
// directory without a home directory), so check the result before use.
func FromAccountName(name string) *keystore.KeyStore {
	dir, err := accountsDir()
	if err != nil {
		return nil
	}
	p := filepath.Join(dir, name)
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

// DefaultLocation returns the current default keystore directory path. It
// returns "" when the config directory cannot be resolved; do not join the
// empty string with a name and write to it, which would be relative to the
// current directory.
func DefaultLocation() string {
	dir, err := accountsDir()
	if err != nil {
		return ""
	}
	return dir
}

// SetDefaultLocation sets the config root, the folder that holds the
// account-keys and pq-keys folders, and creates its account-keys folder if
// needed. A relative directory is resolved against the home directory, not
// the current one (tronctl makes --ks-dir absolute before calling it). If
// that cannot be resolved the directory is still recorded, so later
// operations fail the same way instead of using the previous one; nothing is
// created and the error is printed.
func SetDefaultLocation(directory string) {
	keystoreMu.Lock()
	c.DefaultConfigDirName = directory
	keystoreMu.Unlock()
	dir, err := accountsDir()
	if err != nil {
		fmt.Printf("create keystore dir error: %v\n", err)
		return
	}
	ensureDir(dir)
}

// UnlockedKeystore finds, unlocks, and returns the keystore and account for the given Base58 address.
func UnlockedKeystore(from, passphrase string) (*keystore.KeyStore, *keystore.Account, error) {
	sender, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, nil, fmt.Errorf("address not valid: %s", from)
	}
	if _, err := accountsDir(); err != nil {
		return nil, nil, err
	}
	ks := FromAddress(from)
	if ks == nil {
		return nil, nil, fmt.Errorf("could not open local keystore for %s: %w", from, ErrKeystoreNotFound)
	}
	account, lookupErr := ks.Find(keystore.Account{Address: sender})
	if lookupErr != nil {
		ks.Close()
		return nil, nil, findError(from, lookupErr)
	}
	if unlockError := ks.Unlock(account, passphrase); unlockError != nil {
		ks.Close()
		return nil, nil, fmt.Errorf("%w: %w", unlockError, ErrNoUnlockBadPassphrase)
	}
	return ks, &account, nil
}
