package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	color "github.com/fatih/color"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

const defaultTimeout = 40

var (
	addr            tronAddress
	signer          string
	signerAddress   tronAddress
	verbose         bool
	dryRun          bool
	useLedgerWallet bool
	noPrettyOutput  bool
	node            string
	keyStoreDir     string
	givenFilePath   string
	timeout         uint32
	conn            *client.GrpcClient
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
			if err := conn.Start(); err != nil {
				return err
			}

			if len(signer) > 0 {
				var err error
				if signerAddress, err = findAddress(signer); err != nil {
					return err
				}
			}

			return nil
		},
		Long: fmt.Sprintf(`
CLI interface to the Tron blockchain

%s`, g("type 'tronclt --help' details")),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}
)

func init() {
	vS := "dump out debug information, same as env var GOTRON_SDK_DEBUG=true"
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, vS)
	RootCmd.PersistentFlags().StringVarP(&signer, "signer", "s", "", "<signer>")
	RootCmd.PersistentFlags().StringVarP(&node, "node", "n", defaultNodeAddr, "<host>")
	RootCmd.PersistentFlags().BoolVar(
		&noPrettyOutput, "no-pretty", false, "Disable pretty print JSON outputs",
	)
	RootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "do not send signed transaction")
	RootCmd.Flags().Uint32Var(&timeout, "timeout", defaultTimeout, "set timeout in seconds. Set to 0 to not wait for confirm")

	RootCmd.PersistentFlags().BoolVarP(&useLedgerWallet, "ledger", "e", false, "Use ledger hardware wallet")
	RootCmd.PersistentFlags().StringVar(&givenFilePath, "file", "", "Path to file for given command when applicable")
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
	versionLink     = "https://cryptochain.network/tronctl_ver"
	versionFormat   = regexp.MustCompile("v[0-9]+-[a-z0-9]{7}")
)

// Execute kicks off the tronctl CLI
func Execute() {
	RootCmd.SilenceErrors = true
	if err := RootCmd.Execute(); err != nil {
		resp, _ := http.Get(versionLink)
		defer resp.Body.Close()
		// If error, no op
		if resp != nil && resp.StatusCode == 200 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)

			currentVersion := versionFormat.FindAllString(buf.String(), 1)
			if currentVersion != nil && currentVersion[0] != VersionWrapDump {
				warnMsg := fmt.Sprintf("Warning: Using outdated version. Redownload to upgrade to %s\n", currentVersion[0])
				fmt.Fprintf(os.Stderr, color.RedString(warnMsg))
			}
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
		return address, fmt.Errorf("Invalid one address/Invalid account name: %s", value)
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
	if timeout > 0 {
		ctlr.Behavior.ConfirmationWaitTime = timeout
	}
}
