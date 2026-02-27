package keystore_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestKeyStore creates a KeyStore backed by a temporary directory using light
// scrypt parameters so tests run quickly.
func newTestKeyStore(t *testing.T) *keystore.KeyStore {
	t.Helper()
	dir := t.TempDir()
	return keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
}

func randomHash(t *testing.T) []byte {
	t.Helper()
	hash := make([]byte, 32)
	_, err := rand.Read(hash)
	require.NoError(t, err)
	return hash
}

func eventuallySignHashFails(t *testing.T, ks *keystore.KeyStore, acc keystore.Account, hash []byte, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := ks.SignHash(acc, hash); err != nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, err := ks.SignHash(acc, hash)
	require.Error(t, err, "expected signing to fail before timeout")
}

// ---------- NewKeyStore ----------

func TestNewKeyStore(t *testing.T) {
	tests := []struct {
		name    string
		scryptN int
		scryptP int
	}{
		{
			name:    "light scrypt parameters",
			scryptN: keystore.LightScryptN,
			scryptP: keystore.LightScryptP,
		},
		{
			name:    "custom scrypt parameters",
			scryptN: 1 << 10,
			scryptP: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			ks := keystore.NewKeyStore(dir, tt.scryptN, tt.scryptP)

			require.NotNil(t, ks)
			assert.Empty(t, ks.Accounts(), "fresh keystore should have no accounts")
		})
	}
}

// ---------- NewAccount ----------

func TestNewAccount(t *testing.T) {
	tests := []struct {
		name       string
		passphrase string
	}{
		{name: "empty passphrase", passphrase: ""},
		{name: "simple passphrase", passphrase: "test-password"},
		{name: "unicode passphrase", passphrase: "p@ssw0rd-with-symbols!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ks := newTestKeyStore(t)

			acc, err := ks.NewAccount(tt.passphrase)
			require.NoError(t, err)
			assert.NotEmpty(t, acc.Address, "account address must not be empty")
			assert.NotEmpty(t, acc.URL.Path, "account URL path must not be empty")
			assert.Equal(t, keystore.KeyStoreScheme, acc.URL.Scheme)

			// Account must appear in the Accounts() list.
			accounts := ks.Accounts()
			require.Len(t, accounts, 1)
			assert.True(t, bytes.Equal(accounts[0].Address, acc.Address))
		})
	}
}

func TestNewAccount_multiple(t *testing.T) {
	ks := newTestKeyStore(t)

	const numAccounts = 3
	created := make([]keystore.Account, 0, numAccounts)
	for i := 0; i < numAccounts; i++ {
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)
		created = append(created, acc)
	}

	accounts := ks.Accounts()
	assert.Len(t, accounts, numAccounts, "all created accounts must be listed")

	// Each created account must be findable.
	for _, c := range created {
		assert.True(t, ks.HasAddress(c.Address), "HasAddress must return true for created account")
	}
}

// ---------- Delete ----------

func TestDelete(t *testing.T) {
	t.Run("delete existing account with correct passphrase", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("secret")
		require.NoError(t, err)

		err = ks.Delete(acc, "secret")
		require.NoError(t, err)

		assert.Empty(t, ks.Accounts(), "accounts list should be empty after deletion")
		assert.False(t, ks.HasAddress(acc.Address), "HasAddress must return false after deletion")
	})

	t.Run("delete with wrong passphrase fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("correct")
		require.NoError(t, err)

		err = ks.Delete(acc, "wrong")
		assert.Error(t, err, "delete with wrong passphrase must fail")

		// Account must still exist.
		assert.Len(t, ks.Accounts(), 1, "account must remain after failed delete")
	})

	t.Run("delete non-existent account fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		fakeAccount := keystore.Account{
			Address: []byte("nonexistent-address-placeholder!"),
		}

		err := ks.Delete(fakeAccount, "anything")
		assert.Error(t, err)
	})
}

// ---------- Update ----------

