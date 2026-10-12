package store_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/require"
)

// newOutsideKeystore writes a keystore in a folder next to accountsDir, which
// a name such as "../outside" would reach if it were not validated.
func newOutsideKeystore(t *testing.T, accountsDir string) {
	t.Helper()
	outside := filepath.Join(filepath.Dir(accountsDir), "outside")
	ks := keystore.NewKeyStore(outside, keystore.LightScryptN, keystore.LightScryptP)
	_, err := ks.NewAccount("pass")
	ks.Close()
	require.NoError(t, err)
}

var invalidAccountNames = []string{"", ".", "..", "../outside", "a/b", "/abs"}

func TestAddressFromAccountName_RejectsInvalidNames(t *testing.T) {
	loc := withTempLocation(t)
	newOutsideKeystore(t, loc)

	for _, name := range invalidAccountNames {
		t.Run(name, func(t *testing.T) {
			addr, err := store.AddressFromAccountName(name)
			require.ErrorIs(t, err, store.ErrInvalidAccountName)
			require.Empty(t, addr)
		})
	}
}

func TestStore_AddressFromAccountName_RejectsInvalidNames(t *testing.T) {
	s := newTempStore(t)
	newOutsideKeystore(t, s.DefaultLocation())

	for _, name := range invalidAccountNames {
		t.Run(name, func(t *testing.T) {
			addr, err := s.AddressFromAccountName(name)
			require.ErrorIs(t, err, store.ErrInvalidAccountName)
			require.Empty(t, addr)
		})
	}
}

func TestAddressFromAccountName_ValidNameStillResolves(t *testing.T) {
	loc := withTempLocation(t)
	ks := keystore.NewKeyStore(filepath.Join(loc, "my-wallet"), keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("pass")
	ks.Close()
	require.NoError(t, err)

	addr, err := store.AddressFromAccountName("my-wallet")
	require.NoError(t, err)
	require.Equal(t, acct.Address.String(), addr)

	_, err = os.Stat(filepath.Join(loc, "my-wallet"))
	require.NoError(t, err)
}

// Before v0.26.0 account names were not validated, so on Unix a name with a
// backslash, an ordinary file name character there, could be created. Such
// accounts must still resolve by name.
func TestAddressFromAccountName_LegacyBackslashName(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a backslash is a path separator on Windows")
	}
	loc := withTempLocation(t)
	ks := keystore.NewKeyStore(filepath.Join(loc, `work\main`), keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("pass")
	ks.Close()
	require.NoError(t, err)

	addr, err := store.AddressFromAccountName(`work\main`)
	require.NoError(t, err)
	require.Equal(t, acct.Address.String(), addr)
}
