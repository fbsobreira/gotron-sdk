package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// TIP-899 post-quantum key store.
//
// Post-quantum keystores live in <configRoot>/pq-keys/<name>/<file>.json,
// next to the ECDSA keystores in <configRoot>/account-keys/<name>/. Both
// folders share the config root, so SetDefaultLocation moves them together,
// and an account name is unique across both (see NameExists). Each account
// folder holds exactly one keystore file in the wallet-cli format written by
// pqc.EncryptKeystore.

// maxPQKeystoreFileSize bounds how much of a keystore file is read. A
// TIP-899 keystore is a few KiB; the cap keeps a stray large file in pq-keys
// from being read into memory.
const maxPQKeystoreFileSize = 64 << 10

var (
	// ErrPQAccountNotFound is returned when no post-quantum account matches
	// the requested name or address.
	//
	// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
	ErrPQAccountNotFound = errors.New("post-quantum account not found")
	// ErrAccountNameExists is returned when an account name is already used
	// in account-keys or pq-keys.
	//
	// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
	ErrAccountNameExists = errors.New("account name already exists")
	// ErrPQAddressExists is returned when a post-quantum account with the
	// same address is already stored.
	//
	// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
	ErrPQAddressExists = errors.New("post-quantum address already exists")
	// ErrInvalidAccountName is returned for an account name that is empty,
	// contains a path separator or NUL, is "." or "..", or is absolute.
	//
	// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
	ErrInvalidAccountName = errors.New("invalid account name")
	// ErrInvalidPQKeystore is returned when a post-quantum account folder
	// exists but does not hold exactly one readable keystore with a supported
	// scheme and a valid address, or when pq-keys/<name> is not a real folder
	// (a plain file or a symlink, which is never followed or removed).
	//
	// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
	ErrInvalidPQKeystore = errors.New("invalid post-quantum keystore")
)

// pqWriteMu serializes the uniqueness re-checks and the write of new
// post-quantum accounts, and removals, within this process.
//
// Name uniqueness holds across processes too: the account folder is created
// with os.Mkdir, which fails if another process created the same name in
// between. Address uniqueness is enforced only within one process: two
// processes storing the same key under different names at the same time can
// both succeed.
var pqWriteMu sync.Mutex

// PQAccount describes a stored post-quantum keystore. All fields come from
// the clear part of the keystore file; nothing is decrypted to fill them.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
type PQAccount struct {
	// Name is the account name, which is also its folder under pq-keys.
	Name string
	// Address is the Base58Check address ("T...") stored in the keystore.
	Address string
	// Scheme is the TIP-899 signature scheme of the key.
	Scheme core.PQScheme
	// Path is the absolute path of the keystore file.
	Path string
}

// ValidateAccountName reports whether name is a valid account name: a single,
// plain path element, so that a name can never address a file outside its
// folder. It returns an error wrapping ErrInvalidAccountName for a name that
// is empty, contains a path separator ("/" or "\") or NUL, is "." or "..",
// or is absolute. The same rule applies to ECDSA and post-quantum accounts.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func ValidateAccountName(name string) error {
	if name == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, "/\\\x00") ||
		filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return fmt.Errorf("%w %q", ErrInvalidAccountName, name)
	}
	return nil
}

func pqKeysDirIn(root string) string {
	return filepath.Join(root, c.DefaultConfigPQKeysDirName)
}

// root resolves the config root of the Store. It fails only for a relative
// config directory when the home directory is unknown; see resolveConfigRoot.
func (s *Store) root() (string, error) {
	return resolveConfigRoot(s.getConfigDir())
}

// nameExistsIn reports whether name is taken in account-keys or pq-keys
// under root. An invalid name is never taken; validate it first. A stat error other than "not
// exist" counts as taken, since the name cannot be shown to be free.
func nameExistsIn(root, name string) bool {
	if ValidateAccountName(name) != nil {
		return false
	}
	for _, dir := range []string{
		filepath.Join(root, c.DefaultConfigAccountAliasesDirName),
		pqKeysDirIn(root),
	} {
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			return true
		}
	}
	return false
}

// readLimited reads at most maxPQKeystoreFileSize bytes of path and fails if
// the file is larger.
func readLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxPQKeystoreFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPQKeystoreFileSize {
		return nil, fmt.Errorf("%s is larger than %d bytes", path, maxPQKeystoreFileSize)
	}
	return data, nil
}

