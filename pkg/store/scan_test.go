package store_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unreadableNames(list []store.UnreadablePQAccount) []string {
	names := make([]string, 0, len(list))
	for _, u := range list {
		names = append(names, u.Name)
	}
	return names
}

func TestScanPQAccounts(t *testing.T) {
	s, _ := newTempStoreRoot(t)

	// A missing pq-keys folder is two empty lists and no error.
	good, bad, err := s.ScanPQAccounts()
	require.NoError(t, err)
	assert.Empty(t, good)
	assert.Empty(t, bad)
	assert.NotNil(t, bad)

	dataB, _ := mlKeystore(t, "pw")
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
	require.NoError(t, os.MkdirAll(filepath.Join(pq, "emptydir"), 0700))
	// Plain files are ignored, as they always were.
	write("stray-file.json", dataB)
	write(".DS_Store", []byte{0, 1, 2})
	want := []string{"badaddr", "badscheme", "corrupt", "emptydir", "twofiles"}
	if runtime.GOOS != "windows" {
		// A folder whose name is not a valid account name (a backslash is an
		// ordinary file name character on Unix).
		write(`back\slash/UTC--x.json`, dataB)
		want = []string{"back\\slash", "badaddr", "badscheme", "corrupt", "emptydir", "twofiles"}
	}

	good, bad, err = s.ScanPQAccounts()
	require.NoError(t, err)
	assert.Equal(t, []store.PQAccount{alice, bob}, good, "sorted by name")
	assert.Equal(t, want, unreadableNames(bad), "unreadable folders, sorted by name")
	for _, u := range bad {
		assert.Error(t, u.Err, u.Name)
		_, byName := s.PQAccountByName(u.Name)
		if u.Name != `back\slash` {
			assert.Error(t, byName, u.Name)
		}
	}

	// PQAccounts keeps its behaviour: the same usable list, no extra error.
	list, err := s.PQAccounts()
	require.NoError(t, err)
	assert.Equal(t, good, list)
}

func TestScanPQAccounts_ReadError(t *testing.T) {
	s, _ := newTempStoreRoot(t)
	// pq-keys as a plain file: ReadDir fails with something other than not-exist.
	require.NoError(t, os.WriteFile(s.PQKeysDir(), []byte("x"), 0600))
	good, bad, err := s.ScanPQAccounts()
	require.Error(t, err)
	assert.Nil(t, good)
	assert.Nil(t, bad)
	_, err = s.PQAccounts()
	require.Error(t, err)
}

func TestScanPQAccounts_PackageLevel(t *testing.T) {
	withTempLocation(t)

	dataB, _ := mlKeystore(t, "pw")
	bob, err := store.ImportPQKeystore("bob", dataB, "pw")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(store.PQKeysDir(), "broken"), 0700))

	good, bad, err := store.ScanPQAccounts()
	require.NoError(t, err)
	assert.Equal(t, []store.PQAccount{bob}, good)
	assert.Equal(t, []string{"broken"}, unreadableNames(bad))
}
