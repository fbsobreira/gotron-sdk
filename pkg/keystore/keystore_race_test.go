package keystore_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/require"
)

func TestKeyStoreConcurrentUnlock(t *testing.T) {
	t.Parallel()

	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)

	const goroutines = 10
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if err := ks.Unlock(acc, "pass"); err != nil {
				errs <- fmt.Errorf("Unlock: %w", err)
			}
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the account is unlocked by signing.
	hash := randomHash(t)
	sig, err := ks.SignHash(acc, hash)
	require.NoError(t, err)
	require.Len(t, sig, 65)
}

func TestKeyStoreConcurrentSignHash(t *testing.T) {
	t.Parallel()

	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)
	require.NoError(t, ks.Unlock(acc, "pass"))

	const goroutines = 10
	errs := make(chan error, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			hash := randomHash(t)
			if _, err := ks.SignHash(acc, hash); err != nil {
				errs <- fmt.Errorf("SignHash: %w", err)
			}
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestKeyStoreConcurrentNewAccount(t *testing.T) {
	t.Parallel()

	ks := newTestKeyStore(t)

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)

	accounts := make([]keystore.Account, goroutines)
	errs := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			accounts[idx], errs[idx] = ks.NewAccount("pass")
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		require.NoError(t, err, "goroutine %d failed to create account", i)
	}

	allAccounts := ks.Accounts()
	require.Len(t, allAccounts, goroutines)
}

func TestKeyStoreConcurrentLockUnlock(t *testing.T) {
	t.Parallel()

	ks := newTestKeyStore(t)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)

	const goroutines = 10
	errs := make(chan error, goroutines*2)
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if err := ks.Unlock(acc, "pass"); err != nil {
				errs <- fmt.Errorf("Unlock: %w", err)
			}
		}()
		go func() {
			defer wg.Done()
			if err := ks.Lock(acc.Address); err != nil {
				errs <- fmt.Errorf("Lock: %w", err)
			}
		}()
	}

	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("unexpected error: %v", err)
	}
}
