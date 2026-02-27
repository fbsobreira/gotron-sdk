package account_test

import (
	"os"
	"path"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestStore redirects the account store to a temporary directory
// and returns a cleanup function that restores the original config.
func setupTestStore(t *testing.T) {
	t.Helper()
	origConfigDir := common.DefaultConfigDirName
	tmpDir := t.TempDir()
	store.SetDefaultLocation(tmpDir)
	t.Cleanup(func() {
		common.DefaultConfigDirName = origConfigDir
	})
}

// testPrivateKey is a well-known ECDSA private key used only for testing.
// It corresponds to TRON address TJTm4FRMmQZSkMjDEeiBxPXYLmZRBEiB9G (or similar).
const testPrivateKey = "e9a6e2a4e8e050b8616870520abc3c61f0368a4aaee3c3f0e742cf29e4cc8501"

func TestImportFromPrivateKey(t *testing.T) {
	tests := []struct {
		name       string
		privateKey string
		acctName   string
		passphrase string
		wantErr    bool
		errContain string
	}{
		{
			name:       "valid key with explicit name",
			privateKey: testPrivateKey,
			acctName:   "test-account",
			passphrase: "test-pass",
		},
		{
			name:       "valid key with 0x prefix",
			privateKey: "0x" + testPrivateKey,
			acctName:   "test-0x-account",
			passphrase: "test-pass",
		},
		{
			name:       "valid key with empty name generates name",
			privateKey: testPrivateKey,
			acctName:   "",
			passphrase: "test-pass",
		},
		{
			name:       "invalid hex key",
			privateKey: "zzzz-not-hex",
			acctName:   "bad-key-account",
			passphrase: "test-pass",
			wantErr:    true,
		},
		{
			name:       "empty private key",
			privateKey: "",
			acctName:   "empty-key-account",
			passphrase: "test-pass",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestStore(t)

			resultName, err := account.ImportFromPrivateKey(tt.privateKey, tt.acctName, tt.passphrase)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}

			require.NoError(t, err)
			if tt.acctName != "" {
				assert.Equal(t, tt.acctName, resultName)
			} else {
				assert.NotEmpty(t, resultName, "generated name should not be empty")
				assert.Contains(t, resultName, "-imported", "auto-generated name should have -imported suffix")
			}

			// Verify account exists in the store
			assert.True(t, store.DoesNamedAccountExist(resultName),
				"imported account should exist in the store")
		})
	}
}

func TestImportFromPrivateKey_DuplicateName(t *testing.T) {
	setupTestStore(t)

	name := "duplicate-account"
	_, err := account.ImportFromPrivateKey(testPrivateKey, name, "pass")
	require.NoError(t, err, "first import should succeed")

	// Second import with same name should fail
	_, err = account.ImportFromPrivateKey(testPrivateKey, name, "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestImportFromPrivateKey_EmptyPassphrase(t *testing.T) {
	setupTestStore(t)

	name, err := account.ImportFromPrivateKey(testPrivateKey, "no-pass-account", "")
	require.NoError(t, err)
	assert.Equal(t, "no-pass-account", name)
	assert.True(t, store.DoesNamedAccountExist(name))
}

func TestRemoveAccount(t *testing.T) {
	tests := []struct {
		name       string
		acctName   string
		setup      func(t *testing.T) // run before removal
		wantErr    bool
		errContain string
	}{
		{
			name:     "remove existing account",
			acctName: "to-be-removed",
			setup: func(t *testing.T) {
				t.Helper()
				_, err := account.ImportFromPrivateKey(testPrivateKey, "to-be-removed", "pass")
				require.NoError(t, err)
			},
		},
		{
			name:       "remove non-existent account",
			acctName:   "does-not-exist",
			setup:      func(t *testing.T) {},
			wantErr:    true,
			errContain: "doesn't exist",
		},
		{
			name:       "remove with empty name",
			acctName:   "",
			setup:      func(t *testing.T) {},
			wantErr:    true,
			errContain: "doesn't exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestStore(t)
			tt.setup(t)

			err := account.RemoveAccount(tt.acctName)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				return
			}

			require.NoError(t, err)
			// Verify account no longer exists
			assert.False(t, store.DoesNamedAccountExist(tt.acctName),
				"removed account should no longer exist in the store")
		})
	}
}

func TestImportThenRemove(t *testing.T) {
	setupTestStore(t)

	name := "import-then-remove"
	_, err := account.ImportFromPrivateKey(testPrivateKey, name, "pass")
	require.NoError(t, err)
	assert.True(t, store.DoesNamedAccountExist(name))

	err = account.RemoveAccount(name)
	require.NoError(t, err)
	assert.False(t, store.DoesNamedAccountExist(name))
}

