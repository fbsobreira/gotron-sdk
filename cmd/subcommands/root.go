package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	color "github.com/fatih/color"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"golang.org/x/term"
)

var (
	addr                   tronAddress
	signer                 string
	signerAddress          tronAddress
	verbose                bool
	dryRun                 bool
	noWait                 bool
	useLedgerWallet        bool
	noPrettyOutput         bool
	userProvidesPassphrase bool
	passphraseFilePath     string
	defaultKeystoreDir     string
	node                   string
	givenFilePath          string
	timeout                uint32
	withTLS                bool
	apiKey                 string
	conn                   *client.GrpcClient
	// RootCmd is single entry point of the CLI
	RootCmd = &cobra.Command{
		Use:          "tronctl",
		Short:        "Tron Blockchain Controller",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// sync flag values into runtime
			rt.Verbose = verbose
			rt.Node = node
			rt.APIKey = apiKey
			rt.WithTLS = withTLS
			rt.NoPrettyOutput = noPrettyOutput
			rt.NoWait = noWait
			rt.DryRun = dryRun
			rt.Timeout = timeout
			rt.UseLedgerWallet = useLedgerWallet
			rt.GivenFilePath = givenFilePath
			rt.DefaultKeystoreDir = defaultKeystoreDir

			rt.loadDotEnv()
			rt.setupVerbose()
			signer = rt.applyEnvOverrides(config.Node, signer, withTLS)

			if err := rt.setupNetwork(); err != nil {
				return err
			}
			// sync back to legacy globals
			conn = rt.Conn

			if err := rt.setupSigner(signer); err != nil {
				return err
			}
			signerAddress = rt.SignerAddress

			var err error
			rt.Passphrase, err = getPassphrase()
			if err != nil {
				return err
			}
			passphrase = rt.Passphrase

			rt.setupKeystore()

			return nil
		},
		Long: fmt.Sprintf(`
CLI interface to Tron blockchain

%s`, g("type 'tronctl --help' for details")),
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
)

func init() {
	initConfig()

	vS := "dump out debug information, same as env var GOTRON_SDK_DEBUG=true"
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", config.Verbose, vS)
	RootCmd.PersistentFlags().StringVarP(&signer, "signer", "s", "", "<signer>")
	RootCmd.PersistentFlags().StringVarP(&node, "node", "n", config.Node, "<host>")
	RootCmd.PersistentFlags().StringVarP(&apiKey, "apiKey", "k", config.APIKey, "<api-key>")
	RootCmd.PersistentFlags().BoolVar(&withTLS, "withTLS", config.WithTLS, "<bool>")
	RootCmd.PersistentFlags().BoolVar(
		&noPrettyOutput, "no-pretty", config.NoPretty, "Disable pretty print JSON outputs",
	)
	RootCmd.PersistentFlags().BoolVar(&noWait, "no-wait", false, "do not wait for TX confirmation")
	RootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "do not send signed transaction")
	RootCmd.Flags().Uint32Var(&timeout, "timeout", config.Timeout, "set timeout in seconds. Set to 0 to not wait for confirm")

	RootCmd.PersistentFlags().BoolVarP(&useLedgerWallet, "ledger", "e", config.Ledger, "Use ledger hardware wallet")
	RootCmd.PersistentFlags().StringVar(&givenFilePath, "file", "", "Path to file for given command when applicable")

	// Password
	RootCmd.PersistentFlags().BoolVar(&userProvidesPassphrase, "passphrase", false, ppPrompt)
	RootCmd.PersistentFlags().StringVar(&passphraseFilePath, "passphrase-file", "", "path to a file containing the passphrase")
	RootCmd.PersistentFlags().StringVar(&defaultKeystoreDir, "ks-dir", "", "path to keystore")

	RootCmd.AddCommand(&cobra.Command{
		Use:   "docs",
		Short: fmt.Sprintf("Generate docs to a local %s directory", tronctlDocsDir),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			docDir := path.Join(cwd, tronctlDocsDir)
			err := os.Mkdir(docDir, 0700)
			if err != nil && !os.IsExist(err) {
				return fmt.Errorf("could not create %s directory: %v", tronctlDocsDir, err)
			}
			err = doc.GenMarkdownTree(RootCmd, docDir)
			return err
		},
	})
}