func TestUpdate(t *testing.T) {
	t.Run("change passphrase successfully", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("old-pass")
		require.NoError(t, err)

		err = ks.Update(acc, "old-pass", "new-pass")
		require.NoError(t, err)

		// Old passphrase must no longer work for unlock.
		err = ks.Unlock(acc, "old-pass")
		assert.Error(t, err, "old passphrase must not work after update")

		// New passphrase must work.
		err = ks.Unlock(acc, "new-pass")
		assert.NoError(t, err, "new passphrase must work after update")
	})

	t.Run("update with wrong old passphrase fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("correct")
		require.NoError(t, err)

		err = ks.Update(acc, "wrong", "new")
		assert.Error(t, err, "update with wrong old passphrase must fail")
	})
}

// ---------- SignHash ----------

func TestSignHash(t *testing.T) {
	t.Run("sign when unlocked produces valid recoverable signature", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		err = ks.Unlock(acc, "pass")
		require.NoError(t, err)

		hash := randomHash(t)

		sig, err := ks.SignHash(acc, hash)
		require.NoError(t, err)
		assert.Len(t, sig, 65, "ECDSA signature must be 65 bytes [R || S || V]")

		pub, err := crypto.SigToPub(hash, sig)
		require.NoError(t, err)
		recovered := address.PubkeyToAddress(*pub)
		assert.Equal(t, acc.Address, recovered)
	})

	t.Run("sign when locked returns ErrLocked", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		// Account is not unlocked.
		hash := randomHash(t)

		_, err = ks.SignHash(acc, hash)
		assert.Error(t, err, "signing when locked must fail")
	})

	t.Run("sign rejects hash sizes other than 32 bytes", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)
		require.NoError(t, ks.Unlock(acc, "pass"))

		_, err = ks.SignHash(acc, make([]byte, 31))
		require.Error(t, err)

		_, err = ks.SignHash(acc, make([]byte, 33))
		require.Error(t, err)
	})
}

// ---------- SignHashWithPassphrase ----------

func TestSignHashWithPassphrase(t *testing.T) {
	t.Run("sign with correct passphrase", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("my-secret")
		require.NoError(t, err)

		hash := randomHash(t)

		sig, err := ks.SignHashWithPassphrase(acc, "my-secret", hash)
		require.NoError(t, err)
		assert.Len(t, sig, 65, "ECDSA signature must be 65 bytes")

		pub, err := crypto.SigToPub(hash, sig)
		require.NoError(t, err)
		recovered := address.PubkeyToAddress(*pub)
		assert.Equal(t, acc.Address, recovered)
	})

	t.Run("sign with wrong passphrase fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("correct")
		require.NoError(t, err)

		hash := randomHash(t)

		_, err = ks.SignHashWithPassphrase(acc, "wrong", hash)
		assert.Error(t, err, "wrong passphrase must fail")
	})

	t.Run("deterministic signature for same hash", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		hash := randomHash(t)

		sig1, err := ks.SignHashWithPassphrase(acc, "pass", hash)
		require.NoError(t, err)

		sig2, err := ks.SignHashWithPassphrase(acc, "pass", hash)
		require.NoError(t, err)

		assert.Equal(t, sig1, sig2, "same hash with same key must produce identical signature")
	})

	t.Run("sign with passphrase rejects hash sizes other than 32 bytes", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		_, err = ks.SignHashWithPassphrase(acc, "pass", make([]byte, 31))
		require.Error(t, err)

		_, err = ks.SignHashWithPassphrase(acc, "pass", make([]byte, 33))
		require.Error(t, err)
	})
}

// ---------- Unlock / Lock ----------

func TestUnlockLock(t *testing.T) {
	t.Run("unlock then sign then lock then sign fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		// Unlock.
		err = ks.Unlock(acc, "pass")
		require.NoError(t, err)

		// Sign should succeed while unlocked.
		hash := make([]byte, 32)
		_, err = rand.Read(hash)
		require.NoError(t, err)

		sig, err := ks.SignHash(acc, hash)
		require.NoError(t, err)
		assert.Len(t, sig, 65)

		// Lock.
		err = ks.Lock(acc.Address)
		require.NoError(t, err)

		// Sign should fail after locking.
		_, err = ks.SignHash(acc, hash)
		assert.Error(t, err, "signing after lock must fail")
	})

	t.Run("unlock with wrong passphrase fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("correct")
		require.NoError(t, err)

		err = ks.Unlock(acc, "wrong")
		assert.Error(t, err, "unlock with wrong passphrase must fail")
	})

	t.Run("lock without prior unlock is a no-op", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		// Locking a never-unlocked account should not error.
		err = ks.Lock(acc.Address)
		assert.NoError(t, err)
	})

	t.Run("double unlock is idempotent", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		err = ks.Unlock(acc, "pass")
		require.NoError(t, err)

		err = ks.Unlock(acc, "pass")
		require.NoError(t, err)

		// Should still be able to sign.
		hash := make([]byte, 32)
		_, err = rand.Read(hash)
		require.NoError(t, err)

		sig, err := ks.SignHash(acc, hash)
		require.NoError(t, err)
		assert.Len(t, sig, 65)
	})
}