func TestCreateNewLocalAccount(t *testing.T) {
	tests := []struct {
		name     string
		creation *account.Creation
		wantErr  bool
	}{
		{
			name: "create with generated mnemonic",
			creation: &account.Creation{
				Name:       "new-local-account",
				Passphrase: "strong-pass",
			},
		},
		{
			name: "create with explicit mnemonic",
			creation: &account.Creation{
				Name:       "mnemonic-account",
				Passphrase: "strong-pass",
				Mnemonic:   "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
			},
		},
		{
			name: "create with empty passphrase",
			creation: &account.Creation{
				Name:       "empty-pass-account",
				Passphrase: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestStore(t)

			err := account.CreateNewLocalAccount(tt.creation)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.True(t, store.DoesNamedAccountExist(tt.creation.Name),
				"created account should exist in the store")
		})
	}
}

func TestCreateNewLocalAccount_Duplicate(t *testing.T) {
	setupTestStore(t)

	creation := &account.Creation{
		Name:       "dup-local",
		Passphrase: "pass",
		Mnemonic:   "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
	}

	err := account.CreateNewLocalAccount(creation)
	require.NoError(t, err)

	// Creating again with the same mnemonic should fail because the same key/address
	// is imported a second time into the same keystore.
	err = account.CreateNewLocalAccount(creation)
	require.Error(t, err)
}

func TestNew(t *testing.T) {
	result := account.New()
	assert.Equal(t, "New Account", result)
}

func TestIsValidPassphrase(t *testing.T) {
	tests := []struct {
		name string
		pass string
		want bool
	}{
		{name: "non-empty passphrase", pass: "strong-password-123!", want: true},
		{name: "empty passphrase", pass: "", want: true},
		{name: "short passphrase", pass: "a", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, account.IsValidPassphrase(tt.pass))
		})
	}
}

func TestImportKeyStore(t *testing.T) {
	setupTestStore(t)

	// First, import a private key to create a keystore file we can export/re-import
	acctName := "keystore-source"
	_, err := account.ImportFromPrivateKey(testPrivateKey, acctName, "pass")
	require.NoError(t, err)

	// Get the address from the account
	addr, err := store.AddressFromAccountName(acctName)
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	// Export the keystore to a temp directory
	exportDir := t.TempDir()
	outFile, err := account.ExportKeystore(addr, exportDir, "pass")
	require.NoError(t, err)
	require.NotEmpty(t, outFile)

	// Verify the exported file exists
	_, statErr := os.Stat(outFile)
	require.NoError(t, statErr, "exported keystore file should exist")

	// Remove the original account so the address is no longer in the store
	// (ImportKeyStore rejects duplicate addresses)
	err = account.RemoveAccount(acctName)
	require.NoError(t, err)

	// Now import the keystore file with a new name
	importedName, err := account.ImportKeyStore(outFile, "re-imported", "pass")
	require.NoError(t, err)
	assert.Equal(t, "re-imported", importedName)
	assert.True(t, store.DoesNamedAccountExist("re-imported"))
}

func TestImportKeyStore_InvalidPath(t *testing.T) {
	setupTestStore(t)

	_, err := account.ImportKeyStore("/nonexistent/path/keystore.json", "some-name", "pass")
	require.Error(t, err)
}

