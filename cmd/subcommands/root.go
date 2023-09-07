package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	color "github.com/fatih/color"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"golang.org/x/crypto/ssh/terminal"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	keyStoreDir            string
	givenFilePath          string
	timeout                uint32
	withTLS                bool
	apiKey                 string
	conn                   *client.GrpcClient
	// RootCmd is single entry point of the CLI
	RootCmd = &cobra.Command{
		Use:          "tronctl",
		Short:        "Tron Blokchain Controller ",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if verbose {
				common.EnableAllVerbose()
			}
			switch URLcomponents := strings.Split(node, ":"); len(URLcomponents) {
			case 1:
				node = node + ":50051"
			}
			conn = client.NewGrpcClient(node)

			// load grpc options
			opts := make([]grpc.DialOption, 0)
			if withTLS {
				opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
			} else {
				opts = append(opts, grpc.WithInsecure())
			}

			// check for env API Key
			if trongridKey := os.Getenv("TRONGRID_APIKEY"); len(trongridKey) > 0 {
				apiKey = trongridKey
			}
			// set API
			conn.SetAPIKey(apiKey)

			if err := conn.Start(opts...); err != nil {
				return err
			}

			if len(signer) > 0 {
				var err error
				if signerAddress, err = findAddress(signer); err != nil {
					return err
				}
			}

			var err error
			passphrase, err = getPassphrase()
			if err != nil {
				return err
			}

			if len(defaultKeystoreDir) > 0 {
				// set default directory
				store.SetDefaultLocation(defaultKeystoreDir)
			}

			return nil
		},
		Long: fmt.Sprintf(`
CLI interface to Tron blockchain

%s`, g("type 'tronclt --help' for details")),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
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
			os.Mkdir(docDir, 0700)
			err := doc.GenMarkdownTree(RootCmd, docDir)
			return err
		},
	})
}

var (
	// VersionWrapDump meant to be set from main.go
	VersionWrapDump = ""
	versionLink     = "https://api.github.com/repos/fbsobreira/gotron-sdk/releases/latest"
	versionTagLink  = "https://api.github.com/repos/fbsobreira/gotron-sdk/git/ref/tags/"
	versionFormat   = regexp.MustCompile("v[0-9]+-[a-z0-9]{7}")
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
		SHA string `json:"sha"`
	} `json:"object"`
}

func getGitVersion() (string, error) {
	resp, err := http.Get(versionLink)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	// if error, no op
	if resp != nil && resp.StatusCode == 200 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		release := &GitHubRelease{}
		if err := json.Unmarshal(buf.Bytes(), release); err != nil {
			return "", err
		}

		respTag, _ := http.Get(versionTagLink + release.TagName)
		defer resp.Body.Close()
		// if error, no op
		if respTag != nil && respTag.StatusCode == 200 {
			buf.Reset()
			buf.ReadFrom(respTag.Body)

			releaseTag := &GitHubTag{}
			if err := json.Unmarshal(buf.Bytes(), releaseTag); err != nil {
				return "", err
			}
			commit := strings.Split(VersionWrapDump, "-")

			if releaseTag.DATA.SHA[:8] != commit[1] {
				warnMsg := fmt.Sprintf("Warning: Using outdated version. Redownload to upgrade to %s\n", release.TagName)
				fmt.Fprintf(os.Stderr, color.RedString(warnMsg))
				return release.TagName, fmt.Errorf(warnMsg)
			}
			return release.TagName, nil
		}
	}
	return "", fmt.Errorf("could not fetch version")
}

// Execute kicks off the tronctl CLI
func Execute() {
	RootCmd.SilenceErrors = true
	if err := RootCmd.Execute(); err != nil {
		if tag, errGit := getGitVersion(); errGit == nil {
			VersionWrapDump += ":" + tag
		}
		errMsg := errors.Wrapf(err, "commit: %s, error", VersionWrapDump).Error()
		fmt.Fprintf(os.Stderr, errMsg+"\n")
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
	if dryRun {
		ctlr.Behavior.DryRun = true
	}
	if useLedgerWallet {
		ctlr.Behavior.SigningImpl = transaction.Ledger
	}
	if noWait {
		ctlr.Behavior.ConfirmationWaitTime = 0
	} else if timeout > 0 {
		ctlr.Behavior.ConfirmationWaitTime = timeout
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
		dat, err := ioutil.ReadFile(passphraseFilePath)
		if err != nil {
			return "", err
		}
		pw := strings.TrimSuffix(string(dat), "\n")
		return pw, nil
	} else if userProvidesPassphrase {
		fmt.Println("Enter passphrase:")
		pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		return string(pass), nil
	} else {
		return c.DefaultPassphrase, nil
	}
}

// getPassphrase fetches the correct passphrase depending on if a file is available to
// read from or if the user wants to enter in their own passphrase. Otherwise, just use
// the default passphrase. Passphrase requires a confirmation
func getPassphraseWithConfirm() (string, error) {
	if passphraseFilePath != "" {
		if _, err := os.Stat(passphraseFilePath); os.IsNotExist(err) {
			return "", fmt.Errorf("passphrase file not found at `%s`", passphraseFilePath)
		}
		dat, err := ioutil.ReadFile(passphraseFilePath)
		if err != nil {
			return "", err
		}
		pw := strings.TrimSuffix(string(dat), "\n")
		return pw, nil
	} else if userProvidesPassphrase {
		fmt.Println("Enter passphrase:")
		pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		fmt.Println("Repeat the passphrase:")
		repeatPass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
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