// parsePQClearFields returns the clear scheme and address of a keystore. It
// does not decrypt anything.
func parsePQClearFields(data []byte) (core.PQScheme, string, error) {
	if len(data) > maxPQKeystoreFileSize {
		return 0, "", fmt.Errorf("%w: larger than %d bytes", ErrInvalidPQKeystore, maxPQKeystoreFileSize)
	}
	var clearFields struct {
		Address string `json:"address"`
		Scheme  string `json:"scheme"`
	}
	if err := json.Unmarshal(data, &clearFields); err != nil {
		return 0, "", fmt.Errorf("%w: %v", ErrInvalidPQKeystore, err)
	}
	var scheme core.PQScheme
	switch clearFields.Scheme {
	case core.PQScheme_ML_DSA_44.String():
		scheme = core.PQScheme_ML_DSA_44
	case core.PQScheme_FN_DSA_512.String():
		scheme = core.PQScheme_FN_DSA_512
	default:
		return 0, "", fmt.Errorf("%w: unsupported scheme %.32q", ErrInvalidPQKeystore, clearFields.Scheme)
	}
	a, err := address.Base58ToAddress(clearFields.Address)
	if err != nil || len(a) != address.AddressLength || a[0] != address.TronBytePrefix ||
		a.String() != clearFields.Address {
		return 0, "", fmt.Errorf("%w: address %.40q is not a TRON Base58Check address",
			ErrInvalidPQKeystore, clearFields.Address)
	}
	return scheme, clearFields.Address, nil
}

// readPQAccount reads the account in folder dir. The folder must hold
// exactly one regular, non-hidden *.json file.
func readPQAccount(dir, name string) (PQAccount, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return PQAccount{}, fmt.Errorf("%w: %v", ErrInvalidPQKeystore, err)
	}
	var files []string
	for _, e := range entries {
		n := e.Name()
		if e.Type().IsRegular() && !strings.HasPrefix(n, ".") && strings.EqualFold(filepath.Ext(n), ".json") {
			files = append(files, n)
		}
	}
	if len(files) != 1 {
		return PQAccount{}, fmt.Errorf("%w: account %q has %d keystore files, want 1",
			ErrInvalidPQKeystore, name, len(files))
	}
	p := filepath.Join(dir, files[0])
	data, err := readLimited(p)
	if err != nil {
		return PQAccount{}, fmt.Errorf("%w: %v", ErrInvalidPQKeystore, err)
	}
	scheme, addr, err := parsePQClearFields(data)
	if err != nil {
		return PQAccount{}, fmt.Errorf("account %q: %w", name, err)
	}
	return PQAccount{Name: name, Address: addr, Scheme: scheme, Path: p}, nil
}

func pqAccountsIn(root string) ([]PQAccount, error) {
	accounts, _, err := scanPQAccountsIn(root)
	return accounts, err
}

func scanPQAccountsIn(root string) ([]PQAccount, []UnreadablePQAccount, error) {
	pqDir := pqKeysDirIn(root)
	entries, err := os.ReadDir(pqDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PQAccount{}, []UnreadablePQAccount{}, nil
		}
		return nil, nil, fmt.Errorf("read %s: %w", pqDir, err)
	}
	accounts := []PQAccount{}
	unreadable := []UnreadablePQAccount{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := ValidateAccountName(e.Name()); err != nil {
			unreadable = append(unreadable, UnreadablePQAccount{Name: e.Name(), Err: err})
			continue
		}
		acct, err := readPQAccount(filepath.Join(pqDir, e.Name()), e.Name())
		if err != nil {
			unreadable = append(unreadable, UnreadablePQAccount{Name: e.Name(), Err: err})
			continue
		}
		accounts = append(accounts, acct)
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].Name < accounts[j].Name })
	sort.Slice(unreadable, func(i, j int) bool { return unreadable[i].Name < unreadable[j].Name })
	return accounts, unreadable, nil
}

// notADirectory is the error for a pq-keys/<name> entry that is not a real
// folder (a plain file, or a symlink, which is never followed). The entry
// takes the name (see NameExists) but is not an account, so the store
// neither reads nor removes it and the user must deal with it.
func notADirectory(name string) error {
	return fmt.Errorf("%w: %s/%s is not a directory; remove it manually",
		ErrInvalidPQKeystore, c.DefaultConfigPQKeysDirName, name)
}

