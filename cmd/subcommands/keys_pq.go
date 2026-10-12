package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	pkgsigner "github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
)

// TIP-899 post-quantum (PQ) key support for the keys commands and signing.
// Experimental: TIP-899 is a draft enabled on Nile only. FN-DSA-512 keystores
// and Toolkit files parse in every build, but generating and signing with an
// FN-DSA-512 key needs a tronctl built with -tags falcon (CGO and liboqs), such
// as the tronctl-pq release archives.

const (
	falconHowToFix = "FN-DSA-512 needs a tronctl built with Falcon support: " +
		"download a tronctl-pq release archive, or build with " +
		"CGO_ENABLED=1 go build -tags falcon ./cmd/tronctl (requires liboqs)"
	pqExperimentalNote = "TIP-899 post-quantum keys are experimental (Nile only)"
	ecdsaSchemeLabel   = "ECDSA"
	// maxImportFileSize bounds how much of a keystore or Toolkit file is read.
	maxImportFileSize = 1 << 20
)

// pqScryptN and pqScryptP are the scrypt costs of keystores written by keys
// add --scheme and keys import-toolkit. Tests lower them.
var (
	pqScryptN = keystore.StandardScryptN
	pqScryptP = keystore.StandardScryptP
)

// parsePQScheme maps a --scheme value to its TIP-899 scheme. It accepts
// fn-dsa-512 and ml-dsa-44 in any case, with "-" or "_" as separators.
func parsePQScheme(s string) (core.PQScheme, error) {
	switch strings.ReplaceAll(strings.ToUpper(strings.TrimSpace(s)), "-", "_") {
	case core.PQScheme_FN_DSA_512.String():
		return core.PQScheme_FN_DSA_512, nil
	case core.PQScheme_ML_DSA_44.String():
		return core.PQScheme_ML_DSA_44, nil
	}
	return 0, fmt.Errorf("invalid --scheme %q: valid values are fn-dsa-512 and ml-dsa-44", s)
}

// withFalconHint adds how to get FN-DSA-512 support to an error wrapping
// pqc.ErrFalconUnavailable; errors.Is keeps matching it. Other errors are
// returned unchanged.
func withFalconHint(err error) error {
	if errors.Is(err, pqc.ErrFalconUnavailable) {
		return fmt.Errorf("%s: %w", falconHowToFix, err)
	}
	return err
}

// readImportFile reads a user-supplied keystore or key file, refusing one
// larger than maxImportFileSize.
func readImportFile(path string) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // the user names the file to import
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxImportFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxImportFileSize {
		clear(data)
		return nil, fmt.Errorf("%s is larger than %d bytes", path, maxImportFileSize)
	}
	return data, nil
}

// isPQKeystore reports whether a keystore JSON has a non-empty top-level
// "scheme" string, which marks a TIP-899 keystore. Ethereum V3 keystores have
// none; an absent, empty or null scheme means ECDSA, as in pkg/keystore.
func isPQKeystore(data []byte) bool {
	var peek struct {
		Scheme string `json:"scheme"`
	}
	if err := json.Unmarshal(data, &peek); err != nil {
		return false
	}
	return peek.Scheme != ""
}

// generateImportName returns a free "<word>-imported" account name, checked
// against both ECDSA and PQ accounts.
func generateImportName() (string, error) {
	for range 100 {
		m, err := mnemonic.Generate()
		if err != nil {
			return "", fmt.Errorf("generate account name: %w", err)
		}
		for _, w := range strings.Fields(m) {
			if name := w + "-imported"; !store.NameExists(name) {
				return name, nil
			}
		}
	}
	return "", errors.New("generate account name: no free name found")
}

func accountExistsError(name string) error {
	return fmt.Errorf("account %s already exists", name)
}

// checkNewAccountName fails when name is not a valid account name or is
// already taken by an ECDSA or a PQ account.
func checkNewAccountName(name string) error {
	if err := store.ValidateAccountName(name); err != nil {
		return err
	}
	if store.DoesNamedAccountExist(name) || store.NameExists(name) {
		return accountExistsError(name)
	}
	return nil
}

// printPQAccount prints the details of a stored PQ account.
func printPQAccount(w io.Writer, acct store.PQAccount) {
	_, _ = fmt.Fprintf(w, "Scheme: %s\n", acct.Scheme)
	_, _ = fmt.Fprintf(w, "Tron Address: %s\n", acct.Address)
	_, _ = fmt.Fprintf(w, "Keystore: %s\n", acct.Path)
}