var (
	// VersionWrapDump meant to be set from main.go
	VersionWrapDump = ""
	versionLink     = "https://api.github.com/repos/fbsobreira/gotron-sdk/releases/latest"
	versionTagLink  = "https://api.github.com/repos/fbsobreira/gotron-sdk/git/ref/tags/"
)

const (
	// versionCheckTimeout bounds the whole version check. getGitVersion runs from
	// Execute on every command failure, so an unreachable or hung endpoint would
	// otherwise delay the user's real error indefinitely — offline use included.
	versionCheckTimeout = 3 * time.Second
	// shortCommitLen is the abbreviation width used for the local commit, by both
	// goreleaser's .ShortCommit and the build-info truncation in cmd/tronctl/main.go.
	shortCommitLen = 7
)

// GitHubReleaseAssets json struct
type GitHubReleaseAssets struct {
	ID   json.Number `json:"id"`
	Name string      `json:"name"`
	Size json.Number `json:"size"`
	URL  string      `json:"browser_download_url"`
}

// GitHubRelease json struct
type GitHubRelease struct {
	Prerelease      bool                  `json:"prerelease"`
	TagName         string                `json:"tag_name"`
	TargetCommitish string                `json:"target_commitish"`
	CreatedAt       time.Time             `json:"created_at"`
	Assets          []GitHubReleaseAssets `json:"assets"`
}

// GitHubTag json struct
type GitHubTag struct {
	Ref    string `json:"ref"`
	NodeID string `json:"node_id"`
	URL    string `json:"url"`
	DATA   struct {
		SHA  string `json:"sha"`
		Type string `json:"type"`
	} `json:"object"`
}

// fetchJSON performs a bounded GET and decodes a JSON body into out.
func fetchJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: unexpected status %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func isHexString(s string) bool {
	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}
	return true
}

// shortCommit returns the abbreviated commit hash that main appends to
// VersionWrapDump as "<version>-<commit>", or "" when this build carries no
// usable commit information.
//
// It scans back to front for the last hex field of at least shortCommitLen, so
// it stays correct for every shape the build actually produces: "v0.26.0-abc1234"
// from goreleaser, "v366-5484f0cc-dirty" from the Makefile, and prerelease tags
// like "v0.27.0-rc1-abc1234" where a fixed field index would pick up "rc1".
//
// It never indexes blindly: getGitVersion runs while reporting an ordinary error,
// so a panic here would replace the user's real message with a stack trace.
func shortCommit(dump string) string {
	fields := strings.Split(dump, "-")
	for i := len(fields) - 1; i >= 1; i-- {
		if len(fields[i]) >= shortCommitLen && isHexString(fields[i]) {
			return fields[i]
		}
	}
	return ""
}

func getGitVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()

	release := &GitHubRelease{}
	if err := fetchJSON(ctx, versionLink, release); err != nil {
		return "", fmt.Errorf("could not fetch version: %w", err)
	}

	releaseTag := &GitHubTag{}
	if err := fetchJSON(ctx, versionTagLink+release.TagName, releaseTag); err != nil {
		return "", fmt.Errorf("failed to fetch tag %s: %w", release.TagName, err)
	}

	// An annotated tag points at a tag object, not at a commit, so its sha is not
	// comparable with this build's commit. Report the tag rather than claiming the
	// build is outdated on a value that was never the commit. (Releases are tagged
	// lightweight today; this guards the day one is annotated.)
	if releaseTag.DATA.Type != "" && releaseTag.DATA.Type != "commit" {
		return release.TagName, nil
	}

	// No commit recorded in this build, or none reported by the API: there is
	// nothing to compare, so do not claim the build is outdated.
	localCommit := shortCommit(VersionWrapDump)
	if localCommit == "" || releaseTag.DATA.SHA == "" {
		return release.TagName, nil
	}

	// The local commit is an abbreviation while the API returns the full 40-char
	// sha, so compare by prefix. The previous sha[:8] != <7-char abbreviation>
	// could never be equal, which made every release build report itself outdated.
	if !strings.HasPrefix(releaseTag.DATA.SHA, localCommit) {
		warnMsg := fmt.Sprintf("Warning: Using outdated version. Redownload to upgrade to %s\n", release.TagName)
		fmt.Fprintf(os.Stderr, "%s", color.RedString(warnMsg))
		return release.TagName, fmt.Errorf("%s", warnMsg)
	}
	return release.TagName, nil
}