func pqAccountByNameIn(root, name string) (PQAccount, error) {
	if err := ValidateAccountName(name); err != nil {
		return PQAccount{}, err
	}
	dir := filepath.Join(pqKeysDirIn(root), name)
	fi, err := os.Lstat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return PQAccount{}, fmt.Errorf("%w: %s", ErrPQAccountNotFound, name)
		}
		return PQAccount{}, fmt.Errorf("stat %s: %w", dir, err)
	}
	if !fi.IsDir() {
		return PQAccount{}, notADirectory(name)
	}
	return readPQAccount(dir, name)
}

func pqAccountByAddressIn(root, addr string) (PQAccount, error) {
	accounts, err := pqAccountsIn(root)
	if err != nil {
		return PQAccount{}, err
	}
	for _, a := range accounts {
		if addr != "" && a.Address == addr {
			return a, nil
		}
	}
	return PQAccount{}, fmt.Errorf("%w: %.40q", ErrPQAccountNotFound, addr)
}

// verifyPQKeystore decrypts data and returns its clear fields. The binding of
// the clear scheme and address to the decrypted key is enforced by
// pqc.DecryptKeystore, which fails with pqc.ErrKeyMismatch when they do not
// describe the key. The key is destroyed before return.
func verifyPQKeystore(data []byte, password string) (core.PQScheme, string, error) {
	scheme, addr, err := parsePQClearFields(data)
	if err != nil {
		return 0, "", err
	}
	k, err := pqc.DecryptKeystore(data, password)
	if err != nil {
		return 0, "", err
	}
	defer k.Destroy()
	// Defense in depth: pqc.DecryptKeystore already binds the clear scheme
	// and address to the key, so this cannot fail today. It keeps the store
	// from writing a mislabeled keystore should that check ever change.
	if k.Scheme() != scheme || k.Public().Address().String() != addr {
		return 0, "", fmt.Errorf("%w: keystore fields do not describe the decrypted key", pqc.ErrKeyMismatch)
	}
	return scheme, addr, nil
}

// checkPQAccountFree returns an error wrapping ErrAccountNameExists if name
// is taken in account-keys or pq-keys, or ErrPQAddressExists if addr is
// already stored in pq-keys. An empty addr skips the address check.
func checkPQAccountFree(root, name, addr string) error {
	if nameExistsIn(root, name) {
		return fmt.Errorf("%w: %s", ErrAccountNameExists, name)
	}
	return checkPQAddressFree(root, addr)
}

// checkPQAddressFree returns an error wrapping ErrPQAddressExists if addr is
// already stored in pq-keys. An empty addr is never taken.
func checkPQAddressFree(root, addr string) error {
	if addr == "" {
		return nil
	}
	if existing, err := pqAccountByAddressIn(root, addr); err == nil {
		return fmt.Errorf("%w: %s (account %q)", ErrPQAddressExists, addr, existing.Name)
	} else if !errors.Is(err, ErrPQAccountNotFound) {
		return err
	}
	return nil
}

// writePQAccount re-checks, under pqWriteMu, that name and address are free
// and writes data to pq-keys/<name>/UTC--<time>--<address>.json with mode
// 0600.
func writePQAccount(root, name string, scheme core.PQScheme, addr string, data []byte) (PQAccount, error) {
	pqWriteMu.Lock()
	defer pqWriteMu.Unlock()

	if err := checkPQAccountFree(root, name, addr); err != nil {
		return PQAccount{}, err
	}

	pqDir := pqKeysDirIn(root)
	if err := os.MkdirAll(pqDir, 0700); err != nil {
		return PQAccount{}, fmt.Errorf("create %s: %w", pqDir, err)
	}
	dir := filepath.Join(pqDir, name)
	if err := os.Mkdir(dir, 0700); err != nil {
		if os.IsExist(err) {
			return PQAccount{}, fmt.Errorf("%w: %s", ErrAccountNameExists, name)
		}
		return PQAccount{}, fmt.Errorf("create %s: %w", dir, err)
	}
	p := filepath.Join(dir, fmt.Sprintf("UTC--%s--%s.json",
		time.Now().UTC().Format("2006-01-02T15-04-05.000000000Z"), addr))
	if err := writeNewFile(p, data); err != nil {
		// dir was created above and holds nothing else; drop it so the
		// name is not left taken by an empty folder.
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			err = errors.Join(err, fmt.Errorf("clean up %s: %w", dir, rmErr))
		}
		return PQAccount{}, err
	}
	return PQAccount{Name: name, Address: addr, Scheme: scheme, Path: p}, nil
}

