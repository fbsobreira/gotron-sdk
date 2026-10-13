package cmd

import (
	"os"
	"path/filepath"
	"testing"

	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Without a home directory a relative config dir cannot be resolved and
// store.NameExists reports every valid name as taken. The keys commands must
// report that cause, not "already exists" or "no free name found", and write
// nothing relative to the current directory.
func TestKeysCommands_NoHomeDirectory(t *testing.T) {
	keyFile := filepath.Join(t.TempDir(), "key.json")
	require.NoError(t, os.WriteFile(keyFile, []byte(`{"address":"not-checked"}`), 0600))

	runCmd := func(build func() *cobra.Command, args ...string) func() error {
		return func() error {
			cmd := build()
			return cmd.RunE(cmd, args)
		}
	}
	tests := []struct {
		name string
		call func() error
	}{
		{"generateImportName", func() error {
			_, err := generateImportName()
			return err
		}},
		{"checkNewAccountName", func() error { return checkNewAccountName("alice") }},
		{"addPQKey", func() error {
			return addPQKey(&cobra.Command{}, "alice", core.PQScheme_ML_DSA_44)
		}},
		{"keys add", runCmd(keysAddCmd, "alice")},
		{"keys import-ks generated name", runCmd(keysImportKSCmd, keyFile)},
		{"keys import-ks given name", runCmd(keysImportKSCmd, keyFile, "alice")},
		{"keys import-private-key generated name", runCmd(keysImportPKCmd, "01")},
		{"keys import-private-key given name", runCmd(keysImportPKCmd, "01", "alice")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pqTestEnv(t, "pw")
			t.Setenv("HOME", "")
			t.Setenv("USERPROFILE", "")
			t.Setenv("home", "") // Plan 9
			if _, err := os.UserHomeDir(); err == nil {
				t.Skip("os.UserHomeDir still succeeds without HOME")
			}
			cwd := t.TempDir()
			t.Chdir(cwd)
			c.DefaultConfigDirName = ".tronctl-nohome-test"

			err := tt.call()
			require.ErrorIs(t, err, errNoKeystoreDir)
			assert.NotContains(t, err.Error(), "already exists")
			assert.NotContains(t, err.Error(), "no free name")

			entries, err := os.ReadDir(cwd)
			require.NoError(t, err)
			assert.Empty(t, entries, "nothing may be created relative to the current directory")
		})
	}
}

func TestResolveConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	assert.Equal(t, home+"/.config/tronctl", resolveConfigDir())

	// Unset HOME keeps the historical path (the drive root on Windows), so
	// existing configs there still load.
	t.Setenv("HOME", "")
	assert.Equal(t, "/.config/tronctl", resolveConfigDir())
}

// TestInitConfig_FolderCannotBeCreated covers the startup path that used to
// panic: when the config folder cannot be created, initConfig keeps the
// defaults in memory and records configErr, which commands that read the
// config return, while version, docs and completion still run.
func TestInitConfig_FolderCannotBeCreated(t *testing.T) {
	savedConfig, savedErr, savedDir, savedFile := config, configErr, ConfigDir, DefaultConfigFile
	t.Cleanup(func() {
		config, configErr, ConfigDir, DefaultConfigFile = savedConfig, savedErr, savedDir, savedFile
	})
	ConfigDir, DefaultConfigFile = "", ""

	// A HOME whose parent is a regular file: MkdirAll cannot create it.
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0600))
	t.Setenv("HOME", filepath.Join(blocker, "home"))

	require.NotPanics(t, initConfig)
	require.Error(t, configErr)
	assert.Contains(t, configErr.Error(), "config folder")
	require.NotNil(t, config)
	assert.Equal(t, defaultNodeAddr, config.Node)
	assert.EqualValues(t, defaultTimeout, config.Timeout)
	assert.Empty(t, DefaultConfigFile)

	assert.ErrorIs(t, RootCmd.PersistentPreRunE(RootCmd, nil), configErr)
	keys := &cobra.Command{Use: "keys"}
	list := &cobra.Command{Use: "list"}
	keys.AddCommand(list)
	assert.ErrorIs(t, configGate(list), configErr)
	for _, name := range []string{"version", "docs", "completion"} {
		cmd := &cobra.Command{Use: name}
		sub := &cobra.Command{Use: "bash"}
		cmd.AddCommand(sub)
		assert.NoError(t, configGate(cmd), name)
		assert.NoError(t, configGate(sub), name+" bash")
	}
}

func TestNoKeystoreDirWrapsStoreSentinel(t *testing.T) {
	assert.ErrorIs(t, errNoKeystoreDir, store.ErrNoConfigDir)
}