// ---------- TimedUnlock ----------

func TestTimedUnlock(t *testing.T) {
	t.Run("timed unlock with wrong passphrase fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		err = ks.TimedUnlock(acc, "wrong", 0)
		assert.Error(t, err)
	})

	t.Run("timed unlock expires and relocks", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		hash := randomHash(t)
		require.NoError(t, ks.TimedUnlock(acc, "pass", 60*time.Millisecond))

		_, err = ks.SignHash(acc, hash)
		require.NoError(t, err, "account should sign while unlock is active")

		eventuallySignHashFails(t, ks, acc, hash, 600*time.Millisecond)
	})

	t.Run("timed unlock can extend previous timeout", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		hash := randomHash(t)
		require.NoError(t, ks.TimedUnlock(acc, "pass", 80*time.Millisecond))
		time.Sleep(30 * time.Millisecond)
		require.NoError(t, ks.TimedUnlock(acc, "pass", 200*time.Millisecond))

		time.Sleep(90 * time.Millisecond) // past original timeout, within extended window
		_, err = ks.SignHash(acc, hash)
		require.NoError(t, err)

		eventuallySignHashFails(t, ks, acc, hash, 800*time.Millisecond)
	})

	t.Run("timed unlock can shorten previous timeout", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		hash := randomHash(t)
		require.NoError(t, ks.TimedUnlock(acc, "pass", 500*time.Millisecond))
		time.Sleep(30 * time.Millisecond)
		require.NoError(t, ks.TimedUnlock(acc, "pass", 40*time.Millisecond))

		eventuallySignHashFails(t, ks, acc, hash, 600*time.Millisecond)
	})

	t.Run("indefinite unlock is not altered by timed unlock", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		hash := randomHash(t)
		require.NoError(t, ks.Unlock(acc, "pass"))
		require.NoError(t, ks.TimedUnlock(acc, "pass", 40*time.Millisecond))

		time.Sleep(120 * time.Millisecond)
		_, err = ks.SignHash(acc, hash)
		require.NoError(t, err, "account should remain unlocked because initial unlock was indefinite")
	})
}

// ---------- Find ----------

func TestFind(t *testing.T) {
	t.Run("find existing account by address", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		found, err := ks.Find(keystore.Account{Address: acc.Address})
		require.NoError(t, err)
		assert.True(t, bytes.Equal(found.Address, acc.Address))
		assert.Equal(t, acc.URL, found.URL)
	})

	t.Run("find non-existing account returns error", func(t *testing.T) {
		ks := newTestKeyStore(t)

		_, err := ks.Find(keystore.Account{
			Address: []byte("this-address-does-not-exist!!!!"),
		})
		assert.Error(t, err)
	})

	t.Run("find by URL path", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		found, err := ks.Find(keystore.Account{
			Address: acc.Address,
			URL:     acc.URL,
		})
		require.NoError(t, err)
		assert.True(t, bytes.Equal(found.Address, acc.Address))
	})
}

// ---------- HasAddress ----------

func TestHasAddress(t *testing.T) {
	t.Run("returns true for existing address", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		assert.True(t, ks.HasAddress(acc.Address))
	})

	t.Run("returns false for unknown address", func(t *testing.T) {
		ks := newTestKeyStore(t)

		assert.False(t, ks.HasAddress([]byte("unknown-address-bytes!!")))
	})
}

// ---------- Export / Import ----------

