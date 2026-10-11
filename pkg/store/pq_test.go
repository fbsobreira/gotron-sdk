package store_test

import (
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newMLKey generates an ML-DSA-44 key that is destroyed when the test ends.
func newMLKey(t *testing.T) pqc.PrivateKey {
	t.Helper()
	k, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	require.NoError(t, err)
	t.Cleanup(k.Destroy)
	return k
}

// mlKeystore returns an encrypted ML-DSA-44 keystore and its address.
func mlKeystore(t *testing.T, password string) ([]byte, string) {
	t.Helper()
	k := newMLKey(t)
	data, err := pqc.EncryptKeystore(k, password, keystore.LightScryptN, keystore.LightScryptP)
	require.NoError(t, err)
	return data, k.Public().Address().String()
}

// withClearAddress rewrites the clear "address" field of a keystore.
func withClearAddress(t *testing.T, data []byte, addr string) []byte {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	m["address"] = addr
	out, err := json.Marshal(m)
	require.NoError(t, err)
	return out
}

// newTempStoreRoot is newTempStore plus the config root it uses.
func newTempStoreRoot(t *testing.T) (*store.Store, string) {
	t.Helper()
	s := newTempStore(t)
	root := filepath.Dir(s.DefaultLocation())
	return s, root
}

func assertNotWritten(t *testing.T, s *store.Store, name string) {
	t.Helper()
	_, err := os.Lstat(filepath.Join(s.PQKeysDir(), name))
	assert.True(t, os.IsNotExist(err), "pq-keys/%s must not exist, stat err: %v", name, err)
	assert.False(t, s.NameExists(name))
}

func TestPQKeysDir(t *testing.T) {
	s, root := newTempStoreRoot(t)
	assert.Equal(t, filepath.Join(root, c.DefaultConfigPQKeysDirName), s.PQKeysDir())
	assert.Equal(t, "pq-keys", c.DefaultConfigPQKeysDirName)
	assert.Equal(t, filepath.Dir(s.DefaultLocation()), filepath.Dir(s.PQKeysDir()),
		"pq-keys must sit next to account-keys")
}

func TestImportPQKeystore_RoundTrip(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	data, addr := mlKeystore(t, "pw")

	acct, err := s.ImportPQKeystore("alice", data, "pw")
	require.NoError(t, err)
	assert.Equal(t, "alice", acct.Name)
	assert.Equal(t, addr, acct.Address)
	assert.Equal(t, core.PQScheme_ML_DSA_44, acct.Scheme)

	// File layout, name, mode, and content.
	assert.Equal(t, filepath.Join(s.PQKeysDir(), "alice"), filepath.Dir(acct.Path))
	base := filepath.Base(acct.Path)
	assert.True(t, strings.HasPrefix(base, "UTC--"), base)
	assert.True(t, strings.HasSuffix(base, "--"+addr+".json"), base)
	got, err := os.ReadFile(acct.Path)
	require.NoError(t, err)
	assert.Equal(t, data, got, "the original bytes must be stored")
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(acct.Path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), fi.Mode().Perm())
		di, err := os.Stat(filepath.Dir(acct.Path))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0700), di.Mode().Perm())
	}

	// Lookups.
	assert.True(t, s.NameExists("alice"))
	assert.False(t, s.DoesNamedAccountExist("alice"), "existing ECDSA lookup is unchanged")
	list, err := s.PQAccounts()
	require.NoError(t, err)
	require.Equal(t, []store.PQAccount{acct}, list)
	byName, err := s.PQAccountByName("alice")
	require.NoError(t, err)
	assert.Equal(t, acct, byName)
	byAddr, err := s.PQAccountByAddress(addr)
	require.NoError(t, err)
	assert.Equal(t, acct, byAddr)

	// Decryption.
	k, err := s.DecryptPQKey(acct, "pw")
	require.NoError(t, err)
	assert.Equal(t, addr, k.Public().Address().String())
	k.Destroy()

	_, err = s.DecryptPQKey(acct, "wrong")
	require.Error(t, err)
	assert.ErrorIs(t, err, keystore.ErrDecrypt)

	tampered := acct
	tampered.Address = newMLKey(t).Public().Address().String()
	_, err = s.DecryptPQKey(tampered, "pw")
	require.Error(t, err)
	assert.ErrorIs(t, err, pqc.ErrKeyMismatch)
}

