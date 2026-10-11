package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveKsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	require.NoError(t, err)
	abs := filepath.Join(t.TempDir(), "cfg")

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "absolute", in: abs, want: abs},
		{name: "absolute uncleaned", in: abs + "/../cfg/", want: abs},
		{name: "relative is under the current directory", in: ".tronctl", want: filepath.Join(cwd, ".tronctl")},
		{name: "dot", in: ".", want: cwd},
		{name: "tilde", in: "~", want: home},
		{name: "tilde slash", in: "~/cfg", want: filepath.Join(home, "cfg")},
		{name: "tilde user is not expanded", in: "~other/cfg", want: filepath.Join(cwd, "~other", "cfg")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveKsDir(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHasHomePrefix(t *testing.T) {
	tests := []struct {
		dir  string
		sep  byte
		want bool
	}{
		{dir: "~", sep: '/', want: true},
		{dir: "~/cfg", sep: '/', want: true},
		{dir: "~/cfg", sep: '\\', want: true},
		{dir: `~\cfg`, sep: '\\', want: true},
		{dir: `~\cfg`, sep: '/', want: false}, // \ is a filename character on Unix
		{dir: "~other/cfg", sep: '/', want: false},
		{dir: "cfg/~", sep: '/', want: false},
		{dir: "", sep: '/', want: false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, hasHomePrefix(tt.dir, tt.sep), "%q with separator %q", tt.dir, tt.sep)
	}
}

// TestSetupKeystoreKsDirValues runs TRONCTL_KS_DIR through applyEnvOverrides
// and setupKeystore and checks where the store then looks for keystores.
func TestSetupKeystoreKsDirValues(t *testing.T) {
	pqTestEnv(t, "pw") // restores the store location
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	require.NoError(t, err)
	base := t.TempDir()

	tests := []struct {
		name     string
		env      string
		wantKeys string
		wantWarn bool
	}{
		{
			name:     "config root",
			env:      filepath.Join(base, "cfg"),
			wantKeys: filepath.Join(base, "cfg", "account-keys"),
		},
		{
			// The value is still the config root; the warning says so.
			name:     "value ending in account-keys",
			env:      filepath.Join(base, "account-keys"),
			wantKeys: filepath.Join(base, "account-keys", "account-keys"),
			wantWarn: true,
		},
		{
			name:     "relative value ending in account-keys",
			env:      ".tronctl/account-keys",
			wantKeys: filepath.Join(cwd, ".tronctl", "account-keys", "account-keys"),
			wantWarn: true,
		},
		{
			name:     "value ending in pq-keys",
			env:      filepath.Join(base, "pq-keys"),
			wantKeys: filepath.Join(base, "pq-keys", "account-keys"),
			wantWarn: true,
		},
		{
			name:     "relative value is not under the home directory",
			env:      "cfg",
			wantKeys: filepath.Join(cwd, "cfg", "account-keys"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TRONCTL_KS_DIR", tt.env)
			r := &Runtime{}
			r.applyEnvOverrides("", "", false)
			var warn bytes.Buffer
			require.NoError(t, r.setupKeystore(&warn))

			assert.Equal(t, tt.wantKeys, store.DefaultLocation())
			assert.DirExists(t, tt.wantKeys)
			if tt.wantWarn {
				root := filepath.Dir(tt.wantKeys)
				assert.Contains(t, warn.String(), "looks like a key folder")
				assert.Contains(t, warn.String(), "keys are read from "+tt.wantKeys+" and "+filepath.Join(root, "pq-keys"))
				assert.Contains(t, warn.String(), "pass the parent, "+filepath.Dir(root)+",")
			} else {
				assert.Empty(t, warn.String())
			}
			_, err := os.Stat(filepath.Join(home, "cfg"))
			assert.True(t, os.IsNotExist(err), "nothing may be created under the home directory")
		})
	}
}

func TestSetupKeystoreFlagWinsOverEnv(t *testing.T) {
	pqTestEnv(t, "pw")
	flagDir := filepath.Join(t.TempDir(), "flag")
	t.Setenv("TRONCTL_KS_DIR", filepath.Join(t.TempDir(), "env"))

	r := &Runtime{DefaultKeystoreDir: flagDir}
	r.applyEnvOverrides("", "", false)
	require.NoError(t, r.setupKeystore(&bytes.Buffer{}))
	assert.Equal(t, filepath.Join(flagDir, "account-keys"), store.DefaultLocation())
}

// TestSetupKeystoreLegacyRelativeWarning checks the hint for a relative value
// that used to mean a folder under the home directory.
func TestSetupKeystoreLegacyRelativeWarning(t *testing.T) {
	pqTestEnv(t, "pw")
	home := t.TempDir()
	t.Setenv("HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "cfg", "account-keys", "acct"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "pqonly", "pq-keys", "acct"), 0o700))
	// Empty key folders at the old location are no reason to warn.
	require.NoError(t, os.MkdirAll(filepath.Join(home, "empty", "account-keys"), 0o700))
	t.Chdir(t.TempDir())
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tests := []struct {
		name     string
		value    string
		wantWarn string
	}{
		{name: "old location has account-keys", value: "cfg", wantWarn: filepath.Join(home, "cfg")},
		{name: "old location has pq-keys", value: "pqonly", wantWarn: filepath.Join(home, "pqonly")},
		{name: "nothing at the old location", value: "fresh"},
		{name: "empty key folder at the old location", value: "empty"},
		{name: "tilde value", value: "~/cfg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var warn bytes.Buffer
			r := &Runtime{DefaultKeystoreDir: tt.value}
			require.NoError(t, r.setupKeystore(&warn))
			if tt.wantWarn == "" {
				assert.Empty(t, warn.String())
				return
			}
			assert.Contains(t, warn.String(), "now means "+filepath.Join(cwd, tt.value))
			assert.Contains(t, warn.String(), "keys exist under "+tt.wantWarn)

			// Repeated while the new root holds no keys, although the first
			// run created its account-keys folder.
			warn.Reset()
			require.NoError(t, r.setupKeystore(&warn))
			assert.Contains(t, warn.String(), "keys exist under "+tt.wantWarn)

			// Stops once the new root holds keys.
			require.NoError(t, os.MkdirAll(filepath.Join(cwd, tt.value, "account-keys", "acct"), 0o700))
			warn.Reset()
			require.NoError(t, r.setupKeystore(&warn))
			assert.Empty(t, warn.String())
		})
	}
}
