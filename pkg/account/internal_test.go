package account

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupInternalTestStore redirects the account store to a temporary directory.
func setupInternalTestStore(t *testing.T) {
	t.Helper()
	origConfigDir := common.DefaultConfigDirName
	tmpDir := t.TempDir()
	store.SetDefaultLocation(tmpDir)
	store.SetKeystoreFactory(keystore.ForPathLight)
	t.Cleanup(func() {
		store.CloseAll()
		common.DefaultConfigDirName = origConfigDir
	})
}

func TestWriteToFile_CreatesFileWithContent(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test-output.txt")

	content := "hello, world"
	err := writeToFile(filePath, content)
	require.NoError(t, err)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestWriteToFile_CreatesNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "a", "b", "c", "nested.txt")

	content := "nested content"
	err := writeToFile(filePath, content)
	require.NoError(t, err)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestWriteToFile_OverwritesExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "overwrite.txt")

	err := writeToFile(filePath, "first")
	require.NoError(t, err)

	err = writeToFile(filePath, "second")
	require.NoError(t, err)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "second", string(data))
}

func TestWriteToFile_InvalidPath(t *testing.T) {
	// /dev/null is a file, not a directory, so creating a file "under" it should fail.
	err := writeToFile("/dev/null/impossible/file.txt", "data")
	require.Error(t, err)
}

func TestGenerateName_ProducesValidName(t *testing.T) {
	setupInternalTestStore(t)

	name, err := generateName()
	require.NoError(t, err)
	assert.NotEmpty(t, name)
	// Name should be a single BIP-39 word: lowercase letters only, reasonable length.
	assert.Regexp(t, `^[a-z]+$`, name)
	assert.GreaterOrEqual(t, len(name), 3, "name should be at least 3 characters")
	assert.LessOrEqual(t, len(name), 20, "name should be at most 20 characters")
}

func TestGenerateName_Uniqueness(t *testing.T) {
	setupInternalTestStore(t)

	seen := make(map[string]struct{})
	const iterations = 20
	for range iterations {
		name, err := generateName()
		require.NoError(t, err)
		seen[name] = struct{}{}
	}
	// With 20 calls drawing from 2048-word BIP-39 list, collisions are unlikely.
	// We expect at least 10 distinct names.
	assert.GreaterOrEqual(t, len(seen), 10,
		"expected at least 10 unique names from %d calls, got %d", iterations, len(seen))
}
