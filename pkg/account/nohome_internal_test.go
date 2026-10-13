package account

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noHomeRelativeStore removes the home directory from the environment, points
// the package-level store at a relative config dir (which then cannot be
// resolved) and moves into an empty temp directory, so any write relative to
// the current directory is detectable. It returns that directory.
func noHomeRelativeStore(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("home", "") // Plan 9
	if _, err := os.UserHomeDir(); err == nil {
		t.Skipf("os.UserHomeDir still succeeds on %s without HOME", runtime.GOOS)
	}
	orig := common.DefaultConfigDirName
	common.DefaultConfigDirName = ".tronctl-account-nohome-test"
	t.Cleanup(func() {
		store.CloseAll()
		common.DefaultConfigDirName = orig
	})
	dir := t.TempDir()
	t.Chdir(dir)
	require.Equal(t, "", store.DefaultLocation(), "the config dir must not resolve")
	return dir
}

// withDeadline runs fn and kills the test binary if it does not return
// within 5s. It panics from a timer goroutine instead of failing the test:
// a test failure would run the cleanups, which restore HOME while fn keeps
// running, and fn could then write to the real keystore.
func withDeadline(t *testing.T, what string, fn func() error) error {
	t.Helper()
	timer := time.AfterFunc(5*time.Second, func() {
		panic(fmt.Sprintf("%s did not return within 5s without a home directory "+
			"(endless name loop?)", what))
	})
	defer timer.Stop()
	return fn()
}

func requireNothingIn(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	assert.Empty(t, names, "nothing may be created relative to the current directory")
}

func TestNoHome_FailsBeforePickingAName(t *testing.T) {
	const pk = "0x0000000000000000000000000000000000000000000000000000000000000001"

	keyFile := filepath.Join(t.TempDir(), "key.json")
	require.NoError(t, os.WriteFile(keyFile, []byte(`{"not":"checked"}`), 0600))

	tests := []struct {
		name string
		call func() error
	}{
		{"ImportFromPrivateKey generated name", func() error {
			_, err := ImportFromPrivateKey(pk, "", "pw")
			return err
		}},
		{"ImportFromPrivateKey given name", func() error {
			_, err := ImportFromPrivateKey(pk, "alice", "pw")
			return err
		}},
		{"ImportKeyStore generated name", func() error {
			_, err := ImportKeyStore(keyFile, "", "pw")
			return err
		}},
		{"ImportKeyStore given name", func() error {
			_, err := ImportKeyStore(keyFile, "alice", "pw")
			return err
		}},
		{"CreateNewLocalAccount", func() error {
			return CreateNewLocalAccount(&Creation{Name: "alice", Passphrase: "pw"})
		}},
		{"RemoveAccount", func() error {
			return RemoveAccount("alice")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cwd := noHomeRelativeStore(t)
			err := withDeadline(t, tt.name, tt.call)
			require.ErrorIs(t, err, errNoConfigDir)
			assert.NotContains(t, err.Error(), "already exists")
			requireNothingIn(t, cwd)
		})
	}
}
