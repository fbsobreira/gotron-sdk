package account_test

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bip39 "github.com/fbsobreira/go-bip39"
	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// savePQName stores a fresh ML-DSA-44 account under name in the test store.
func savePQName(t *testing.T, name string) {
	t.Helper()
	k, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	require.NoError(t, err)
	t.Cleanup(k.Destroy)
	_, err = store.SavePQKey(name, k, "pw", keystore.LightScryptN, keystore.LightScryptP)
	require.NoError(t, err)
}

// exportedKeystoreFile returns a valid keystore JSON file whose address is no
// longer in the store, so ImportKeyStore would accept it under a free name.
func exportedKeystoreFile(t *testing.T) string {
	t.Helper()
	_, err := account.ImportFromPrivateKey(testPrivateKey, "ks-src", "pass")
	require.NoError(t, err)
	addr, err := store.AddressFromAccountName("ks-src")
	require.NoError(t, err)
	file, err := account.ExportKeystore(addr, t.TempDir(), "pass")
	require.NoError(t, err)
	require.NoError(t, account.RemoveAccount("ks-src"))
	return file
}

// accountPaths runs each create/import path with a given account name.
var accountPaths = []struct {
	name string
	run  func(t *testing.T, acctName string) error
}{
	{
		name: "ImportFromPrivateKey",
		run: func(t *testing.T, acctName string) error {
			_, err := account.ImportFromPrivateKey(testPrivateKey, acctName, "pass")
			return err
		},
	},
	{
		name: "ImportKeyStore",
		run: func(t *testing.T, acctName string) error {
			_, err := account.ImportKeyStore(exportedKeystoreFile(t), acctName, "pass")
			return err
		},
	},
	{
		name: "CreateNewLocalAccount",
		run: func(t *testing.T, acctName string) error {
			return account.CreateNewLocalAccount(&account.Creation{Name: acctName, Passphrase: "pass"})
		},
	},
}

// A name used by a post-quantum account must not be reused for an ECDSA one.
func TestCreateImport_RejectsNameTakenByPQAccount(t *testing.T) {
	for _, p := range accountPaths {
		t.Run(p.name, func(t *testing.T) {
			setupTestStore(t)
			savePQName(t, "taken")

			err := p.run(t, "taken")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "already exists")
			assert.False(t, store.DoesNamedAccountExist("taken"),
				"no ECDSA account may be created under a PQ account's name")
		})
	}
}

// Names that are not a single path element must be rejected before any
// file is written, so they cannot escape account-keys.
func TestCreateImport_RejectsInvalidNames(t *testing.T) {
	names := []struct {
		name     string
		acctName string
	}{
		{name: "dot", acctName: "."},
		{name: "dotdot", acctName: ".."},
		{name: "parent traversal", acctName: "../escape"},
		{name: "nested traversal", acctName: "a/../../escape"},
		{name: "forward slash", acctName: "a/b"},
		{name: "backslash", acctName: `a\b`},
		{name: "absolute", acctName: "/escape"},
		{name: "NUL byte", acctName: "a\x00b"},
	}
	for _, p := range accountPaths {
		for _, n := range names {
			t.Run(p.name+"/"+n.name, func(t *testing.T) {
				setupTestStore(t)
				root := common.DefaultConfigDirName

				err := p.run(t, n.acctName)
				require.Error(t, err)
				assert.ErrorIs(t, err, store.ErrInvalidAccountName)
				assert.Contains(t, err.Error(), "invalid account name")

				_, statErr := os.Stat(filepath.Join(root, "escape"))
				assert.True(t, os.IsNotExist(statErr), "nothing may be written outside account-keys")
				assert.Empty(t, store.LocalAccounts(), "no account may be created")
			})
		}
	}
}

func TestCreateNewLocalAccount_RejectsEmptyName(t *testing.T) {
	setupTestStore(t)
	err := account.CreateNewLocalAccount(&account.Creation{Name: "", Passphrase: "pass"})
	require.Error(t, err)
	assert.ErrorIs(t, err, store.ErrInvalidAccountName)
	assert.Empty(t, store.LocalAccounts())
}

// The auto-generated "<word>-imported" name must skip names used in pq-keys.
// All but a few BIP39 words are taken as PQ names, so without the cross-store
// check the generated name almost surely collides.
func TestImport_AutoNameAvoidsPQNames(t *testing.T) {
	words := bip39.GetWordList()
	require.Len(t, words, 2048)
	free := map[string]bool{}
	for _, w := range words[:16] {
		free[w+"-imported"] = true
	}

	for _, p := range accountPaths[:2] {
		t.Run(p.name, func(t *testing.T) {
			setupTestStore(t)
			// NameExists only checks that pq-keys/<name> exists, so empty
			// folders stand in for PQ accounts.
			pqDir := filepath.Join(common.DefaultConfigDirName, common.DefaultConfigPQKeysDirName)
			for _, w := range words[16:] {
				require.NoError(t, os.MkdirAll(filepath.Join(pqDir, w+"-imported"), 0700))
			}

			var name string
			var err error
			if p.name == "ImportFromPrivateKey" {
				name, err = account.ImportFromPrivateKey(testPrivateKey, "", "pass")
			} else {
				name, err = account.ImportKeyStore(exportedKeystoreFile(t), "", "pass")
			}
			require.NoError(t, err)
			require.True(t, strings.HasSuffix(name, "-imported"), name)
			assert.True(t, free[name], "auto name %q collides with a post-quantum account", name)
			assert.True(t, store.DoesNamedAccountExist(name))
		})
	}
}

func TestRemoveAccount_NotFoundIsErrAccountNotFound(t *testing.T) {
	tests := []struct {
		name     string
		acctName string
		wantMsg  string
	}{
		{name: "missing account", acctName: "does-not-exist", wantMsg: "account does-not-exist doesn't exist"},
		{name: "empty name", acctName: "", wantMsg: "account  doesn't exist"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupTestStore(t)
			err := account.RemoveAccount(tt.acctName)
			require.Error(t, err)
			assert.True(t, errors.Is(err, account.ErrAccountNotFound))
			assert.EqualError(t, err, tt.wantMsg, "the error message must not change")
		})
	}
}
