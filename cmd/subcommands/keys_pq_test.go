package cmd

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pqTestEnv points the store at a temp config root, writes password to a
// temp --passphrase-file and sets the CLI globals the keys commands and
// resolveSigner read. Everything is restored when the test ends. Tests that
// use it must not run in parallel: these are process-wide globals.
func pqTestEnv(t *testing.T, password string) string {
	t.Helper()
	prevDir := c.DefaultConfigDirName
	prevPPFile, prevPrompt := passphraseFilePath, userProvidesPassphrase
	prevPass, prevLedger, prevAddr, prevQuiet := passphrase, useLedgerWallet, addr, quietImport
	prevN, prevP := pqScryptN, pqScryptP
	pqScryptN, pqScryptP = keystore.LightScryptN, keystore.LightScryptP

	root := t.TempDir()
	store.SetDefaultLocation(root)
	store.SetKeystoreFactory(keystore.ForPathLight)

	ppFile := filepath.Join(t.TempDir(), "passphrase")
	require.NoError(t, os.WriteFile(ppFile, []byte(password+"\n"), 0600))
	passphraseFilePath, userProvidesPassphrase = ppFile, false
	passphrase, useLedgerWallet, quietImport = password, false, false

	t.Cleanup(func() {
		releaseSigningKeys()
		store.CloseAll() // also resets the keystore factory
		pqScryptN, pqScryptP = prevN, prevP
		// Assign directly: SetDefaultLocation would create the real
		// ~/.tronctl/account-keys folder.
		c.DefaultConfigDirName = prevDir
		passphraseFilePath, userProvidesPassphrase = prevPPFile, prevPrompt
		passphrase, useLedgerWallet, addr, quietImport = prevPass, prevLedger, prevAddr, prevQuiet
	})
	return root
}

// newMLKey generates an ML-DSA-44 key that is destroyed when the test ends.
func newMLKey(t *testing.T) pqc.PrivateKey {
	t.Helper()
	k, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	require.NoError(t, err)
	t.Cleanup(k.Destroy)
	return k
}

// savePQAccount stores a fresh ML-DSA-44 key under name with light scrypt.
func savePQAccount(t *testing.T, name, password string) store.PQAccount {
	t.Helper()
	acct, err := store.SavePQKey(name, newMLKey(t), password, keystore.LightScryptN, keystore.LightScryptP)
	require.NoError(t, err)
	return acct
}

// writeTempFile writes data to a new file in a temp dir and returns its path.
func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, data, 0600))
	return p
}

// mlKeystoreFile writes an encrypted ML-DSA-44 keystore and returns its path
// and address.
func mlKeystoreFile(t *testing.T, password string) (string, string) {
	t.Helper()
	k := newMLKey(t)
	data, err := pqc.EncryptKeystore(k, password, keystore.LightScryptN, keystore.LightScryptP)
	require.NoError(t, err)
	return writeTempFile(t, "pq.json", data), k.Public().Address().String()
}

// ecdsaKeystoreFile creates an ECDSA V3 keystore outside the store and
// returns its path and address.
func ecdsaKeystoreFile(t *testing.T, password string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	ks := keystore.ForPathLight(dir)
	acct, err := ks.NewAccount(password)
	ks.Close()
	require.NoError(t, err)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	return filepath.Join(dir, entries[0].Name()), acct.Address.String()
}

// addECDSAAccount creates an ECDSA account named name in the store.
func addECDSAAccount(t *testing.T, name, password string) string {
	t.Helper()
	ks := store.FromAccountName(name)
	defer ks.Close()
	acct, err := ks.NewAccount(password)
	require.NoError(t, err)
	return acct.Address.String()
}

// keysSubCmd returns the keys subcommand called name.
func keysSubCmd(t *testing.T, name string) *cobra.Command {
	t.Helper()
	for _, cmd := range keysSub() {
		if cmd.Name() == name {
			return cmd
		}
	}
	t.Fatalf("keys %s is not registered", name)
	return nil
}

