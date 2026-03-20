package store_test

import (
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTempLocation sets DefaultConfigDirName to a temp directory and restores
// it when the test completes. It returns the full account-keys path.
func withTempLocation(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	origConfigDir := c.DefaultConfigDirName
	store.SetDefaultLocation(tmpDir)
	t.Cleanup(func() {
		c.DefaultConfigDirName = origConfigDir
	})

	return store.DefaultLocation()
}

// newTempStore creates a Store backed by a temp directory. The directory is
// automatically cleaned up when the test completes.
func newTempStore(t *testing.T) *store.Store {
	t.Helper()
	tmpDir := t.TempDir()
	s := store.NewStore(tmpDir)
	s.InitConfigDir()
	t.Cleanup(s.CloseAll)
	return s
}

func TestDefaultLocation(t *testing.T) {
	loc := store.DefaultLocation()
	assert.NotEmpty(t, loc, "default location should not be empty")
	assert.Contains(t, loc, "account-keys", "should contain account-keys directory")
}

func TestDoesNamedAccountExist_NotFound(t *testing.T) {
	exists := store.DoesNamedAccountExist("nonexistent-account-xyz-12345")
	assert.False(t, exists, "non-existent account should return false")
}

func TestDoesNamedAccountExist_WithTempDir(t *testing.T) {
	acctDir := withTempLocation(t)

	// No accounts in fresh dir
	assert.False(t, store.DoesNamedAccountExist("test-account"))

	// Create a directory to simulate an account
	err := os.MkdirAll(path.Join(acctDir, "test-account"), 0700)
	require.NoError(t, err)

	assert.True(t, store.DoesNamedAccountExist("test-account"))
	assert.False(t, store.DoesNamedAccountExist("other-account"))
}

func TestLocalAccounts_Empty(t *testing.T) {
	withTempLocation(t)

	accounts := store.LocalAccounts()
	assert.Empty(t, accounts, "fresh directory should have no accounts")
}

func TestLocalAccounts_WithAccounts(t *testing.T) {
	acctDir := withTempLocation(t)

	// Create account directories
	for _, name := range []string{"alice", "bob"} {
		err := os.MkdirAll(path.Join(acctDir, name), 0700)
		require.NoError(t, err)
	}
	// Create a file (should be ignored - only directories count)
	err := os.WriteFile(path.Join(acctDir, "not-an-account"), []byte("test"), 0600)
	require.NoError(t, err)

	accounts := store.LocalAccounts()
	assert.Len(t, accounts, 2)
	assert.Contains(t, accounts, "alice")
	assert.Contains(t, accounts, "bob")
}

func TestFromAccountName(t *testing.T) {
	t.Run("returns a non-nil KeyStore for any name", func(t *testing.T) {
		withTempLocation(t)

		ks := store.FromAccountName("my-wallet")
		require.NotNil(t, ks, "FromAccountName must return a non-nil KeyStore")

		// A fresh keystore with no key files should have zero accounts.
		accounts := ks.Accounts()
		assert.Empty(t, accounts, "fresh keystore should have no accounts")
	})

	t.Run("returns different keystores for different names", func(t *testing.T) {
		withTempLocation(t)

		ks1 := store.FromAccountName("wallet-a")
		ks2 := store.FromAccountName("wallet-b")

		require.NotNil(t, ks1)
		require.NotNil(t, ks2)

		// They should be distinct instances (different pointers).
		assert.NotSame(t, ks1, ks2,
			"different account names must return different KeyStore instances")
	})
}

func TestSetDefaultLocation(t *testing.T) {
	t.Run("changes DefaultLocation to new path", func(t *testing.T) {
		tmpDir := t.TempDir()
		origConfigDir := c.DefaultConfigDirName
		t.Cleanup(func() {
			c.DefaultConfigDirName = origConfigDir
		})

		store.SetDefaultLocation(tmpDir)

		loc := store.DefaultLocation()
		assert.Contains(t, loc, tmpDir,
			"DefaultLocation must contain the new directory name")
		assert.Contains(t, loc, "account-keys",
			"DefaultLocation must still include account-keys subdirectory")
	})

	t.Run("creates directory structure if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		origConfigDir := c.DefaultConfigDirName
		t.Cleanup(func() {
			c.DefaultConfigDirName = origConfigDir
		})

		newDir := path.Join(tmpDir, "custom-config")
		store.SetDefaultLocation(newDir)

		loc := store.DefaultLocation()
		info, err := os.Stat(loc)
		require.NoError(t, err, "directory must be created by SetDefaultLocation")
		assert.True(t, info.IsDir(), "created path must be a directory")
	})

	t.Run("is idempotent for same directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		origConfigDir := c.DefaultConfigDirName
		t.Cleanup(func() {
			c.DefaultConfigDirName = origConfigDir
		})

		store.SetDefaultLocation(tmpDir)
		loc1 := store.DefaultLocation()

		// Call again with the same directory.
		store.SetDefaultLocation(tmpDir)
		loc2 := store.DefaultLocation()

		assert.Equal(t, loc1, loc2,
			"calling SetDefaultLocation twice with same dir must be idempotent")
	})
}

