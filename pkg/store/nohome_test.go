package store_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noHome removes the home directory from the environment, so a relative
// config directory cannot be resolved, and moves into an empty temp directory
// so any write relative to the current directory is detectable. It returns
// that directory.
func noHome(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "") // Plan 9
	if _, err := os.UserHomeDir(); err == nil {
		t.Skipf("os.UserHomeDir still succeeds on %s without HOME", runtime.GOOS)
	}
	dir := t.TempDir()
	t.Chdir(dir)
	return dir
}

func requireEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Empty(t, names, "nothing may be created relative to the current directory")
}

const relDir = ".tronctl-nohome-test"

func requireHomeError(t *testing.T, err error, what string) {
	t.Helper()
	require.Error(t, err, what)
	assert.Contains(t, err.Error(), "home directory", what)
	assert.ErrorIs(t, err, store.ErrNoConfigDir, what)
}

func TestStore_NoHomeDirectory(t *testing.T) {
	cwd := noHome(t)
	s := store.NewStore(relDir)
	t.Cleanup(s.CloseAll)
	pqData, _ := mlKeystore(t, "pw")
	addr := newMLKey(t).Public().Address().String()
	tronAddr := addr

	assert.NotPanics(t, s.InitConfigDir)
	assert.NotPanics(t, s.DescribeLocalAccounts)
	assert.NotPanics(t, func() { s.SetDefaultLocation("another-relative-dir") })
	// SetDefaultLocation recorded the new directory; it must still not resolve.
	assert.Equal(t, "", s.DefaultLocation())
	s.SetDefaultLocation(relDir)

	// Methods without an error result degrade to a safe value.
	assert.Equal(t, []string{}, s.LocalAccounts())
	assert.False(t, s.DoesNamedAccountExist("alice"))
	assert.Nil(t, s.FromAddress(tronAddr))
	assert.Nil(t, s.FromAccountName("alice"))
	assert.Equal(t, "", s.DefaultLocation())
	assert.Equal(t, "", s.PQKeysDir())
	// A valid name counts as taken (it cannot be shown to be free); an invalid
	// name never does.
	assert.True(t, s.NameExists("alice"))
	assert.False(t, s.NameExists("../x"))
	assert.False(t, s.NameExists(""))

	// Methods with an error result return it.
	_, err := s.AddressFromAccountName("alice")
	requireHomeError(t, err, "AddressFromAccountName")
	_, _, err = s.UnlockedKeystore(tronAddr, "pw")
	requireHomeError(t, err, "UnlockedKeystore")
	_, err = s.PQAccounts()
	requireHomeError(t, err, "PQAccounts")
	good, bad, err := s.ScanPQAccounts()
	requireHomeError(t, err, "ScanPQAccounts")
	assert.Nil(t, good)
	assert.Nil(t, bad)
	_, err = s.PQAccountByName("alice")
	requireHomeError(t, err, "PQAccountByName")
	_, err = s.PQAccountByAddress(addr)
	requireHomeError(t, err, "PQAccountByAddress")
	_, err = s.ImportPQKeystore("alice", pqData, "pw")
	requireHomeError(t, err, "ImportPQKeystore")
	_, err = s.SavePQKey("alice", newMLKey(t), "pw", keystore.LightScryptN, keystore.LightScryptP)
	requireHomeError(t, err, "SavePQKey")
	requireHomeError(t, s.RemovePQAccount("alice"), "RemovePQAccount")

	requireEmptyDir(t, cwd)
}

func TestPackageLevel_NoHomeDirectory(t *testing.T) {
	cwd := noHome(t)
	orig := c.DefaultConfigDirName
	t.Cleanup(func() { c.DefaultConfigDirName = orig })
	c.DefaultConfigDirName = relDir
	t.Cleanup(store.CloseAll)
	pqData, _ := mlKeystore(t, "pw")
	addr := newMLKey(t).Public().Address().String()

	assert.NotPanics(t, store.InitConfigDir)
	assert.NotPanics(t, store.DescribeLocalAccounts)
	assert.NotPanics(t, func() { store.SetDefaultLocation(".other-relative-dir") })
	// The new directory is recorded; later calls do not use the old one.
	assert.Equal(t, ".other-relative-dir", c.DefaultConfigDirName)
	assert.Equal(t, "", store.DefaultLocation())
	assert.NotPanics(t, func() { _ = store.DefaultStoreInstance() })
	assert.Equal(t, "", store.DefaultStoreInstance().DefaultLocation())

	assert.Equal(t, []string{}, store.LocalAccounts())
	assert.False(t, store.DoesNamedAccountExist("alice"))
	assert.Nil(t, store.FromAddress(addr))
	assert.Nil(t, store.FromAccountName("alice"))
	assert.Equal(t, "", store.PQKeysDir())
	assert.True(t, store.NameExists("alice"))
	assert.False(t, store.NameExists("../x"))

	_, err := store.AddressFromAccountName("alice")
	requireHomeError(t, err, "AddressFromAccountName")
	_, _, err = store.UnlockedKeystore(addr, "pw")
	requireHomeError(t, err, "UnlockedKeystore")
	_, err = store.PQAccounts()
	requireHomeError(t, err, "PQAccounts")
	_, _, err = store.ScanPQAccounts()
	requireHomeError(t, err, "ScanPQAccounts")
	_, err = store.PQAccountByName("alice")
	requireHomeError(t, err, "PQAccountByName")
	_, err = store.PQAccountByAddress(addr)
	requireHomeError(t, err, "PQAccountByAddress")
	_, err = store.ImportPQKeystore("alice", pqData, "pw")
	requireHomeError(t, err, "ImportPQKeystore")
	_, err = store.SavePQKey("alice", newMLKey(t), "pw", keystore.LightScryptN, keystore.LightScryptP)
	requireHomeError(t, err, "SavePQKey")
	requireHomeError(t, store.RemovePQAccount("alice"), "RemovePQAccount")

	requireEmptyDir(t, cwd)
}

// An absolute directory does not need a home directory.
func TestStore_AbsoluteDirWithoutHome(t *testing.T) {
	noHome(t)
	root := t.TempDir()
	s := store.NewStore(root)
	t.Cleanup(s.CloseAll)
	s.InitConfigDir()

	assert.Equal(t, filepath.Join(root, c.DefaultConfigAccountAliasesDirName), s.DefaultLocation())
	assert.Equal(t, filepath.Join(root, c.DefaultConfigPQKeysDirName), s.PQKeysDir())
	assert.False(t, s.NameExists("alice"))
	_, err := s.PQAccounts()
	require.NoError(t, err)

	data, _ := mlKeystore(t, "pw")
	_, err = s.ImportPQKeystore("alice", data, "pw")
	require.NoError(t, err)
	assert.True(t, s.NameExists("alice"))
}