func TestImportPQKeystore_PackageLevel(t *testing.T) {
	acctDir := withTempLocation(t)
	root := filepath.Dir(acctDir)
	data, addr := mlKeystore(t, "pw")

	assert.Equal(t, filepath.Join(root, "pq-keys"), store.PQKeysDir())
	acct, err := store.ImportPQKeystore("pkg-level", data, "pw")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "pq-keys", "pkg-level"), filepath.Dir(acct.Path))
	assert.True(t, store.NameExists("pkg-level"))

	list, err := store.PQAccounts()
	require.NoError(t, err)
	assert.Equal(t, []store.PQAccount{acct}, list)
	byName, err := store.PQAccountByName("pkg-level")
	require.NoError(t, err)
	assert.Equal(t, acct, byName)
	byAddr, err := store.PQAccountByAddress(addr)
	require.NoError(t, err)
	assert.Equal(t, acct, byAddr)

	k, err := store.DecryptPQKey(acct, "pw")
	require.NoError(t, err)
	k.Destroy()

	saved, err := store.SavePQKey("pkg-saved", newMLKey(t), "pw2", keystore.LightScryptN, keystore.LightScryptP)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "pq-keys", "pkg-saved"), filepath.Dir(saved.Path))

	require.NoError(t, store.RemovePQAccount("pkg-level"))
	_, err = store.PQAccountByName("pkg-level")
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound)
}

func TestSetDefaultLocation_MovesPQKeys(t *testing.T) {
	first := filepath.Dir(withTempLocation(t))
	assert.Equal(t, filepath.Join(first, "pq-keys"), store.PQKeysDir())

	data, _ := mlKeystore(t, "pw")
	_, err := store.ImportPQKeystore("mover", data, "pw")
	require.NoError(t, err)

	second := t.TempDir()
	store.SetDefaultLocation(second)
	assert.Equal(t, filepath.Join(second, "pq-keys"), store.PQKeysDir())
	assert.Equal(t, filepath.Dir(store.DefaultLocation()), filepath.Dir(store.PQKeysDir()))
	list, err := store.PQAccounts()
	require.NoError(t, err)
	assert.Empty(t, list, "the new location has no PQ keys")
	assert.False(t, store.NameExists("mover"))

	s := store.NewStore(first)
	s.SetDefaultLocation(second)
	assert.Equal(t, filepath.Join(second, "pq-keys"), s.PQKeysDir())
}

func TestImportPQKeystore_WrongPassword(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	data, _ := mlKeystore(t, "pw")

	_, err := s.ImportPQKeystore("alice", data, "not-pw")
	require.Error(t, err)
	assert.ErrorIs(t, err, keystore.ErrDecrypt)
	assertNotWritten(t, s, "alice")
}

func TestImportPQKeystore_MismatchedClearAddress(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	data, _ := mlKeystore(t, "pw")
	other := newMLKey(t).Public().Address().String()

	_, err := s.ImportPQKeystore("alice", withClearAddress(t, data, other), "pw")
	require.Error(t, err)
	assert.ErrorIs(t, err, pqc.ErrKeyMismatch)
	assertNotWritten(t, s, "alice")
}

func TestImportPQKeystore_NotAKeystore(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	for name, data := range map[string][]byte{
		"garbage":   []byte("not json"),
		"ecdsa-ish": []byte(`{"address":"41aa","crypto":{},"version":3}`),
		"empty":     nil,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := s.ImportPQKeystore("x", data, "pw")
			require.Error(t, err)
			assert.ErrorIs(t, err, store.ErrInvalidPQKeystore)
			assertNotWritten(t, s, "x")
		})
	}
}