// addPQKey generates a PQ key of scheme and stores it under name.
func addPQKey(cmd *cobra.Command, name string, scheme core.PQScheme) error {
	if store.NameExists(name) {
		return accountExistsError(name)
	}
	// Generate before prompting, so a build without Falcon fails at once.
	k, err := pqc.GenerateKey(scheme, nil)
	if err != nil {
		return withFalconHint(err)
	}
	defer k.Destroy()
	pass, err := getPassphraseWithConfirm()
	if err != nil {
		return err
	}
	acct, err := store.SavePQKey(name, k, pass, pqScryptN, pqScryptP)
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "Name: %s\n", acct.Name)
	printPQAccount(w, acct)
	_, _ = color.New(color.FgRed).Fprintf(w,
		"**Important** post-quantum keys have no mnemonic: the keystore file %s is the only backup "+
			"of this key. Copy it to a safe place; losing it or its passphrase loses the account.\n", acct.Path)
	_, _ = fmt.Fprintln(w, pqExperimentalNote)
	return nil
}

// importPQKeystore stores a TIP-899 keystore under name, or under a
// generated name when name is empty.
func importPQKeystore(cmd *cobra.Command, data []byte, name string) error {
	// Validate the name and check both stores before prompting, so a bad
	// name is not rejected only after the passphrase was typed.
	if name != "" {
		if err := checkNewAccountName(name); err != nil {
			return err
		}
	}
	pass, err := getPassphrase()
	if err != nil {
		return err
	}
	if name == "" {
		if name, err = generateImportName(); err != nil {
			return err
		}
	}
	acct, err := store.ImportPQKeystore(name, data, pass)
	if err != nil {
		return err
	}
	if !quietImport {
		w := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(w, "Imported keystore given account alias of `%s`\n", acct.Name)
		printPQAccount(w, acct)
	}
	return nil
}

// importToolkitKey encrypts the key in a TRON Toolkit plaintext key file and
// stores it under name, or under a generated name when name is empty.
func importToolkitKey(cmd *cobra.Command, path, name string) error {
	data, err := readImportFile(path)
	if err != nil {
		return err
	}
	k, err := pqc.ParseToolkitJSON(data)
	clear(data) // the file holds the private key in clear
	if err != nil {
		return err
	}
	defer k.Destroy()
	if name == "" {
		if name, err = generateImportName(); err != nil {
			return err
		}
	} else if err := checkNewAccountName(name); err != nil {
		// Checked before the passphrase prompt.
		return err
	}
	pass, err := getPassphraseWithConfirm()
	if err != nil {
		return err
	}
	acct, err := store.SavePQKey(name, k, pass, pqScryptN, pqScryptP)
	if err != nil {
		return err
	}
	if !quietImport {
		w := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(w, "Imported Toolkit key given account alias of `%s`\n", acct.Name)
		printPQAccount(w, acct)
	}
	_, _ = color.New(color.FgRed).Fprintf(cmd.ErrOrStderr(),
		"Warning: %s holds the private key in plaintext and is still on disk; "+
			"delete it securely once the new keystore is backed up\n", path)
	return nil
}

// writeAccountList prints ECDSA and PQ accounts in one table. NAME and
// ADDRESS keep the widths store.DescribeLocalAccounts uses; SCHEME is the
// last column so parsers of the old two-column output keep working.
// The ECDSA rows are printed before pq-keys is read, so an unreadable
// pq-keys folder still shows them; its error is returned afterwards.
func writeAccountList(w io.Writer) error {
	writeRow := func(name, address, scheme string) error {
		_, err := fmt.Fprintf(w, "%-48s\t%s\t%s\n", name, address, scheme)
		return err
	}
	if _, err := fmt.Fprintf(w, "%-24s\t\t%23s\t%s\n\n", "NAME", "ADDRESS", "SCHEME"); err != nil {
		return err
	}
	for _, name := range store.LocalAccounts() {
		ks := store.FromAccountName(name)
		accts := ks.Accounts()
		ks.Close()
		for _, a := range accts {
			if err := writeRow(name, a.Address.String(), ecdsaSchemeLabel); err != nil {
				return err
			}
		}
	}
	pqAccts, err := store.PQAccounts()
	if err != nil {
		return fmt.Errorf("list post-quantum accounts: %w", err)
	}
	for _, a := range pqAccts {
		if err := writeRow(a.Name, a.Address, a.Scheme.String()); err != nil {
			return err
		}
	}
	return nil
}

// lookupPQAccount returns the PQ account with address addr. found is false,
// with a nil error, when addr is not a PQ account.
func lookupPQAccount(addr string) (acct store.PQAccount, found bool, err error) {
	acct, err = store.PQAccountByAddress(addr)
	switch {
	case err == nil:
		return acct, true, nil
	case errors.Is(err, store.ErrPQAccountNotFound):
		return store.PQAccount{}, false, nil
	default:
		return store.PQAccount{}, false, err
	}
}