// Execute kicks off the tronctl CLI
func Execute() {
	RootCmd.SilenceErrors = true
	if err := RootCmd.Execute(); err != nil {
		if tag, errGit := getGitVersion(); errGit == nil {
			VersionWrapDump += ":" + tag
		}
		errMsg := fmt.Errorf("commit: %s, error: %w", VersionWrapDump, err).Error()
		fmt.Fprintf(os.Stderr, "%s\n", errMsg)
		fmt.Fprintf(os.Stderr, "try adding a `--help` flag\n")
		os.Exit(1)
	}
}

func validateAddress(cmd *cobra.Command, args []string) error {
	// Check if input valid one address
	var err error
	addr, err = findAddress(args[0])
	return err
}

func findAddress(value string) (tronAddress, error) {
	// Check if input valid one address
	address := tronAddress{}
	if err := address.Set(value); err != nil {
		// Check if input is valid account name
		if acc, err := store.AddressFromAccountName(value); err == nil {
			return tronAddress{acc}, nil
		}
		return address, fmt.Errorf("Invalid address/Invalid account name: %s", value)
	}
	return address, nil
}

func opts(ctlr *transaction.Controller) {
	if rt.DryRun {
		ctlr.Behavior.DryRun = true
	}
	if rt.UseLedgerWallet {
		ctlr.Behavior.SigningImpl = transaction.Ledger
	}
	if rt.NoWait {
		ctlr.Behavior.ConfirmationWaitTime = 0
	} else if rt.Timeout > 0 {
		ctlr.Behavior.ConfirmationWaitTime = rt.Timeout
	}
}

// getPassphrase fetches the correct passphrase depending on if a file is available to
// read from or if the user wants to enter in their own passphrase. Otherwise, just use
// the default passphrase. No confirmation of passphrase
func getPassphrase() (string, error) {
	if passphraseFilePath != "" {
		if _, err := os.Stat(passphraseFilePath); os.IsNotExist(err) {
			return "", fmt.Errorf("passphrase file not found at `%s`", passphraseFilePath)
		}
		dat, err := os.ReadFile(passphraseFilePath)
		if err != nil {
			return "", err
		}
		pw := strings.TrimSuffix(string(dat), "\n")
		return pw, nil
	} else if userProvidesPassphrase {
		fmt.Println("Enter passphrase:")
		pass, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		return string(pass), nil
	} else {
		return c.DefaultPassphrase, nil
	}
}

// getPassphraseWithConfirm fetches the correct passphrase depending on if a file is
// available to read from or if the user wants to enter in their own passphrase.
// Otherwise, just use the default passphrase. Passphrase requires a confirmation
func getPassphraseWithConfirm() (string, error) {
	if passphraseFilePath != "" {
		if _, err := os.Stat(passphraseFilePath); os.IsNotExist(err) {
			return "", fmt.Errorf("passphrase file not found at `%s`", passphraseFilePath)
		}
		dat, err := os.ReadFile(passphraseFilePath)
		if err != nil {
			return "", err
		}
		pw := strings.TrimSuffix(string(dat), "\n")
		return pw, nil
	} else if userProvidesPassphrase {
		fmt.Println("Enter passphrase:")
		pass, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		fmt.Println("Repeat the passphrase:")
		repeatPass, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		if string(repeatPass) != string(pass) {
			return "", errors.New("passphrase does not match")
		}
		fmt.Println("") // provide feedback when passphrase is entered.
		return string(repeatPass), nil
	} else {
		return c.DefaultPassphrase, nil
	}
}