func TestImportPQKeystore_DuplicateName(t *testing.T) {
	s, _ := newTempStoreRoot(t)

	// ECDSA folder holds "eve".
	require.NoError(t, os.MkdirAll(filepath.Join(s.DefaultLocation(), "eve"), 0700))
	data, _ := mlKeystore(t, "pw")
	_, err := s.ImportPQKeystore("eve", data, "pw")
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrAccountNameExists)
	_, statErr := os.Lstat(filepath.Join(s.PQKeysDir(), "eve"))
	assert.True(t, os.IsNotExist(statErr))
	assert.True(t, s.NameExists("eve"), "a name in account-keys exists")

	// PQ folder holds "bob".
	_, err = s.ImportPQKeystore("bob", data, "pw")
	require.NoError(t, err)
	data2, _ := mlKeystore(t, "pw")
	_, err = s.ImportPQKeystore("bob", data2, "pw")
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrAccountNameExists)
	_, err = s.SavePQKey("bob", newMLKey(t), "pw", keystore.LightScryptN, keystore.LightScryptP)
	assert.ErrorIs(t, err, store.ErrAccountNameExists)

	list, err := s.PQAccounts()
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

func TestImportPQKeystore_DuplicateAddress(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	data, _ := mlKeystore(t, "pw")

	_, err := s.ImportPQKeystore("first", data, "pw")
	require.NoError(t, err)
	_, err = s.ImportPQKeystore("second", data, "pw")
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrPQAddressExists)
	assertNotWritten(t, s, "second")
}

func TestPQ_InvalidNames(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	data, _ := mlKeystore(t, "pw")
	k := newMLKey(t)

	tests := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"dot", "."},
		{"dotdot", ".."},
		{"slash", "a/b"},
		{"traversal", "../escape"},
		{"backslash", `a\b`},
		{"absolute", "/abs"},
		{"nul", "a\x00b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := s.ImportPQKeystore(tt.in, data, "pw")
			assert.ErrorIs(t, err, store.ErrInvalidAccountName)
			_, err = s.SavePQKey(tt.in, k, "pw", keystore.LightScryptN, keystore.LightScryptP)
			assert.ErrorIs(t, err, store.ErrInvalidAccountName)
			_, err = s.PQAccountByName(tt.in)
			assert.ErrorIs(t, err, store.ErrInvalidAccountName)
			err = s.RemovePQAccount(tt.in)
			assert.ErrorIs(t, err, store.ErrInvalidAccountName)
			assert.False(t, s.NameExists(tt.in))
		})
	}

	// Nothing escaped into the config root or its parent.
	list, err := s.PQAccounts()
	require.NoError(t, err)
	assert.Empty(t, list)
	_, err = os.Lstat(filepath.Join(filepath.Dir(s.PQKeysDir()), "escape"))
	assert.True(t, os.IsNotExist(err))
}

func TestValidateAccountName(t *testing.T) {
	for _, bad := range []string{"", ".", "..", "a/b", "../escape", `a\b`, "/abs", "a\x00b"} {
		err := store.ValidateAccountName(bad)
		assert.ErrorIs(t, err, store.ErrInvalidAccountName, "%q", bad)
	}
	for _, good := range []string{"alice", "my-key_1", ".hidden", "a..b", "名前"} {
		assert.NoError(t, store.ValidateAccountName(good), "%q", good)
	}
}

func TestPQAccounts_ListingSkipsInvalidEntries(t *testing.T) {
	s, _ := newTempStoreRoot(t)

	// Missing pq-keys folder is an empty list, not an error.
	list, err := s.PQAccounts()
	require.NoError(t, err)
	assert.Empty(t, list)

	dataB, addrB := mlKeystore(t, "pw")
	bob, err := s.ImportPQKeystore("bob", dataB, "pw")
	require.NoError(t, err)
	alice, err := s.SavePQKey("alice", newMLKey(t), "pw", keystore.LightScryptN, keystore.LightScryptP)
	require.NoError(t, err)

	pq := s.PQKeysDir()
	write := func(rel string, content []byte) {
		p := filepath.Join(pq, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(p), 0700))
		require.NoError(t, os.WriteFile(p, content, 0600))
	}
	write("corrupt/UTC--x.json", []byte("{not json"))
	write("badscheme/UTC--x.json", withScheme(t, dataB, "RSA_4096"))
	write("badaddr/UTC--x.json", withClearAddress(t, dataB, "not-an-address"))
	write("twofiles/a.json", dataB)
	write("twofiles/b.json", dataB)
	write("nojson/readme.txt", []byte("hi"))
	require.NoError(t, os.MkdirAll(filepath.Join(pq, "emptydir"), 0700))
	write("stray-file.json", dataB)

	list, err = s.PQAccounts()
	require.NoError(t, err)
	require.Equal(t, []store.PQAccount{alice, bob}, list, "sorted by name, invalid entries skipped")

	for _, name := range []string{"corrupt", "badscheme", "badaddr", "twofiles", "nojson", "emptydir"} {
		_, err := s.PQAccountByName(name)
		require.Error(t, err, name)
		assert.NotErrorIs(t, err, store.ErrPQAccountNotFound, "%s exists but is unreadable", name)
		assert.True(t, s.NameExists(name), name)
	}

	_, err = s.PQAccountByName("missing")
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound)
	_, err = s.PQAccountByName("stray-file.json")
	assert.ErrorIs(t, err, store.ErrInvalidPQKeystore, "a plain file is reported, not hidden")
	assert.NotErrorIs(t, err, store.ErrPQAccountNotFound)

	got, err := s.PQAccountByAddress(addrB)
	require.NoError(t, err)
	assert.Equal(t, bob, got)
	_, err = s.PQAccountByAddress(newMLKey(t).Public().Address().String())
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound)
	_, err = s.PQAccountByAddress("")
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound)

	// An ECDSA-only name is not a PQ account.
	require.NoError(t, os.MkdirAll(filepath.Join(s.DefaultLocation(), "ecdsa"), 0700))
	_, err = s.PQAccountByName("ecdsa")
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound)
}

