package store_test

import (
	"fmt"
	"os"
	"path"
	"sync"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/require"
)

func TestStoreConcurrentFromAddress(t *testing.T) {
	t.Parallel()

	s := newTempStore(t)
	s.SetKeystoreFactory(keystore.ForPathLight)
	loc := s.DefaultLocation()

	// Create two accounts in separate directories.
	var addrs []string
	for _, name := range []string{"race-a", "race-b"} {
		acctPath := path.Join(loc, name)
		require.NoError(t, os.MkdirAll(acctPath, 0700))

		ks := keystore.NewKeyStore(acctPath, keystore.LightScryptN, keystore.LightScryptP)
		acc, err := ks.NewAccount("pass")
		require.NoError(t, err)
		addrs = append(addrs, acc.Address.String())
		ks.Close()
	}

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines * len(addrs))

	for i := 0; i < goroutines; i++ {
		for _, addr := range addrs {
			go func(a string) {
				defer wg.Done()
				ks := s.FromAddress(a)
				if ks != nil {
					ks.Close()
				}
			}(addr)
		}
	}

	wg.Wait()
}

func TestStoreConcurrentCloseAll(t *testing.T) {
	t.Parallel()

	s := newTempStore(t)
	s.SetKeystoreFactory(keystore.ForPathLight)
	loc := s.DefaultLocation()

	// Open several keystores.
	for _, name := range []string{"close-a", "close-b", "close-c"} {
		require.NoError(t, os.MkdirAll(path.Join(loc, name), 0700))
		_ = s.FromAccountName(name)
	}

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.CloseAll()
		}()
	}

	wg.Wait()
}

func TestStoreConcurrentSetDefaultLocation(t *testing.T) {
	t.Parallel()

	s := newTempStore(t)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.SetDefaultLocation(fmt.Sprintf("/tmp/test-%d", n))
		}(i)
	}
	wg.Wait()
}

func TestStoreConcurrentFromAccountName(t *testing.T) {
	t.Parallel()

	s := newTempStore(t)
	s.SetKeystoreFactory(keystore.ForPathLight)
	loc := s.DefaultLocation()

	names := []string{"conc-a", "conc-b", "conc-c"}
	for _, name := range names {
		require.NoError(t, os.MkdirAll(path.Join(loc, name), 0700))
	}

	const goroutines = 5
	var wg sync.WaitGroup
	wg.Add(goroutines * len(names))

	for i := 0; i < goroutines; i++ {
		for _, name := range names {
			go func(n string) {
				defer wg.Done()
				ks := s.FromAccountName(n)
				if ks != nil {
					ks.Close()
				}
			}(name)
		}
	}

	wg.Wait()
}