// writeNewFile creates path (which must not exist) with mode 0600 and syncs
// data to it.
func writeNewFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", path, err)
	}
	return nil
}

func importPQKeystoreIn(root, name string, keystoreJSON []byte, password string) (PQAccount, error) {
	// Cheap pre-checks, without the lock, before the scrypt key derivation:
	// a taken name or address is reported whatever the password. The clear
	// address is not verified yet; writePQAccount re-checks the verified one
	// under pqWriteMu.
	if err := ValidateAccountName(name); err != nil {
		return PQAccount{}, err
	}
	if err := checkPQAccountFree(root, name, ""); err != nil {
		return PQAccount{}, err
	}
	_, clearAddr, err := parsePQClearFields(keystoreJSON)
	if err != nil {
		return PQAccount{}, fmt.Errorf("import post-quantum keystore: %w", err)
	}
	if err := checkPQAddressFree(root, clearAddr); err != nil {
		return PQAccount{}, err
	}

	scheme, addr, err := verifyPQKeystore(keystoreJSON, password)
	if err != nil {
		return PQAccount{}, fmt.Errorf("import post-quantum keystore: %w", err)
	}
	return writePQAccount(root, name, scheme, addr, keystoreJSON)
}

func savePQKeyIn(root, name string, k pqc.PrivateKey, password string, scryptN, scryptP int) (PQAccount, error) {
	// Cheap pre-checks, without the lock, before the scrypt key derivation;
	// writePQAccount re-checks under pqWriteMu. A nil key skips the address
	// check and is rejected by pqc.EncryptKeystore.
	if err := ValidateAccountName(name); err != nil {
		return PQAccount{}, err
	}
	var keyAddr string
	if k != nil {
		if pub := k.Public(); pub != nil {
			keyAddr = pub.Address().String()
		}
	}
	if err := checkPQAccountFree(root, name, keyAddr); err != nil {
		return PQAccount{}, err
	}

	data, err := pqc.EncryptKeystore(k, password, scryptN, scryptP)
	if err != nil {
		return PQAccount{}, fmt.Errorf("save post-quantum key: %w", err)
	}
	scheme, addr, err := parsePQClearFields(data)
	if err != nil {
		return PQAccount{}, fmt.Errorf("save post-quantum key: %w", err)
	}
	return writePQAccount(root, name, scheme, addr, data)
}

func removePQAccountIn(root, name string) error {
	if err := ValidateAccountName(name); err != nil {
		return err
	}
	pqDir := pqKeysDirIn(root)
	target := filepath.Join(pqDir, name)
	if filepath.Dir(target) != pqDir {
		return fmt.Errorf("%w %q", ErrInvalidAccountName, name)
	}

	pqWriteMu.Lock()
	defer pqWriteMu.Unlock()
	fi, err := os.Lstat(target)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrPQAccountNotFound, name)
		}
		return fmt.Errorf("stat %s: %w", target, err)
	}
	if !fi.IsDir() {
		return notADirectory(name)
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("failed to remove post-quantum account %s: %w", name, err)
	}
	return nil
}

// DecryptPQKey reads acct's keystore and decrypts it with password. The key
// must match acct's address and scheme, else the error wraps
// pqc.ErrKeyMismatch. A wrong password returns an error wrapping
// keystore.ErrDecrypt. The caller owns the returned key and should Destroy
// it when done.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func DecryptPQKey(acct PQAccount, password string) (pqc.PrivateKey, error) {
	data, err := readLimited(acct.Path)
	if err != nil {
		return nil, fmt.Errorf("read post-quantum account %q: %w", acct.Name, err)
	}
	k, err := pqc.DecryptKeystore(data, password)
	if err != nil {
		return nil, fmt.Errorf("decrypt post-quantum account %q: %w", acct.Name, err)
	}
	if k.Scheme() != acct.Scheme || k.Public().Address().String() != acct.Address {
		k.Destroy()
		return nil, fmt.Errorf("%w: post-quantum account %q keystore does not match its address %s",
			pqc.ErrKeyMismatch, acct.Name, acct.Address)
	}
	return k, nil
}

// UnreadablePQAccount is a folder in pq-keys that ScanPQAccounts could not
// read as an account: its name is not a valid account name, or it does not
// hold exactly one readable keystore with a supported scheme and a valid
// address. Err says why. It never holds key material.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
type UnreadablePQAccount struct {
	// Name is the folder name.
	Name string
	// Err is the reason the folder is not an account.
	Err error
}