// withScheme rewrites the clear "scheme" field of a keystore.
func withScheme(t *testing.T, data []byte, scheme string) []byte {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	m["scheme"] = scheme
	out, err := json.Marshal(m)
	require.NoError(t, err)
	return out
}

func TestSavePQKey(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	k := newMLKey(t)

	acct, err := s.SavePQKey("carol", k, "pw", keystore.LightScryptN, keystore.LightScryptP)
	require.NoError(t, err)
	assert.Equal(t, k.Public().Address().String(), acct.Address)
	assert.Equal(t, core.PQScheme_ML_DSA_44, acct.Scheme)
	assert.NotNil(t, k.Bytes(), "SavePQKey must not destroy the caller's key")
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(acct.Path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), fi.Mode().Perm())
	}

	got, err := s.DecryptPQKey(acct, "pw")
	require.NoError(t, err)
	assert.True(t, got.Public().Equal(k.Public()))
	got.Destroy()

	_, err = s.SavePQKey("dup-addr", k, "pw", keystore.LightScryptN, keystore.LightScryptP)
	assert.ErrorIs(t, err, store.ErrPQAddressExists)
	assertNotWritten(t, s, "dup-addr")

	_, err = s.SavePQKey("nil-key", nil, "pw", keystore.LightScryptN, keystore.LightScryptP)
	require.Error(t, err)
	assertNotWritten(t, s, "nil-key")
}

func TestRemovePQAccount(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	data1, _ := mlKeystore(t, "pw")
	data2, _ := mlKeystore(t, "pw")
	_, err := s.ImportPQKeystore("gone", data1, "pw")
	require.NoError(t, err)
	kept, err := s.ImportPQKeystore("kept", data2, "pw")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(s.DefaultLocation(), "ecdsa"), 0700))

	require.NoError(t, s.RemovePQAccount("gone"))
	_, err = s.PQAccountByName("gone")
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound)
	assertNotWritten(t, s, "gone")

	list, err := s.PQAccounts()
	require.NoError(t, err)
	assert.Equal(t, []store.PQAccount{kept}, list)

	err = s.RemovePQAccount("gone")
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound)
	err = s.RemovePQAccount("ecdsa")
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound, "an ECDSA account is not removed")
	_, err = os.Stat(filepath.Join(s.DefaultLocation(), "ecdsa"))
	assert.NoError(t, err)

	// A plain file in pq-keys is not an account and is left alone.
	stray := filepath.Join(s.PQKeysDir(), "stray")
	require.NoError(t, os.WriteFile(stray, []byte("x"), 0600))
	err = s.RemovePQAccount("stray")
	assert.ErrorIs(t, err, store.ErrInvalidPQKeystore)
	assert.NotErrorIs(t, err, store.ErrPQAccountNotFound)
	_, err = os.Stat(stray)
	assert.NoError(t, err)

	// The pq-keys folder itself survives.
	_, err = os.Stat(s.PQKeysDir())
	assert.NoError(t, err)
}