// runKeysCmd runs cmd's PreRunE and RunE with args, capturing what the
// command writes to cmd.OutOrStdout and cmd.ErrOrStderr.
func runKeysCmd(t *testing.T, cmd *cobra.Command, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	if cmd.PreRunE != nil {
		if err := cmd.PreRunE(cmd, args); err != nil {
			return out.String(), errOut.String(), err
		}
	}
	err := cmd.RunE(cmd, args)
	return out.String(), errOut.String(), err
}

func pqAccountNames(t *testing.T) []string {
	t.Helper()
	accts, err := store.PQAccounts()
	require.NoError(t, err)
	names := make([]string, 0, len(accts))
	for _, a := range accts {
		names = append(names, a.Name)
	}
	return names
}

func TestKeysAddPQScheme(t *testing.T) {
	for _, flag := range []string{"ml-dsa-44", "ML_DSA_44"} {
		t.Run(flag, func(t *testing.T) {
			pqTestEnv(t, "pw")
			cmd := keysAddCmd()
			require.NoError(t, cmd.Flags().Set("scheme", flag))
			out, _, err := runKeysCmd(t, cmd, "pq-new")
			require.NoError(t, err)

			acct, err := store.PQAccountByName("pq-new")
			require.NoError(t, err)
			assert.Equal(t, core.PQScheme_ML_DSA_44, acct.Scheme)
			assert.False(t, store.DoesNamedAccountExist("pq-new"), "no ECDSA account may be created")
			assert.Contains(t, out, acct.Address)
			assert.Contains(t, out, "ML_DSA_44")
			assert.Contains(t, out, acct.Path)
			assert.Contains(t, out, "no mnemonic")

			k, err := store.DecryptPQKey(acct, "pw")
			require.NoError(t, err, "the key must be stored under the --passphrase-file passphrase")
			k.Destroy()
		})
	}
}

func TestKeysAddInvalidScheme(t *testing.T) {
	pqTestEnv(t, "pw")
	cmd := keysAddCmd()
	require.NoError(t, cmd.Flags().Set("scheme", "rsa"))
	_, _, err := runKeysCmd(t, cmd, "bad-scheme")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fn-dsa-512")
	assert.Contains(t, err.Error(), "ml-dsa-44")
	assert.False(t, store.NameExists("bad-scheme"))
}

func TestKeysAddFNDSAWithoutFalcon(t *testing.T) {
	if pqc.SchemeAvailable(core.PQScheme_FN_DSA_512) == nil {
		t.Skip("built with a Falcon backend")
	}
	pqTestEnv(t, "pw")
	cmd := keysAddCmd()
	require.NoError(t, cmd.Flags().Set("scheme", "fn-dsa-512"))
	_, _, err := runKeysCmd(t, cmd, "fn-key")
	require.Error(t, err)
	assert.True(t, errors.Is(err, pqc.ErrFalconUnavailable), "got %v", err)
	assert.Contains(t, err.Error(), "-tags falcon")
	assert.False(t, store.NameExists("fn-key"))
}

func TestKeysAddECDSARefusesPQName(t *testing.T) {
	pqTestEnv(t, "pw")
	savePQAccount(t, "taken", "pw")
	_, _, err := runKeysCmd(t, keysAddCmd(), "taken")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.False(t, store.DoesNamedAccountExist("taken"), "no ECDSA account may be created")
}

func TestKeysAddPQRefusesECDSAName(t *testing.T) {
	pqTestEnv(t, "pw")
	addECDSAAccount(t, "ecdsa-taken", "pw")
	cmd := keysAddCmd()
	require.NoError(t, cmd.Flags().Set("scheme", "ml-dsa-44"))
	_, _, err := runKeysCmd(t, cmd, "ecdsa-taken")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.Empty(t, pqAccountNames(t))
}

