package store_test

import (
	"os"
	"path"
	"testing"

	c "github.com/fbsobreira/gotron-sdk/pkg/common"
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
		assert.Contains(t, err.Error(), "keystore not found")
		assert.Empty(t, addr)
	})

	t.Run("returns error for nonexistent account directory", func(t *testing.T) {
		withTempLocation(t)

		addr, err := store.AddressFromAccountName("does-not-exist")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "keystore not found")
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