func TestExportImport(t *testing.T) {
	t.Run("round-trip export then import", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("original")
		require.NoError(t, err)

		// Export with a new passphrase.
		exported, err := ks.Export(acc, "original", "export-pass")
		require.NoError(t, err)
		assert.NotEmpty(t, exported, "exported JSON must not be empty")

		// Delete the original account so we can re-import without conflict.
		err = ks.Delete(acc, "original")
		require.NoError(t, err)
		assert.Empty(t, ks.Accounts())

		// Import using the export passphrase, storing with a different passphrase.
		imported, err := ks.Import(exported, "export-pass", "import-pass")
		require.NoError(t, err)

		assert.True(t, bytes.Equal(imported.Address, acc.Address),
			"imported account address must match original")

		// Verify the imported account is usable with the import passphrase.
		err = ks.Unlock(imported, "import-pass")
		assert.NoError(t, err)
	})

	t.Run("export with wrong passphrase fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("correct")
		require.NoError(t, err)

		_, err = ks.Export(acc, "wrong", "new")
		assert.Error(t, err, "export with wrong passphrase must fail")
	})

	t.Run("import with wrong passphrase fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		exported, err := ks.Export(acc, "pass", "export-pass")
		require.NoError(t, err)

		// Try importing with a wrong passphrase.
		_, err = ks.Import(exported, "wrong-pass", "new-pass")
		assert.Error(t, err, "import with wrong passphrase must fail")
	})

	t.Run("import duplicate account returns ErrAccountAlreadyExists", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		exported, err := ks.Export(acc, "pass", "export-pass")
		require.NoError(t, err)

		// Import without deleting the original should fail.
		_, err = ks.Import(exported, "export-pass", "new-pass")
		require.Error(t, err)
		assert.True(t, errors.Is(err, keystore.ErrAccountAlreadyExists),
			"importing duplicate account must return ErrAccountAlreadyExists")
	})

	t.Run("import invalid JSON fails", func(t *testing.T) {
		ks := newTestKeyStore(t)

		_, err := ks.Import([]byte("not-valid-json"), "pass", "pass")
		assert.Error(t, err)
	})
}

// ---------- ImportECDSA ----------

func TestImportECDSA(t *testing.T) {
	t.Run("import and use ECDSA key", func(t *testing.T) {
		ks := newTestKeyStore(t)

		// Create an account to obtain a valid private key, then export and re-import via ECDSA.
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		// Get the decrypted key to extract the private key.
		_, key, err := ks.GetDecryptedKey(acc, "pass")
		require.NoError(t, err)
		privKey := key.PrivateKey

		// Delete original so we can re-import.
		err = ks.Delete(acc, "pass")
		require.NoError(t, err)

		imported, err := ks.ImportECDSA(privKey, "ecdsa-pass")
		require.NoError(t, err)

		assert.True(t, bytes.Equal(imported.Address, acc.Address),
			"imported ECDSA account must have same address")

		// Verify it can be unlocked.
		err = ks.Unlock(imported, "ecdsa-pass")
		assert.NoError(t, err)
	})

	t.Run("import duplicate ECDSA key returns error", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		_, key, err := ks.GetDecryptedKey(acc, "pass")
		require.NoError(t, err)

		_, err = ks.ImportECDSA(key.PrivateKey, "pass2")
		require.Error(t, err)
		assert.True(t, errors.Is(err, keystore.ErrAccountAlreadyExists))
	})
}

// ---------- GetDecryptedKey ----------

func TestGetDecryptedKey(t *testing.T) {
	t.Run("returns key with correct passphrase", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		foundAcc, key, err := ks.GetDecryptedKey(acc, "pass")
		require.NoError(t, err)
		assert.NotNil(t, key)
		assert.NotNil(t, key.PrivateKey)
		assert.True(t, bytes.Equal(foundAcc.Address, acc.Address))
	})

	t.Run("fails with wrong passphrase", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("correct")
		require.NoError(t, err)

		_, _, err = ks.GetDecryptedKey(acc, "wrong")
		assert.Error(t, err)
	})
}

// ---------- Wallets ----------