func TestKeysImportKSPQ(t *testing.T) {
	pqTestEnv(t, "pw")

	t.Run("named", func(t *testing.T) {
		file, address := mlKeystoreFile(t, "pw")
		_, _, err := runKeysCmd(t, keysSubCmd(t, "import-ks"), file, "pq-imported")
		require.NoError(t, err)
		acct, err := store.PQAccountByName("pq-imported")
		require.NoError(t, err)
		assert.Equal(t, address, acct.Address)
		assert.False(t, store.DoesNamedAccountExist("pq-imported"))
	})

	t.Run("auto name", func(t *testing.T) {
		file, address := mlKeystoreFile(t, "pw")
		_, _, err := runKeysCmd(t, keysSubCmd(t, "import-ks"), file)
		require.NoError(t, err)
		acct, err := store.PQAccountByAddress(address)
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(acct.Name, "-imported"), "name %q", acct.Name)
	})

	t.Run("wrong passphrase", func(t *testing.T) {
		file, address := mlKeystoreFile(t, "other")
		_, _, err := runKeysCmd(t, keysSubCmd(t, "import-ks"), file, "pq-wrong")
		require.Error(t, err)
		assert.True(t, errors.Is(err, keystore.ErrDecrypt), "got %v", err)
		_, err = store.PQAccountByAddress(address)
		assert.True(t, errors.Is(err, store.ErrPQAccountNotFound))
	})
}

func TestKeysImportKSECDSA(t *testing.T) {
	pqTestEnv(t, "pw")
	savePQAccount(t, "dup", "pw")
	file, address := ecdsaKeystoreFile(t, "pw")

	_, _, err := runKeysCmd(t, keysSubCmd(t, "import-ks"), file, "dup")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.False(t, store.DoesNamedAccountExist("dup"), "no ECDSA account may be created")

	_, _, err = runKeysCmd(t, keysSubCmd(t, "import-ks"), file, "ecdsa-ok")
	require.NoError(t, err)
	got, err := store.AddressFromAccountName("ecdsa-ok")
	require.NoError(t, err)
	assert.Equal(t, address, got)
}

func TestKeysImportToolkit(t *testing.T) {
	pqTestEnv(t, "pw")
	k := newMLKey(t)
	data, err := pqc.MarshalToolkitJSON(k)
	require.NoError(t, err)
	file := writeTempFile(t, "toolkit.json", data)

	out, errOut, err := runKeysCmd(t, keysSubCmd(t, "import-toolkit"), file, "tk")
	require.NoError(t, err)
	acct, err := store.PQAccountByName("tk")
	require.NoError(t, err)
	assert.Equal(t, k.Public().Address().String(), acct.Address)
	assert.Contains(t, out, acct.Address)
	assert.Contains(t, errOut, "plaintext")
	assert.FileExists(t, file, "the source file must not be deleted")

	got, err := store.DecryptPQKey(acct, "pw")
	require.NoError(t, err)
	got.Destroy()

	// A second import of the same key is refused by the store.
	_, _, err = runKeysCmd(t, keysSubCmd(t, "import-toolkit"), file, "tk2")
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrPQAddressExists), "got %v", err)
}

// randomPrivateKeyHex returns a fresh secp256k1 private key in hex.
func randomPrivateKeyHex(t *testing.T) string {
	t.Helper()
	sk, err := keys.GenerateKey()
	require.NoError(t, err)
	return hex.EncodeToString(sk.Serialize())
}

func TestKeysImportPrivateKeyRefusesPQName(t *testing.T) {
	pqTestEnv(t, "pw")
	pq := savePQAccount(t, "dup", "pw")

	_, _, err := runKeysCmd(t, keysSubCmd(t, "import-private-key"), randomPrivateKeyHex(t), "dup")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.False(t, store.DoesNamedAccountExist("dup"), "no ECDSA account may be created")
	got, err := store.PQAccountByName("dup")
	require.NoError(t, err)
	assert.Equal(t, pq.Address, got.Address)

	_, _, err = runKeysCmd(t, keysSubCmd(t, "import-private-key"), randomPrivateKeyHex(t), "../escape")
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrInvalidAccountName), "got %v", err)

	// With no name, the generated one is free in both stores.
	_, _, err = runKeysCmd(t, keysSubCmd(t, "import-private-key"), randomPrivateKeyHex(t))
	require.NoError(t, err)
	names := store.LocalAccounts()
	require.Len(t, names, 1)
	assert.True(t, strings.HasSuffix(names[0], "-imported"), "name %q", names[0])
	assert.Equal(t, []string{"dup"}, pqAccountNames(t))
}