// A taken name or address is reported before the keystore is decrypted, so
// the answer does not depend on the password.
func TestImportPQKeystore_TakenBeforeDecrypt(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	data, _ := mlKeystore(t, "pw")
	_, err := s.ImportPQKeystore("taken", data, "pw")
	require.NoError(t, err)

	// Taken name in pq-keys, wrong password.
	other, _ := mlKeystore(t, "pw")
	_, err = s.ImportPQKeystore("taken", other, "wrong")
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrAccountNameExists)
	assert.NotErrorIs(t, err, keystore.ErrDecrypt)

	// Taken name in account-keys, wrong password.
	require.NoError(t, os.MkdirAll(filepath.Join(s.DefaultLocation(), "ecdsa"), 0700))
	_, err = s.ImportPQKeystore("ecdsa", other, "wrong")
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrAccountNameExists)
	assert.NotErrorIs(t, err, keystore.ErrDecrypt)

	// Free name, stored address, wrong password.
	_, err = s.ImportPQKeystore("fresh", data, "wrong")
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrPQAddressExists)
	assert.NotErrorIs(t, err, keystore.ErrDecrypt)
	assertNotWritten(t, s, "fresh")

	list, err := s.PQAccounts()
	require.NoError(t, err)
	assert.Len(t, list, 1)
}

// A symlink in pq-keys is never followed: it is not listed, not read, and
// not removed, and its target is left alone.
func TestPQ_SymlinkNotFollowed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on Windows")
	}
	s, _ := newTempStoreRoot(t)
	data, _ := mlKeystore(t, "pw")

	// A valid account folder outside the store.
	outside := t.TempDir()
	target := filepath.Join(outside, "victim")
	require.NoError(t, os.Mkdir(target, 0700))
	targetFile := filepath.Join(target, "UTC--x.json")
	require.NoError(t, os.WriteFile(targetFile, data, 0600))

	require.NoError(t, os.MkdirAll(s.PQKeysDir(), 0700))
	link := filepath.Join(s.PQKeysDir(), "link")
	require.NoError(t, os.Symlink(target, link))

	assert.True(t, s.NameExists("link"), "the name is taken by the link")

	_, err := s.PQAccountByName("link")
	require.Error(t, err, "by-name must not follow the link")
	assert.ErrorIs(t, err, store.ErrInvalidPQKeystore)
	assert.NotErrorIs(t, err, store.ErrPQAccountNotFound)

	err = s.RemovePQAccount("link")
	require.Error(t, err, "remove must refuse the link")
	assert.ErrorIs(t, err, store.ErrInvalidPQKeystore)

	_, err = os.Lstat(link)
	assert.NoError(t, err, "the link itself is left in place")
	got, err := os.ReadFile(targetFile)
	require.NoError(t, err, "the link target must survive")
	assert.Equal(t, data, got)

	list, err := s.PQAccounts()
	require.NoError(t, err)
	assert.Empty(t, list, "a symlinked folder is not listed")
}

// A non-directory entry in pq-keys takes the name, so lookups report it
// with a distinct error telling the user to remove it, and leave it alone.
func TestPQ_NonDirectoryEntry(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	require.NoError(t, os.MkdirAll(s.PQKeysDir(), 0700))
	plain := filepath.Join(s.PQKeysDir(), "plain")
	require.NoError(t, os.WriteFile(plain, []byte("x"), 0600))

	assert.True(t, s.NameExists("plain"))
	_, err := s.PQAccountByName("plain")
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrInvalidPQKeystore)
	assert.NotErrorIs(t, err, store.ErrPQAccountNotFound)
	assert.Contains(t, err.Error(), "pq-keys/plain is not a directory; remove it manually")

	err = s.RemovePQAccount("plain")
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrInvalidPQKeystore)
	assert.NotErrorIs(t, err, store.ErrPQAccountNotFound)
	assert.Contains(t, err.Error(), "pq-keys/plain is not a directory; remove it manually")
	_, err = os.Stat(plain)
	assert.NoError(t, err, "the entry is not removed")

	// A truly missing entry is still not found.
	_, err = s.PQAccountByName("missing")
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound)
	err = s.RemovePQAccount("missing")
	assert.ErrorIs(t, err, store.ErrPQAccountNotFound)
}
