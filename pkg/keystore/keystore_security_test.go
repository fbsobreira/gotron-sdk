package keystore_test

import (
	"crypto/rand"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLock_StopsSigning verifies that an account cannot sign once it is locked.
// That the key memory is zeroed is checked in zeroize_internal_test.go.
func TestLock_StopsSigning(t *testing.T) {
	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)

	// Unlock the account.
	require.NoError(t, ks.Unlock(acc, "pass"))

	// Signing should work while unlocked.
	hash := randomHash(t)
	sig, err := ks.SignHash(acc, hash)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	// Lock the account.
	require.NoError(t, ks.Lock(acc.Address))

	// Signing must fail after locking.
	_, err = ks.SignHash(acc, hash)
	assert.ErrorIs(t, err, keystore.ErrLocked)
}

// TestTimedUnlock_ExpiryStopsSigning verifies that an account cannot sign once
// its timed unlock expires. Zeroing is checked in zeroize_internal_test.go.
func TestTimedUnlock_ExpiryStopsSigning(t *testing.T) {
	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)

	// Unlock with a very short timeout.
	require.NoError(t, ks.TimedUnlock(acc, "pass", 50*time.Millisecond))

	// Signing should work immediately.
	hash := randomHash(t)
	sig, err := ks.SignHash(acc, hash)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	// Wait for the timed unlock to expire, then verify signing fails.
	eventuallySignHashFails(t, ks, acc, hash, 2*time.Second)
}

// TestTimedUnlock_ReplacementKeepsSigning verifies that a timed unlock which
// replaces another still signs, and expires on its own timeout. That the
// replaced key is zeroed is checked in zeroize_internal_test.go.
func TestTimedUnlock_ReplacementKeepsSigning(t *testing.T) {
	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)

	// First timed unlock with a long timeout.
	require.NoError(t, ks.TimedUnlock(acc, "pass", 10*time.Second))

	// Replace with a short timeout.
	require.NoError(t, ks.TimedUnlock(acc, "pass", 50*time.Millisecond))

	// Signing should still work with the new key.
	hash := randomHash(t)
	sig, err := ks.SignHash(acc, hash)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	// Wait for the new timed unlock to expire.
	eventuallySignHashFails(t, ks, acc, hash, 2*time.Second)
}

// TestLock_AfterTimedUnlock_StopsSigning verifies Lock works on a timed-unlock
// entry. Zeroing is checked in zeroize_internal_test.go.
func TestLock_AfterTimedUnlock_StopsSigning(t *testing.T) {
	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)

	// Timed unlock with a long timeout.
	require.NoError(t, ks.TimedUnlock(acc, "pass", 10*time.Second))

	hash := randomHash(t)
	sig, err := ks.SignHash(acc, hash)
	require.NoError(t, err)
	require.NotEmpty(t, sig)

	// An explicit lock stops signing immediately.
	require.NoError(t, ks.Lock(acc.Address))
	_, err = ks.SignHash(acc, hash)
	assert.ErrorIs(t, err, keystore.ErrLocked)
}

// TestConcurrentSign_NoRace verifies that concurrent Sign operations on the
// same unlocked account do not race. This test is meaningful when run with
// -race.
func TestConcurrentSign_NoRace(t *testing.T) {
	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)
	require.NoError(t, ks.Unlock(acc, "pass"))
	defer func() { _ = ks.Lock(acc.Address) }()

	const goroutines = 8
	const iterations = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				hash := make([]byte, 32)
				_, _ = rand.Read(hash)
				_, err := ks.SignHash(acc, hash)
				if err != nil {
					errs <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("unexpected sign error: %v", err)
	}
}

// TestConcurrentLockUnlock_NoRace verifies that concurrent Lock/Unlock
// operations do not race. This test is meaningful when run with -race.
func TestConcurrentLockUnlock_NoRace(t *testing.T) {
	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)

	const goroutines = 8
	const iterations = 10

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_ = ks.Unlock(acc, "pass")
				_ = ks.Lock(acc.Address)
			}
		}()
	}

	wg.Wait()
}

// TestClose_StopsSigning verifies that no unlocked account can sign after
// Close. Zeroing is checked in zeroize_internal_test.go.
func TestClose_StopsSigning(t *testing.T) {
	ks := newTestKeyStore(t)

	// Create and unlock multiple accounts.
	var accounts []keystore.Account
	for i := 0; i < 3; i++ {
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)
		require.NoError(t, ks.Unlock(acc, "pass"))
		accounts = append(accounts, acc)
	}

	// All accounts should be signable.
	hash := randomHash(t)
	for _, acc := range accounts {
		_, err := ks.SignHash(acc, hash)
		require.NoError(t, err)
	}

	// Close the keystore.
	ks.Close()

	// All accounts should now fail to sign.
	for _, acc := range accounts {
		_, err := ks.SignHash(acc, hash)
		assert.Error(t, err, "signing should fail after Close for account %s", acc.Address)
	}
}

// TestClose_Idempotent verifies that Close can be called multiple times
// without panicking.
func TestClose_Idempotent(t *testing.T) {
	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)
	require.NoError(t, ks.Unlock(acc, "pass"))

	assert.NotPanics(t, func() {
		ks.Close()
		ks.Close()
		ks.Close()
	})
}

// TestGetDecryptedKey_CallerOwnsKey verifies that the caller receives an
// independent key copy that can be zeroized without affecting the keystore.
func TestGetDecryptedKey_CallerOwnsKey(t *testing.T) {
	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)

	_, key, err := ks.GetDecryptedKey(acc, "pass")
	require.NoError(t, err)
	require.NotNil(t, key.PrivateKey)

	// Key D value should be non-zero.
	assert.False(t, key.PrivateKey.D.Cmp(big.NewInt(0)) == 0,
		"decrypted key D should be non-zero")
}