func TestKeysRecoverMnemonicRefusesPQName(t *testing.T) {
	pqTestEnv(t, "pw")
	savePQAccount(t, "dup", "pw")

	_, _, err := runKeysCmd(t, keysSubCmd(t, "recover-from-mnemonic"), "dup")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
	assert.False(t, store.DoesNamedAccountExist("dup"), "no ECDSA account may be created")

	_, _, err = runKeysCmd(t, keysSubCmd(t, "recover-from-mnemonic"), "../escape")
	require.Error(t, err)
	assert.True(t, errors.Is(err, store.ErrInvalidAccountName), "got %v", err)
}

func TestKeysRemoveRefusesNameInBothStores(t *testing.T) {
	root := pqTestEnv(t, "pw")
	// SavePQKey refuses a taken name but the ECDSA store does not, so this
	// order builds the collision an older build could leave on disk.
	pq := savePQAccount(t, "dup", "pw")
	ecdsaAddr := addECDSAAccount(t, "dup", "pw")

	_, _, err := runKeysCmd(t, keysRemoveCmd(), "dup")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing was removed")
	assert.True(t, store.DoesNamedAccountExist("dup"), "the ECDSA account must survive")
	got, err := store.AddressFromAccountName("dup")
	require.NoError(t, err)
	assert.Equal(t, ecdsaAddr, got)
	gotPQ, err := store.PQAccountByName("dup")
	require.NoError(t, err, "the PQ account must survive")
	assert.Equal(t, pq.Address, gotPQ.Address)

	// An unreadable PQ folder still counts as a PQ account here.
	broken := filepath.Join(root, c.DefaultConfigPQKeysDirName, "dup2")
	require.NoError(t, os.MkdirAll(broken, 0700))
	addECDSAAccount(t, "dup2", "pw")
	_, _, err = runKeysCmd(t, keysRemoveCmd(), "dup2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nothing was removed")
	assert.DirExists(t, broken)
	assert.True(t, store.DoesNamedAccountExist("dup2"))

	// A name typed in another case must not slip past the check on a
	// case-insensitive filesystem (macOS, Windows); on Linux it simply
	// matches nothing. Either way both accounts survive.
	_, _, err = runKeysCmd(t, keysRemoveCmd(), "DUP")
	require.Error(t, err)
	_, err = store.PQAccountByName("dup")
	require.NoError(t, err, "the PQ account must survive an other-case remove")
	assert.True(t, store.DoesNamedAccountExist("dup"))
}

func TestKeysRemoveReportsUnreadablePQStore(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	root := pqTestEnv(t, "pw")
	addECDSAAccount(t, "solo", "pw")
	pqDir := filepath.Join(root, c.DefaultConfigPQKeysDirName)
	require.NoError(t, os.MkdirAll(pqDir, 0700))
	require.NoError(t, os.Chmod(pqDir, 0))
	t.Cleanup(func() { _ = os.Chmod(pqDir, 0700) })

	_, _, err := runKeysCmd(t, keysRemoveCmd(), "solo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "check post-quantum account")
	assert.NotContains(t, err.Error(), "names both")
	require.NoError(t, os.Chmod(pqDir, 0700))
	assert.True(t, store.DoesNamedAccountExist("solo"), "nothing may be removed")
}

func TestIsPQKeystore(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{name: "absent", data: `{"address":"x","crypto":{}}`, want: false},
		{name: "empty", data: `{"scheme":""}`, want: false},
		{name: "null", data: `{"scheme":null}`, want: false},
		{name: "ML_DSA_44", data: `{"scheme":"ML_DSA_44"}`, want: true},
		{name: "FN_DSA_512", data: `{"scheme":"FN_DSA_512"}`, want: true},
		{name: "not json", data: `nope`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isPQKeystore([]byte(tt.data)))
		})
	}
}