func TestWallets(t *testing.T) {
	t.Run("empty keystore returns no wallets", func(t *testing.T) {
		ks := newTestKeyStore(t)
		wallets := ks.Wallets()
		assert.Empty(t, wallets)
	})

	t.Run("one account produces one wallet", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		wallets := ks.Wallets()
		require.Len(t, wallets, 1)

		// Wallet URL should match account URL.
		assert.Equal(t, acc.URL, wallets[0].URL())

		// Wallet accounts should contain the created account.
		wAccounts := wallets[0].Accounts()
		require.Len(t, wAccounts, 1)
		assert.True(t, bytes.Equal(wAccounts[0].Address, acc.Address))
	})

	t.Run("wallet status is Locked when not unlocked", func(t *testing.T) {
		ks := newTestKeyStore(t)
		_, err := ks.NewAccount("pass")
		require.NoError(t, err)

		wallets := ks.Wallets()
		require.Len(t, wallets, 1)

		status, err := wallets[0].Status()
		require.NoError(t, err)
		assert.Equal(t, "Locked", status)
	})

	t.Run("wallet status is Unlocked after unlock", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		err = ks.Unlock(acc, "pass")
		require.NoError(t, err)

		wallets := ks.Wallets()
		require.Len(t, wallets, 1)

		status, err := wallets[0].Status()
		require.NoError(t, err)
		assert.Equal(t, "Unlocked", status)
	})
}

// ---------- EncryptKey / DecryptKey ----------

func TestEncryptDecryptKey(t *testing.T) {
	t.Run("round-trip encrypt then decrypt", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		_, key, err := ks.GetDecryptedKey(acc, "pass")
		require.NoError(t, err)

		encrypted, err := keystore.EncryptKey(key, "encrypt-pass", keystore.LightScryptN, keystore.LightScryptP)
		require.NoError(t, err)
		assert.NotEmpty(t, encrypted)

		decrypted, err := keystore.DecryptKey(encrypted, "encrypt-pass")
		require.NoError(t, err)
		assert.NotNil(t, decrypted)
		assert.NotNil(t, decrypted.PrivateKey)
		assert.True(t, bytes.Equal(decrypted.Address, key.Address),
			"decrypted key address must match original")
	})

	t.Run("decrypt with wrong passphrase fails", func(t *testing.T) {
		ks := newTestKeyStore(t)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		_, key, err := ks.GetDecryptedKey(acc, "pass")
		require.NoError(t, err)

		encrypted, err := keystore.EncryptKey(key, "correct", keystore.LightScryptN, keystore.LightScryptP)
		require.NoError(t, err)

		_, err = keystore.DecryptKey(encrypted, "wrong")
		assert.Error(t, err, "decrypt with wrong passphrase must fail")
	})

	t.Run("decrypt invalid JSON fails", func(t *testing.T) {
		_, err := keystore.DecryptKey([]byte("{invalid"), "pass")
		assert.Error(t, err)
	})
}

// ---------- StoreKey (package-level function) ----------

func TestStoreKey(t *testing.T) {
	t.Run("store and retrieve key", func(t *testing.T) {
		dir := t.TempDir()

		acc, err := keystore.StoreKey(dir, "pass", keystore.LightScryptN, keystore.LightScryptP)
		require.NoError(t, err)
		assert.NotEmpty(t, acc.Address, "stored account must have an address")
	})
}

// ---------- ForPath ----------

func TestForPath(t *testing.T) {
	dir := t.TempDir()
	ks := keystore.ForPath(dir)
	require.NotNil(t, ks)
	assert.Empty(t, ks.Accounts(), "fresh keystore from ForPath should have no accounts")

	// Create an account to verify it functions.
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)
	assert.NotEmpty(t, acc.Address)
	assert.Len(t, ks.Accounts(), 1)
}

// ---------- Subscribe ----------

func TestSubscribe(t *testing.T) {
	t.Run("subscribe receives wallet events", func(t *testing.T) {
		ks := newTestKeyStore(t)

		sink := make(chan keystore.WalletEvent, 4)
		sub := ks.Subscribe(sink)
		defer sub.Unsubscribe()

		// Creating an account should trigger a WalletArrived event.
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)

		require.NotNil(t, sub)

		select {
		case evt := <-sink:
			assert.Equal(t, keystore.WalletArrived, evt.Kind)
			evtAccounts := evt.Wallet.Accounts()
			require.NotEmpty(t, evtAccounts)
			assert.Equal(t, acc.Address, evtAccounts[0].Address)
		case <-time.After(2 * time.Second):
			t.Fatal("expected wallet event but none was received")
		}
	})
}