func TestInitConfigDir(t *testing.T) {
	t.Run("creates the config directory structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		origConfigDir := c.DefaultConfigDirName
		t.Cleanup(func() {
			c.DefaultConfigDirName = origConfigDir
		})

		// Point config to a new subdirectory inside temp.
		c.DefaultConfigDirName = path.Join(tmpDir, "init-test-config")

		store.InitConfigDir()

		// Verify the full path exists.
		expectedPath := path.Join(c.DefaultConfigDirName) // relative to homedir though
		// Since InitConfigDir uses homedir.Dir(), we verify via DefaultLocation.
		// But InitConfigDir creates: homedir/DefaultConfigDirName/account-keys
		// We can't easily predict homedir, so instead we verify that calling
		// InitConfigDir does not panic and subsequent operations work.
		// Use SetDefaultLocation with tmpDir to verify directory creation.
		c.DefaultConfigDirName = origConfigDir
		_ = expectedPath

		// Alternative: set to tmpDir-based path and verify.
		newConfigDir := path.Join(tmpDir, "init-config-dir-test")
		c.DefaultConfigDirName = newConfigDir

		// Remove the directory if it was created by SetDefaultLocation.
		loc := store.DefaultLocation()
		_ = os.RemoveAll(loc)

		// Now call InitConfigDir to create it.
		store.InitConfigDir()

		// InitConfigDir uses homedir.Dir() as base, so the path is
		// homedir/<DefaultConfigDirName>/account-keys.
		// Verify via DefaultLocation.
		info, err := os.Stat(store.DefaultLocation())
		require.NoError(t, err, "InitConfigDir must create the directory structure")
		assert.True(t, info.IsDir())
	})
}

func TestAddressFromAccountName(t *testing.T) {
	t.Run("returns error for empty keystore", func(t *testing.T) {
		acctDir := withTempLocation(t)

		// Create the account directory but leave it empty (no key files).
		err := os.MkdirAll(path.Join(acctDir, "empty-wallet"), 0700)
		require.NoError(t, err)

		addr, err := store.AddressFromAccountName("empty-wallet")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no accounts found in keystore")
		assert.Empty(t, addr)
	})

	t.Run("returns error for nonexistent account directory", func(t *testing.T) {
		withTempLocation(t)

		addr, err := store.AddressFromAccountName("does-not-exist")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no accounts found in keystore")
		assert.Empty(t, addr)
	})
}

func TestFromAddress(t *testing.T) {
	t.Run("returns nil for unknown address", func(t *testing.T) {
		withTempLocation(t)

		// Use a valid-looking TRON address that is not in the keystore.
		ks := store.FromAddress("TJRabPrwbZy45sbavfcjinPJC18kjpRTv8")
		assert.Nil(t, ks, "FromAddress must return nil for unknown address")
	})

	t.Run("returns nil when no accounts exist", func(t *testing.T) {
		withTempLocation(t)

		ks := store.FromAddress("TSomeRandomAddressThatDoesNotExist")
		assert.Nil(t, ks, "FromAddress must return nil when no accounts exist")
	})
}

func TestUnlockedKeystore(t *testing.T) {
	t.Run("returns error for invalid address", func(t *testing.T) {
		withTempLocation(t)

		ks, acct, err := store.UnlockedKeystore("not-a-valid-address", "password")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "address not valid")
		assert.Nil(t, ks)
		assert.Nil(t, acct)
	})

	t.Run("returns error when keystore not found for valid address format", func(t *testing.T) {
		withTempLocation(t)

		// Use a structurally valid TRON address that does not exist in the keystore.
		ks, acct, err := store.UnlockedKeystore("TJRabPrwbZy45sbavfcjinPJC18kjpRTv8", "password")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not open local keystore")
		assert.Nil(t, ks)
		assert.Nil(t, acct)
	})
}