func TestAccountSignRefusesPQ(t *testing.T) {
	pqTestEnv(t, "pw")
	prevSigner := signerAddress
	t.Cleanup(func() { signerAddress = prevSigner })
	acct := savePQAccount(t, "pq-msg", "pw")
	require.NoError(t, signerAddress.Set(acct.Address))

	_, _, err := runKeysCmd(t, accountSignCmd(), "hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support post-quantum keys")
}

func TestKeysListShowsECDSAWhenPQUnreadable(t *testing.T) {
	root := pqTestEnv(t, "pw")
	ecdsaAddr := addECDSAAccount(t, "alice", "pw")
	// pq-keys as a plain file cannot be read as a directory.
	require.NoError(t, os.WriteFile(filepath.Join(root, c.DefaultConfigPQKeysDirName), []byte("x"), 0600))

	out, _, err := runKeysCmd(t, keysListCmd())
	require.Error(t, err, "the PQ read error must still be reported")
	assert.Contains(t, out, "NAME")
	assert.Contains(t, out, "alice"+strings.Repeat(" ", 48-len("alice"))+"\t"+ecdsaAddr+"\tECDSA\n")
}

func TestKeysListBothKinds(t *testing.T) {
	pqTestEnv(t, "pw")
	ecdsaAddr := addECDSAAccount(t, "alice", "pw")
	pq := savePQAccount(t, "bob", "pw")

	cmd := keysListCmd()
	out, _, err := runKeysCmd(t, cmd)
	require.NoError(t, err)

	lines := strings.Split(out, "\n")
	require.GreaterOrEqual(t, len(lines), 4, "output:\n%s", out)
	header := lines[0]
	assert.True(t, strings.HasPrefix(header, "NAME                    \t\t                ADDRESS"), "header %q", header)
	assert.Equal(t, []string{"NAME", "ADDRESS", "SCHEME"}, strings.Fields(header))
	assert.Empty(t, lines[1], "a blank line follows the header, as before")

	assert.Contains(t, out, "alice"+strings.Repeat(" ", 48-len("alice"))+"\t"+ecdsaAddr+"\tECDSA\n")
	assert.Contains(t, out, "bob"+strings.Repeat(" ", 48-len("bob"))+"\t"+pq.Address+"\tML_DSA_44\n")
}

func TestResolveSignerPQ(t *testing.T) {
	pqTestEnv(t, "pw")
	acct := savePQAccount(t, "pq-signer", "pw")
	var a tronAddress
	require.NoError(t, a.Set(acct.Address))

	s, err := resolveSigner(a)
	require.NoError(t, err)
	pqs, ok := s.(interface{ PQScheme() core.PQScheme })
	require.True(t, ok, "resolveSigner must return a post-quantum signer, got %T", s)
	assert.Equal(t, core.PQScheme_ML_DSA_44, pqs.PQScheme())
	assert.Equal(t, acct.Address, s.Address().String())

	passphrase = "wrong"
	s, err = resolveSigner(a)
	require.Error(t, err)
	assert.Nil(t, s)
	assert.True(t, errors.Is(err, keystore.ErrDecrypt), "got %v", err)
}

func TestFindAddressPQName(t *testing.T) {
	pqTestEnv(t, "pw")
	acct := savePQAccount(t, "pq-named", "pw")
	got, err := findAddress("pq-named")
	require.NoError(t, err)
	assert.Equal(t, acct.Address, got.String())

	_, err = findAddress("no-such-account")
	require.Error(t, err)
}

