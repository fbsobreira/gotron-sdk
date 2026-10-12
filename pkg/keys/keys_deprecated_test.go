package keys

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/require"
)

// captureOutput runs fn with os.Stdout and os.Stderr redirected. This package's
// tests do not run in parallel, so swapping the globals is safe here. The
// originals are restored by t.Cleanup too, so a failing fn cannot leave them
// redirected.
func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	redirect := func(f **os.File) func() string {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		saved := *f
		*f = w
		var buf bytes.Buffer
		copyErr := make(chan error, 1)
		go func() {
			_, err := io.Copy(&buf, r)
			copyErr <- err
		}()
		restored := false
		restore := func() {
			if restored {
				return
			}
			restored = true
			*f = saved
			if err := w.Close(); err != nil {
				t.Errorf("close capture pipe: %v", err)
			}
			if err := <-copyErr; err != nil {
				t.Errorf("read captured output: %v", err)
			}
			if err := r.Close(); err != nil {
				t.Errorf("close capture pipe: %v", err)
			}
		}
		t.Cleanup(restore)
		return func() string {
			restore()
			return buf.String()
		}
	}
	stopOut := redirect(&os.Stdout)
	stopErr := redirect(&os.Stderr)
	fn()
	return stopOut(), stopErr()
}

// setHome points the home folder at dir for os.UserHomeDir on every OS.
func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

func TestListKeys_UsesGivenDirectory(t *testing.T) {
	setHome(t, t.TempDir()) // the default folder must not be touched
	dir := t.TempDir()
	ks := keystore.NewKeyStore(dir, keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("pass")
	ks.Close()
	require.NoError(t, err)

	out, _ := captureOutput(t, func() { ListKeys(dir) })
	require.Contains(t, out, acct.Address.String())

	home, err := os.UserHomeDir()
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(home, ".tronctl", "keystore"))
	require.True(t, os.IsNotExist(err), "ListKeys(dir) must not create the default folder")
}

func TestAddNewKey_FailureIsNotReportedAsAnAccount(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	// A file where the keystore folder should be makes NewAccount fail.
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".tronctl"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".tronctl", "keystore"), nil, 0o600))

	out, errOut := captureOutput(t, func() { AddNewKey("pass") })
	require.NotContains(t, out, "account:")
	require.NotContains(t, out, "URL:")
	require.Contains(t, errOut, "new account error")
}