func TestDescribeLocalAccounts(t *testing.T) {
	t.Run("does not panic with empty directory", func(t *testing.T) {
		withTempLocation(t)

		// DescribeLocalAccounts prints to stdout. We just verify it does not panic.
		assert.NotPanics(t, func() {
			store.DescribeLocalAccounts()
		})
	})

	t.Run("does not panic with account directories present", func(t *testing.T) {
		acctDir := withTempLocation(t)

		err := os.MkdirAll(path.Join(acctDir, "test-wallet"), 0700)
		require.NoError(t, err)

		assert.NotPanics(t, func() {
			store.DescribeLocalAccounts()
		})
	})
}

func TestErrNoUnlockBadPassphrase(t *testing.T) {
	// Verify the sentinel error is defined and has a meaningful message.
	assert.NotNil(t, store.ErrNoUnlockBadPassphrase)
	assert.Contains(t, store.ErrNoUnlockBadPassphrase.Error(), "could not unlock account")
}

func TestSetKeystoreFactory(t *testing.T) {
	acctDir := withTempLocation(t)
	t.Cleanup(store.CloseAll)

	// Create an account directory with a key file
	acctPath := path.Join(acctDir, "factory-test")
	err := os.MkdirAll(acctPath, 0700)
	require.NoError(t, err)

	ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
	_, err = ks.NewAccount("pass")
	require.NoError(t, err)
	ks.Close()

	t.Run("custom factory is used", func(t *testing.T) {
		callCount := 0
		store.SetKeystoreFactory(func(p string) *keystore.KeyStore {
			callCount++
			return keystore.NewKeyStore(p, keystore.LightScryptN, keystore.LightScryptP)
		})
		t.Cleanup(store.CloseAll)

		loaded := store.FromAccountName("factory-test")
		require.NotNil(t, loaded)
		assert.Equal(t, 1, callCount, "custom factory should have been called once")
		loaded.Close()
	})
}

func TestCloseAll(t *testing.T) {
	t.Run("safe to call with no open keystores", func(t *testing.T) {
		withTempLocation(t)
		assert.NotPanics(t, store.CloseAll)
	})

	t.Run("safe to call multiple times", func(t *testing.T) {
		withTempLocation(t)
		store.CloseAll()
		assert.NotPanics(t, store.CloseAll)
	})

	t.Run("closes tracked keystores", func(t *testing.T) {
		acctDir := withTempLocation(t)

		acctPath := path.Join(acctDir, "close-test")
		err := os.MkdirAll(acctPath, 0700)
		require.NoError(t, err)

		// Open a keystore via FromAccountName (which tracks it)
		_ = store.FromAccountName("close-test")

		// CloseAll should not panic
		assert.NotPanics(t, store.CloseAll)
	})

	t.Run("resets factory to default", func(t *testing.T) {
		withTempLocation(t)

		callCount := 0
		store.SetKeystoreFactory(func(p string) *keystore.KeyStore {
			callCount++
			return keystore.NewKeyStore(p, keystore.LightScryptN, keystore.LightScryptP)
		})

		store.CloseAll() // Should reset factory

		// After CloseAll, further FromAccountName calls should use default factory
		before := callCount
		loaded := store.FromAccountName("any-name")
		require.NotNil(t, loaded)
		loaded.Close()
		store.CloseAll()

		// Custom factory should NOT have been called after CloseAll reset
		assert.Equal(t, before, callCount, "custom factory should not be called after CloseAll resets it")
	})
}

func TestUnlockedKeystore_WithRealKey(t *testing.T) {
	acctDir := withTempLocation(t)
	t.Cleanup(store.CloseAll)

	// Create account with real key
	acctPath := path.Join(acctDir, "unlock-test")
	err := os.MkdirAll(acctPath, 0700)
	require.NoError(t, err)

	ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("correct-pass")
	require.NoError(t, err)
	addrStr := acct.Address.String()
	ks.Close()

	// Yield to let the keystore's watcher goroutine finish after Close().
	runtime.Gosched()

	t.Run("wrong passphrase returns ErrNoUnlockBadPassphrase", func(t *testing.T) {
		resultKs, resultAcct, err := store.UnlockedKeystore(addrStr, "wrong-pass")
		require.Error(t, err)
		assert.ErrorIs(t, err, store.ErrNoUnlockBadPassphrase)
		assert.Nil(t, resultKs)
		assert.Nil(t, resultAcct)
		store.CloseAll()
	})

	t.Run("correct passphrase returns keystore and account", func(t *testing.T) {
		resultKs, resultAcct, err := store.UnlockedKeystore(addrStr, "correct-pass")
		require.NoError(t, err)
		require.NotNil(t, resultKs)
		require.NotNil(t, resultAcct)
		assert.Equal(t, addrStr, resultAcct.Address.String())
		resultKs.Close()
		store.CloseAll()
	})
}