func TestKeysExportKSPQ(t *testing.T) {
	pqTestEnv(t, "pw")
	acct := savePQAccount(t, "pq-export", "pw")
	outDir := t.TempDir()

	out, _, err := runKeysCmd(t, keysExportKSCmd(), acct.Address, outDir)
	require.NoError(t, err)
	dst := filepath.Join(outDir, filepath.Base(acct.Path))
	assert.Contains(t, out, dst)
	want, err := os.ReadFile(acct.Path)
	require.NoError(t, err)
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	fi, err := os.Stat(dst)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), fi.Mode().Perm())

	// The exported file is a valid keystore for the same key.
	var clearFields struct{ Address, Scheme string }
	require.NoError(t, json.Unmarshal(got, &clearFields))
	assert.Equal(t, acct.Address, clearFields.Address)

	// Never overwrite an existing file.
	require.NoError(t, os.WriteFile(dst, []byte("keep"), 0600))
	_, _, err = runKeysCmd(t, keysExportKSCmd(), acct.Address, outDir)
	require.Error(t, err)
	kept, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "keep", string(kept))
}

func TestKeysExportPrivateKeyRefusesPQ(t *testing.T) {
	pqTestEnv(t, "pw")
	acct := savePQAccount(t, "pq-nohex", "pw")
	_, _, err := runKeysCmd(t, keysExportPKCmd(), acct.Address)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "export-ks")
}

func TestKeysRemovePQ(t *testing.T) {
	root := pqTestEnv(t, "pw")
	savePQAccount(t, "pq-gone", "pw")

	_, _, err := runKeysCmd(t, keysRemoveCmd(), "pq-gone")
	require.NoError(t, err)
	_, err = store.PQAccountByName("pq-gone")
	assert.True(t, errors.Is(err, store.ErrPQAccountNotFound), "got %v", err)
	assert.False(t, store.NameExists("pq-gone"))

	// A folder with no valid keystore is still removable.
	broken := filepath.Join(root, c.DefaultConfigPQKeysDirName, "pq-broken")
	require.NoError(t, os.MkdirAll(broken, 0700))
	_, _, err = runKeysCmd(t, keysRemoveCmd(), "pq-broken")
	require.NoError(t, err)
	assert.NoDirExists(t, broken)

	// ECDSA removal is unchanged.
	addECDSAAccount(t, "ecdsa-gone", "pw")
	_, _, err = runKeysCmd(t, keysRemoveCmd(), "ecdsa-gone")
	require.NoError(t, err)
	assert.False(t, store.DoesNamedAccountExist("ecdsa-gone"))
}