// --- Store methods ---

// PQKeysDir returns the post-quantum keystore folder, <configRoot>/pq-keys.
// It is not created until the first key is stored. It returns "" when the
// config directory cannot be resolved (a relative directory without a home
// directory); do not join the empty string with a name and write to it.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) PQKeysDir() string {
	root, err := s.root()
	if err != nil {
		return ""
	}
	return pqKeysDirIn(root)
}

// NameExists reports whether name is used by an ECDSA account (account-keys)
// or a post-quantum account (pq-keys). New code that creates accounts should
// use it instead of DoesNamedAccountExist, which only checks account-keys.
// An invalid name always reports false, so callers should check it with
// ValidateAccountName first. Any entry counts, including a plain file or a
// symlink, and a stat error other than "not exist" counts as taken. So does a
// valid name when the config directory cannot be resolved (a relative
// directory without a home directory): the name cannot be shown to be free, so
// a caller that checks before creating stops there.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) NameExists(name string) bool {
	root, err := s.root()
	if err != nil {
		return ValidateAccountName(name) == nil
	}
	return nameExistsIn(root, name)
}

// PQAccounts lists the stored post-quantum accounts, sorted by name. Only the
// clear scheme and address fields are read; nothing is decrypted. Entries that
// are not folders, and folders that do not hold exactly one readable keystore
// with a supported scheme and a valid address, are skipped without an error;
// use ScanPQAccounts to be told which folders were skipped and why. The error
// is non-nil only when the pq-keys folder exists but cannot be read, or the
// config directory cannot be resolved (a relative directory without a home
// directory); a missing folder yields an empty list.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) PQAccounts() ([]PQAccount, error) {
	accounts, _, err := s.ScanPQAccounts()
	return accounts, err
}

// ScanPQAccounts lists the stored post-quantum accounts like PQAccounts, and
// also the folders of pq-keys that are not usable accounts: a folder with an
// invalid account name, or one that does not hold exactly one readable
// keystore with a supported scheme and a valid address. Both lists are sorted
// by name. Plain files and symlinks in pq-keys (for example a .DS_Store) are
// ignored. Only the clear scheme and address fields are read; nothing is
// decrypted and the unreadable entries carry no key material. The error is
// non-nil only when the pq-keys folder exists but cannot be read, or the
// config directory cannot be resolved; a missing folder yields empty lists.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) ScanPQAccounts() ([]PQAccount, []UnreadablePQAccount, error) {
	root, err := s.root()
	if err != nil {
		return nil, nil, err
	}
	return scanPQAccountsIn(root)
}

// PQAccountByName returns the post-quantum account stored under name. A name
// with no entry in pq-keys returns an error wrapping ErrPQAccountNotFound. An
// entry that is not a real folder (a plain file or a symlink, which is never
// followed) returns one wrapping ErrInvalidPQKeystore that says to remove it
// manually, as does a folder without a valid keystore.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) PQAccountByName(name string) (PQAccount, error) {
	root, err := s.root()
	if err != nil {
		return PQAccount{}, err
	}
	return pqAccountByNameIn(root, name)
}

// PQAccountByAddress returns the post-quantum account with the Base58
// address addr, or an error wrapping ErrPQAccountNotFound.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) PQAccountByAddress(addr string) (PQAccount, error) {
	root, err := s.root()
	if err != nil {
		return PQAccount{}, err
	}
	return pqAccountByAddressIn(root, addr)
}

// ImportPQKeystore stores a wallet-cli post-quantum keystore under name. The
// keystore must decrypt with password to a key matching its clear scheme and
// address; the decrypted key is destroyed immediately. The name must be free
// in both account-keys and pq-keys (ErrAccountNameExists) and the address
// must not already be in pq-keys (ErrPQAddressExists). The name and the
// clear address are checked before the keystore is decrypted, so a taken
// name or address is reported whatever the password, and checked again
// before writing. The original bytes are written, with mode 0600, to
// pq-keys/<name>/UTC--<time>--<address>.json. Nothing is written when any
// check fails.
//
// Name uniqueness holds across processes; address uniqueness is enforced
// only within one process, so two processes importing the same keystore
// under different names at the same time can both succeed.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) ImportPQKeystore(name string, keystoreJSON []byte, password string) (PQAccount, error) {
	root, err := s.root()
	if err != nil {
		return PQAccount{}, err
	}
	return importPQKeystoreIn(root, name, keystoreJSON, password)
}