func TestAddressFromAccountName_WithRealKey(t *testing.T) {
	acctDir := withTempLocation(t)
	t.Cleanup(store.CloseAll)

	acctPath := path.Join(acctDir, "addr-test")
	err := os.MkdirAll(acctPath, 0700)
	require.NoError(t, err)

	ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("pass")
	require.NoError(t, err)
	expectedAddr := acct.Address.String()
	ks.Close()

	addr, err := store.AddressFromAccountName("addr-test")
	require.NoError(t, err)
	assert.Equal(t, expectedAddr, addr)
	store.CloseAll()
}

func TestFromAddress_WithRealKey(t *testing.T) {
	acctDir := withTempLocation(t)
	t.Cleanup(store.CloseAll)

	acctPath := path.Join(acctDir, "find-test")
	err := os.MkdirAll(acctPath, 0700)
	require.NoError(t, err)

	ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("pass")
	require.NoError(t, err)
	addrStr := acct.Address.String()
	ks.Close()

	found := store.FromAddress(addrStr)
	require.NotNil(t, found, "FromAddress should find the account")
	found.Close()
	store.CloseAll()
}

func TestFromAddress_NoGoroutineLeak(t *testing.T) {
	acctDir := withTempLocation(t)

	// Create multiple account directories with real key files
	names := []string{"wallet-a", "wallet-b", "wallet-c"}
	var targetAddr string
	for i, name := range names {
		acctPath := path.Join(acctDir, name)
		err := os.MkdirAll(acctPath, 0700)
		require.NoError(t, err)

		ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)
		// Use the last account as target so all others must be iterated and closed
		if i == len(names)-1 {
			targetAddr = acc.Address.String()
		}
		ks.Close()
	}

	// Let any background goroutines settle
	time.Sleep(100 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	// Call FromAddress multiple times — non-matching keystores should be closed
	const iterations = 3
	for i := 0; i < iterations; i++ {
		ks := store.FromAddress(targetAddr)
		require.NotNil(t, ks, "FromAddress must find the target address")
		ks.Close()
	}

	// Also test with an address that doesn't exist (all keystores should be closed)
	ks := store.FromAddress("TJRabPrwbZy45sbavfcjinPJC18kjpRTv8")
	assert.Nil(t, ks)

	// Allow goroutines to wind down
	time.Sleep(100 * time.Millisecond)

	after := runtime.NumGoroutine()
	assert.LessOrEqual(t, after, baseline+2,
		"goroutine count should return near baseline after FromAddress calls (baseline=%d, after=%d)", baseline, after)
}

// --- Tests for Store struct methods ---

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	s := store.NewStore(tmpDir)
	require.NotNil(t, s)

	loc := s.DefaultLocation()
	assert.Contains(t, loc, tmpDir)
	assert.Contains(t, loc, "account-keys")
}

func TestDefaultStoreInstance(t *testing.T) {
	s := store.DefaultStoreInstance()
	require.NotNil(t, s)

	loc := s.DefaultLocation()
	assert.NotEmpty(t, loc)
	assert.Contains(t, loc, "account-keys")
}