func TestParsePQScheme(t *testing.T) {
	tests := []struct {
		in      string
		want    core.PQScheme
		wantErr bool
	}{
		{in: "fn-dsa-512", want: core.PQScheme_FN_DSA_512},
		{in: "FN-DSA-512", want: core.PQScheme_FN_DSA_512},
		{in: "FN_DSA_512", want: core.PQScheme_FN_DSA_512},
		{in: "fn_dsa_512", want: core.PQScheme_FN_DSA_512},
		{in: "ml-dsa-44", want: core.PQScheme_ML_DSA_44},
		{in: "ML-DSA-44", want: core.PQScheme_ML_DSA_44},
		{in: "ML_DSA_44", want: core.PQScheme_ML_DSA_44},
		{in: "", wantErr: true},
		{in: "ecdsa", wantErr: true},
		{in: "ml-dsa-65", wantErr: true},
		{in: "fndsa512", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parsePQScheme(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "fn-dsa-512 and ml-dsa-44")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPQSigningKeyLifetime(t *testing.T) {
	pqTestEnv(t, "pw")
	acct := savePQAccount(t, "pq-life", "pw")
	var a tronAddress
	require.NoError(t, a.Set(acct.Address))

	s, err := resolveSigner(a)
	require.NoError(t, err)
	tx := &core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}}
	signed, err := s.Sign(tx)
	require.NoError(t, err, "the key must stay alive until the command ends")
	require.NotNil(t, signed)

	RootCmd.PersistentPostRun(RootCmd, nil)
	_, err = s.Sign(&core.Transaction{RawData: &core.TransactionRaw{Timestamp: 2}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, pqc.ErrKeyDestroyed), "got %v", err)

	// Idempotent.
	releaseSigningKeys()
}

func TestResolveSignerFNDSAWithoutFalcon(t *testing.T) {
	if pqc.SchemeAvailable(core.PQScheme_FN_DSA_512) == nil {
		t.Skip("built with a Falcon backend")
	}
	pqTestEnv(t, "pw")
	// An FN-DSA-512 account cannot be generated without Falcon; resolve
	// one directly through the helper with a stub account.
	_, err := resolvePQSigner(store.PQAccount{Name: "fn", Scheme: core.PQScheme_FN_DSA_512})
	require.Error(t, err)
	assert.True(t, errors.Is(err, pqc.ErrFalconUnavailable), "got %v", err)
	assert.Contains(t, err.Error(), "-tags falcon")
}

// TestPQImportRejectsInvalidNameBeforePrompt checks that import-toolkit and a
// PQ import-ks validate an explicit account name before asking for a
// passphrase. --passphrase is set and stdin is not a terminal, so reaching
// the prompt would fail with a terminal error instead.
func TestPQImportRejectsInvalidNameBeforePrompt(t *testing.T) {
	pqTestEnv(t, "pw")
	passphraseFilePath, userProvidesPassphrase = "", true

	k := newMLKey(t)
	toolkit, err := pqc.MarshalToolkitJSON(k)
	require.NoError(t, err)
	toolkitFile := writeTempFile(t, "toolkit.json", toolkit)
	ksFile, _ := mlKeystoreFile(t, "pw")

	tests := []struct {
		name string
		cmd  string
		file string
	}{
		{name: "import-toolkit", cmd: "import-toolkit", file: toolkitFile},
		{name: "import-ks post-quantum", cmd: "import-ks", file: ksFile},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := runKeysCmd(t, keysSubCmd(t, tt.cmd), tt.file, "../x")
			require.Error(t, err)
			assert.True(t, errors.Is(err, store.ErrInvalidAccountName), "got %v", err)
			assert.NotContains(t, out, "passphrase", "no passphrase prompt before the name check")
			assert.Empty(t, pqAccountNames(t))
		})
	}
}

// TestResolveSignerECDSAWhenPQStoreUnreadable checks that an unreadable
// pq-keys folder does not stop an ECDSA account from signing, and that when
// the ECDSA lookup fails too the error names both causes.
func TestResolveSignerECDSAWhenPQStoreUnreadable(t *testing.T) {
	root := pqTestEnv(t, "pw")
	ecdsaAddr := addECDSAAccount(t, "alice", "pw")
	// pq-keys as a plain file cannot be read as a directory.
	require.NoError(t, os.WriteFile(filepath.Join(root, c.DefaultConfigPQKeysDirName), []byte("x"), 0600))

	var a tronAddress
	require.NoError(t, a.Set(ecdsaAddr))
	s, err := resolveSigner(a)
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, ecdsaAddr, s.Address().String())

	var unknown tronAddress
	require.NoError(t, unknown.Set("TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"))
	s, err = resolveSigner(unknown)
	require.Error(t, err)
	assert.Nil(t, s)
	assert.Contains(t, err.Error(), "post-quantum")
	assert.Contains(t, err.Error(), c.DefaultConfigPQKeysDirName)
}

func TestAccountSignECDSAWhenPQStoreUnreadable(t *testing.T) {
	root := pqTestEnv(t, "pw")
	ecdsaAddr := addECDSAAccount(t, "alice", "pw")
	require.NoError(t, os.WriteFile(filepath.Join(root, c.DefaultConfigPQKeysDirName), []byte("x"), 0600))
	prevSigner := signerAddress
	t.Cleanup(func() { signerAddress = prevSigner })
	require.NoError(t, signerAddress.Set(ecdsaAddr))

	// account sign prints with fmt.Println; success here is "no error".
	_, _, err := runKeysCmd(t, accountSignCmd(), "hello")
	require.NoError(t, err, "an unreadable pq-keys folder must not block ECDSA message signing")
}