// exportPQKeystore copies acct's keystore file, unchanged, into dir with
// mode 0600 and returns the new path. An existing file is never overwritten.
func exportPQKeystore(acct store.PQAccount, dir string) (string, error) {
	dirPath, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(acct.Path)
	if err != nil {
		return "", fmt.Errorf("read post-quantum account %q: %w", acct.Name, err)
	}
	dst := filepath.Join(dirPath, filepath.Base(acct.Path))
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600) //nolint:gosec // the user names the output directory
	if err != nil {
		if os.IsExist(err) {
			return "", fmt.Errorf("%s already exists; refusing to overwrite it", dst)
		}
		return "", err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		if rmErr := os.Remove(dst); rmErr != nil {
			err = errors.Join(err, rmErr)
		}
		return "", fmt.Errorf("write %s: %w", dst, err)
	}
	return dst, nil
}

// refusePQPrivateKeyExport fails when addr is a PQ account: PQ private keys
// are only exported as encrypted keystores.
func refusePQPrivateKeyExport(addr string) error {
	acct, found, err := lookupPQAccount(addr)
	if err != nil {
		return err
	}
	if found {
		return fmt.Errorf("%s is a post-quantum (%s) account; its private key is not exported as hex. "+
			"Use `tronctl keys export-ks %s <OUTPUT_DIRECTORY>` to export the encrypted keystore",
			addr, acct.Scheme, addr)
	}
	return nil
}

// ecdsaNamePresent reports whether account-keys has an entry called name. It
// matches names the way the filesystem does (case-insensitively on macOS and
// Windows), like RemovePQAccount, so the collision check below cannot be
// bypassed by typing a name in a different case.
func ecdsaNamePresent(name string) (bool, error) {
	_, err := os.Lstat(filepath.Join(store.DefaultLocation(), name))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// pqNamePresent reports whether pq-keys has an entry called name. An entry
// whose keystore is unreadable, or that is not a directory, still counts as
// present. Any other error is returned, since the name cannot be shown to be
// free.
func pqNamePresent(name string) (bool, error) {
	_, err := store.PQAccountByName(name)
	switch {
	case err == nil, errors.Is(err, store.ErrInvalidPQKeystore):
		return true, nil
	case errors.Is(err, store.ErrPQAccountNotFound), errors.Is(err, store.ErrInvalidAccountName):
		return false, nil
	default:
		return false, err
	}
}

// removeAccountByName removes the PQ account name, or else the ECDSA one.
// A PQ folder whose keystore is unreadable is still removed. A name taken by
// both an ECDSA and a PQ account (left by an older build or an SDK caller)
// is refused and neither is removed, since the user cannot say which is meant.
func removeAccountByName(name string, removeECDSA func(string) error) error {
	if store.ValidateAccountName(name) == nil {
		inECDSA, err := ecdsaNamePresent(name)
		if err != nil {
			return fmt.Errorf("check ECDSA account %q: %w; nothing was removed", name, err)
		}
		inPQ, err := pqNamePresent(name)
		if err != nil {
			return fmt.Errorf("check post-quantum account %q: %w; nothing was removed", name, err)
		}
		if inECDSA && inPQ {
			return fmt.Errorf("%q names both an ECDSA and a post-quantum account; nothing was removed. "+
				"Move one of the two %q entries out of account-keys/ or pq-keys/ by hand, then retry", name, name)
		}
	}
	err := store.RemovePQAccount(name)
	if err == nil {
		return nil
	}
	if !errors.Is(err, store.ErrPQAccountNotFound) && !errors.Is(err, store.ErrInvalidAccountName) {
		return err
	}
	return removeECDSA(name)
}

// resolvePQSigner decrypts acct with the global passphrase and returns a
// signer for it. The key is kept until releaseSigningKeys.
func resolvePQSigner(acct store.PQAccount) (pkgsigner.Signer, error) {
	if err := pqc.SchemeAvailable(acct.Scheme); err != nil {
		return nil, withFalconHint(err)
	}
	k, err := store.DecryptPQKey(acct, passphrase)
	if err != nil {
		return nil, err
	}
	s, err := pkgsigner.NewPQSigner(k)
	if err != nil {
		k.Destroy()
		return nil, withFalconHint(err)
	}
	keepPQSigningKey(k)
	return s, nil
}

func keysImportToolkitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-toolkit <TOOLKIT_KEY_FILE> [ACCOUNT_NAME]",
		Short: "Import a TRON Toolkit plaintext post-quantum key file (experimental, Nile only)",
		Long: "Import the plaintext JSON key file written by java-tron's `Toolkit.jar pq-key new` " +
			"(FN-DSA-512 or ML-DSA-44) and store it as an encrypted keystore. " +
			"The source file is left on disk. " + pqExperimentalNote + ".",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := ""
			if len(args) == 2 {
				name = args[1]
			}
			return importToolkitKey(cmd, args[0], name)
		},
	}
	cmd.Flags().BoolVar(&quietImport, "quiet", false, "do not print out imported account name")
	return cmd
}