func TestStore_InitConfigDir(t *testing.T) {
	s := newTempStore(t)
	loc := s.DefaultLocation()

	info, err := os.Stat(loc)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestStore_LocalAccounts(t *testing.T) {
	s := newTempStore(t)
	loc := s.DefaultLocation()

	assert.Empty(t, s.LocalAccounts())

	for _, name := range []string{"alice", "bob"} {
		require.NoError(t, os.MkdirAll(path.Join(loc, name), 0700))
	}

	accounts := s.LocalAccounts()
	assert.Len(t, accounts, 2)
	assert.Contains(t, accounts, "alice")
	assert.Contains(t, accounts, "bob")
}

func TestStore_DoesNamedAccountExist(t *testing.T) {
	s := newTempStore(t)
	loc := s.DefaultLocation()

	assert.False(t, s.DoesNamedAccountExist("test"))

	require.NoError(t, os.MkdirAll(path.Join(loc, "test"), 0700))
	assert.True(t, s.DoesNamedAccountExist("test"))
}

func TestStore_FromAccountName(t *testing.T) {
	s := newTempStore(t)

	ks := s.FromAccountName("my-wallet")
	require.NotNil(t, ks)
	assert.Empty(t, ks.Accounts())
	ks.Close()
}

func TestStore_SetDefaultLocation(t *testing.T) {
	s := newTempStore(t)

	newDir := t.TempDir()
	s.SetDefaultLocation(newDir)

	loc := s.DefaultLocation()
	assert.Contains(t, loc, newDir)
	assert.Contains(t, loc, "account-keys")

	info, err := os.Stat(loc)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestStore_SetKeystoreFactory(t *testing.T) {
	s := newTempStore(t)
	loc := s.DefaultLocation()

	acctPath := path.Join(loc, "factory-acct")
	require.NoError(t, os.MkdirAll(acctPath, 0700))

	ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
	_, err := ks.NewAccount("pass")
	require.NoError(t, err)
	ks.Close()

	callCount := 0
	s.SetKeystoreFactory(func(p string) *keystore.KeyStore {
		callCount++
		return keystore.NewKeyStore(p, keystore.LightScryptN, keystore.LightScryptP)
	})

	loaded := s.FromAccountName("factory-acct")
	require.NotNil(t, loaded)
	assert.Equal(t, 1, callCount)
	loaded.Close()
}

func TestStore_CloseAll(t *testing.T) {
	s := newTempStore(t)

	// Safe to call with nothing open
	assert.NotPanics(t, s.CloseAll)

	// Open some and close
	_ = s.FromAccountName("a")
	_ = s.FromAccountName("b")
	assert.NotPanics(t, s.CloseAll)
}

func TestStore_DescribeLocalAccounts(t *testing.T) {
	s := newTempStore(t)

	assert.NotPanics(t, func() {
		s.DescribeLocalAccounts()
	})
}

func TestStore_AddressFromAccountName(t *testing.T) {
	s := newTempStore(t)
	loc := s.DefaultLocation()

	// Empty keystore returns error
	require.NoError(t, os.MkdirAll(path.Join(loc, "empty"), 0700))
	_, err := s.AddressFromAccountName("empty")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no accounts found in keystore")

	// With a real key
	acctPath := path.Join(loc, "real")
	require.NoError(t, os.MkdirAll(acctPath, 0700))

	ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("pass")
	require.NoError(t, err)
	expected := acct.Address.String()
	ks.Close()

	addr, err := s.AddressFromAccountName("real")
	require.NoError(t, err)
	assert.Equal(t, expected, addr)
}

func TestStore_FromAddress(t *testing.T) {
	s := newTempStore(t)
	loc := s.DefaultLocation()

	// Not found
	assert.Nil(t, s.FromAddress("TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"))

	// Create account and find it
	acctPath := path.Join(loc, "lookup")
	require.NoError(t, os.MkdirAll(acctPath, 0700))

	ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("pass")
	require.NoError(t, err)
	addrStr := acct.Address.String()
	ks.Close()

	found := s.FromAddress(addrStr)
	require.NotNil(t, found)
	found.Close()
}

func TestStore_UnlockedKeystore(t *testing.T) {
	s := newTempStore(t)
	loc := s.DefaultLocation()

	// Invalid address
	_, _, err := s.UnlockedKeystore("bad-addr", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "address not valid")

	// Create account
	acctPath := path.Join(loc, "unlock")
	require.NoError(t, os.MkdirAll(acctPath, 0700))

	ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("correct")
	require.NoError(t, err)
	addrStr := acct.Address.String()
	ks.Close()

	runtime.Gosched()

	// Wrong passphrase
	rKs, rAcct, err := s.UnlockedKeystore(addrStr, "wrong")
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrNoUnlockBadPassphrase)
	assert.Nil(t, rKs)
	assert.Nil(t, rAcct)
	s.CloseAll()

	// Correct passphrase
	rKs, rAcct, err = s.UnlockedKeystore(addrStr, "correct")
	require.NoError(t, err)
	require.NotNil(t, rKs)
	require.NotNil(t, rAcct)
	assert.Equal(t, addrStr, rAcct.Address.String())
	rKs.Close()
}