func TestImportKeyStore_DuplicateName(t *testing.T) {
	setupTestStore(t)

	// Create a fake keystore file (content doesn't matter for name check)
	_, err := account.ImportFromPrivateKey(testPrivateKey, "taken-name", "pass")
	require.NoError(t, err)

	// Create a dummy file to use as keystore path
	tmpFile := path.Join(t.TempDir(), "dummy.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte("{}"), 0600))

	_, err = account.ImportKeyStore(tmpFile, "taken-name", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestExportKeystore(t *testing.T) {
	setupTestStore(t)

	// Import an account first
	_, err := account.ImportFromPrivateKey(testPrivateKey, "export-test", "pass")
	require.NoError(t, err)

	addr, err := store.AddressFromAccountName("export-test")
	require.NoError(t, err)

	exportDir := t.TempDir()
	outFile, err := account.ExportKeystore(addr, exportDir, "pass")
	require.NoError(t, err)
	assert.NotEmpty(t, outFile)
	assert.Contains(t, outFile, addr)

	// File should exist and contain data
	data, readErr := os.ReadFile(outFile)
	require.NoError(t, readErr)
	assert.NotEmpty(t, data)
}

func TestExportPrivateKey(t *testing.T) {
	setupTestStore(t)

	_, err := account.ImportFromPrivateKey(testPrivateKey, "export-pk-test", "pass")
	require.NoError(t, err)

	addr, err := store.AddressFromAccountName("export-pk-test")
	require.NoError(t, err)

	// ExportPrivateKey prints to stdout; we just verify it returns no error
	// with correct passphrase
	err = account.ExportPrivateKey(addr, "pass")
	require.NoError(t, err)
}

func TestExportPrivateKey_WrongPassphrase(t *testing.T) {
	setupTestStore(t)

	_, err := account.ImportFromPrivateKey(testPrivateKey, "export-pk-wrong", "correct-pass")
	require.NoError(t, err)

	addr, err := store.AddressFromAccountName("export-pk-wrong")
	require.NoError(t, err)

	err = account.ExportPrivateKey(addr, "wrong-pass")
	require.Error(t, err, "exporting with wrong passphrase should fail")
}

func TestExportPrivateKey_UnknownAddress(t *testing.T) {
	setupTestStore(t)

	err := account.ExportPrivateKey("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "keystore not found")
}

func TestImportKeyStore_AutoGeneratedName(t *testing.T) {
	setupTestStore(t)

	// Create and export a keystore file
	_, err := account.ImportFromPrivateKey(testPrivateKey, "auto-gen-source", "pass")
	require.NoError(t, err)

	addr, err := store.AddressFromAccountName("auto-gen-source")
	require.NoError(t, err)

	exportDir := t.TempDir()
	outFile, err := account.ExportKeystore(addr, exportDir, "pass")
	require.NoError(t, err)

	// Remove original so address is available
	err = account.RemoveAccount("auto-gen-source")
	require.NoError(t, err)

	// Import with empty name to trigger auto-generation
	importedName, err := account.ImportKeyStore(outFile, "", "pass")
	require.NoError(t, err)
	assert.NotEmpty(t, importedName)
	assert.Contains(t, importedName, "-imported", "auto-generated name should have -imported suffix")
	assert.True(t, store.DoesNamedAccountExist(importedName))
}

func TestImportKeyStore_WrongPassphrase(t *testing.T) {
	setupTestStore(t)

	// Create and export a keystore
	_, err := account.ImportFromPrivateKey(testPrivateKey, "wrong-pass-source", "correct")
	require.NoError(t, err)

	addr, err := store.AddressFromAccountName("wrong-pass-source")
	require.NoError(t, err)

	exportDir := t.TempDir()
	outFile, err := account.ExportKeystore(addr, exportDir, "correct")
	require.NoError(t, err)

	// Remove original
	err = account.RemoveAccount("wrong-pass-source")
	require.NoError(t, err)

	// Try to import with wrong passphrase
	_, err = account.ImportKeyStore(outFile, "wrong-pass-acct", "wrong-passphrase")
	require.Error(t, err, "importing with wrong passphrase should fail")
}

func TestImportKeyStore_DuplicateAddress(t *testing.T) {
	setupTestStore(t)

	// Import account (creates keystore with address)
	_, err := account.ImportFromPrivateKey(testPrivateKey, "addr-dup-source", "pass")
	require.NoError(t, err)

	addr, err := store.AddressFromAccountName("addr-dup-source")
	require.NoError(t, err)

	// Export the keystore
	exportDir := t.TempDir()
	outFile, err := account.ExportKeystore(addr, exportDir, "pass")
	require.NoError(t, err)

	// Do NOT remove the original -- address should still exist in store
	// Import with a different name but same underlying address should fail
	_, err = account.ImportKeyStore(outFile, "different-name", "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists in keystore")
}

func TestExportKeystore_WrongPassphrase(t *testing.T) {
	setupTestStore(t)

	_, err := account.ImportFromPrivateKey(testPrivateKey, "export-wrong-pass", "correct")
	require.NoError(t, err)

	addr, err := store.AddressFromAccountName("export-wrong-pass")
	require.NoError(t, err)

	exportDir := t.TempDir()
	_, err = account.ExportKeystore(addr, exportDir, "wrong")
	require.Error(t, err, "exporting with wrong passphrase should fail")
}

func TestExportKeystore_UnknownAddress(t *testing.T) {
	setupTestStore(t)

	_, err := account.ExportKeystore("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", t.TempDir(), "pass")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "keystore not found")
}

func TestAccountModelTypes(t *testing.T) {
	// Verify the Account struct can be instantiated with expected fields
	acct := account.Account{
		Address:   "TJTm4FRMmQZSkMjDEeiBxPXYLmZRBEiB9G",
		Type:      "Normal",
		Name:      "test",
		Balance:   1000000,
		Allowance: 0,
		Assets:    map[string]int64{"TRX": 1000000},
		Votes:     map[string]int64{"TSomeWitness": 100},
	}

	assert.Equal(t, "TJTm4FRMmQZSkMjDEeiBxPXYLmZRBEiB9G", acct.Address)
	assert.Equal(t, int64(1000000), acct.Balance)
	assert.Equal(t, int64(1000000), acct.Assets["TRX"])
	assert.Equal(t, int64(100), acct.Votes["TSomeWitness"])

	// Verify FrozenResource and UnfrozenResource can be created
	frozen := account.FrozenResource{
		Amount:     500000,
		DelegateTo: "TDelegateAddress",
		Expire:     1700000000,
	}
	assert.Equal(t, int64(500000), frozen.Amount)

	unfrozen := account.UnfrozenResource{
		Amount: 300000,
		Expire: 1700000000,
	}
	assert.Equal(t, int64(300000), unfrozen.Amount)
}