// SavePQKey encrypts k with password (see pqc.EncryptKeystore for scryptN
// and scryptP) and stores it under name, with the same checks and file
// layout as ImportPQKeystore. The name and the key's address are checked
// before the slow encryption and again before writing. As with
// ImportPQKeystore, name uniqueness holds across processes but address
// uniqueness only within one. The caller owns k and must Destroy it.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) SavePQKey(name string, k pqc.PrivateKey, password string, scryptN, scryptP int) (PQAccount, error) {
	root, err := s.root()
	if err != nil {
		return PQAccount{}, err
	}
	return savePQKeyIn(root, name, k, password, scryptN, scryptP)
}

// DecryptPQKey decrypts acct's keystore; see the package-level DecryptPQKey.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) DecryptPQKey(acct PQAccount, password string) (pqc.PrivateKey, error) {
	return DecryptPQKey(acct, password)
}

// RemovePQAccount deletes the post-quantum account folder pq-keys/<name>.
// It never touches account-keys. A name with no entry in pq-keys returns an
// error wrapping ErrPQAccountNotFound. An entry that is not a real folder (a
// plain file or a symlink) is neither followed nor removed; the error wraps
// ErrInvalidPQKeystore and says to remove it manually.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func (s *Store) RemovePQAccount(name string) error {
	root, err := s.root()
	if err != nil {
		return err
	}
	return removePQAccountIn(root, name)
}

// --- Package-level functions (default config root) ---

// PQKeysDir returns the post-quantum keystore folder, <configRoot>/pq-keys.
// It follows SetDefaultLocation and is not created until the first key is
// stored.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func PQKeysDir() string {
	root, err := configRoot()
	if err != nil {
		return ""
	}
	return pqKeysDirIn(root)
}

// NameExists reports whether name is used in account-keys or pq-keys; see
// Store.NameExists. New code that creates accounts should use it instead of
// DoesNamedAccountExist. An invalid name always reports false, so callers
// should check it with ValidateAccountName first.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func NameExists(name string) bool {
	root, err := configRoot()
	if err != nil {
		return ValidateAccountName(name) == nil
	}
	return nameExistsIn(root, name)
}

// PQAccounts lists the stored post-quantum accounts; see Store.PQAccounts.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func PQAccounts() ([]PQAccount, error) {
	accounts, _, err := ScanPQAccounts()
	return accounts, err
}

// ScanPQAccounts lists the stored post-quantum accounts and the unusable
// folders of pq-keys; see Store.ScanPQAccounts.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func ScanPQAccounts() ([]PQAccount, []UnreadablePQAccount, error) {
	root, err := configRoot()
	if err != nil {
		return nil, nil, err
	}
	return scanPQAccountsIn(root)
}

// PQAccountByName returns the post-quantum account stored under name; see
// Store.PQAccountByName.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func PQAccountByName(name string) (PQAccount, error) {
	root, err := configRoot()
	if err != nil {
		return PQAccount{}, err
	}
	return pqAccountByNameIn(root, name)
}

// PQAccountByAddress returns the post-quantum account with the Base58
// address addr; see Store.PQAccountByAddress.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func PQAccountByAddress(addr string) (PQAccount, error) {
	root, err := configRoot()
	if err != nil {
		return PQAccount{}, err
	}
	return pqAccountByAddressIn(root, addr)
}

// ImportPQKeystore stores a wallet-cli post-quantum keystore under name; see
// Store.ImportPQKeystore.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func ImportPQKeystore(name string, keystoreJSON []byte, password string) (PQAccount, error) {
	root, err := configRoot()
	if err != nil {
		return PQAccount{}, err
	}
	return importPQKeystoreIn(root, name, keystoreJSON, password)
}

// SavePQKey encrypts and stores k under name; see Store.SavePQKey. The caller
// owns k and must Destroy it.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func SavePQKey(name string, k pqc.PrivateKey, password string, scryptN, scryptP int) (PQAccount, error) {
	root, err := configRoot()
	if err != nil {
		return PQAccount{}, err
	}
	return savePQKeyIn(root, name, k, password, scryptN, scryptP)
}

// RemovePQAccount deletes the post-quantum account folder pq-keys/<name>;
// see Store.RemovePQAccount.
//
// Experimental: TIP-899 post-quantum support is a draft (Nile only) and may change.
func RemovePQAccount(name string) error {
	root, err := configRoot()
	if err != nil {
		return err
	}
	return removePQAccountIn(root, name)
}
