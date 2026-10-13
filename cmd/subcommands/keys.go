package cmd

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/ledger"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	seedPhraseWarning = "**Important** write this seed phrase in a safe place, " +
		"it is the only way to recover your account if you ever forget your password\n\n"
)

var (
	quietImport         bool
	recoverFromMnemonic bool
	passphrase          string
	ppPrompt            = fmt.Sprintf(
		"prompt for passphrase, otherwise use default passphrase: \"`%s`\"", c.DefaultPassphrase,
	)
)

func keysListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all the local accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if useLedgerWallet {
				dev, err := ledger.OpenDevice()
				if err != nil {
					return err
				}
				defer func() { _ = dev.Close() }()
				addr, err := dev.GetAddress()
				if err != nil {
					return err
				}
				fmt.Printf("%-24s\t\t%23s\n", "NAME", "ADDRESS")
				// TODO: add Device.Name() to the ledger.Device interface to support
				// Nano X, Nano S Plus, Stax, and Flex models instead of hardcoding.
				fmt.Printf("%-48s\t%s\n", "Ledger Nano S", addr)
				return nil
			}
			return writeAccountListTo(cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	return cmd
}

func keysLocationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "location",
		Short: "Show where `tronctl` keeps accounts & their keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(store.DefaultLocation())
			return nil
		},
	}
	return cmd
}

func keysAddCmd() *cobra.Command {
	var scheme string
	cmd := &cobra.Command{
		Use:   "add <ACCOUNT_NAME>",
		Short: "Create a new keystore key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("scheme") {
				s, err := parsePQScheme(scheme)
				if err != nil {
					return err
				}
				return addPQKey(cmd, args[0], s)
			}
			if err := requireKeystoreDir(); err != nil {
				return err
			}
			if store.DoesNamedAccountExist(args[0]) || store.NameExists(args[0]) {
				return fmt.Errorf("account %s already exists", args[0])
			}
			passphrase, err := getPassphraseWithConfirm()
			if err != nil {
				return err
			}
			acc := account.Creation{
				Name:       args[0],
				Passphrase: passphrase,
			}

			if err := account.CreateNewLocalAccount(&acc); err != nil {
				return err
			}
			if !recoverFromMnemonic {
				color.Red(seedPhraseWarning)
				fmt.Println(acc.Mnemonic)
			}
			addr, err := store.AddressFromAccountName(acc.Name)
			if err != nil {
				return err
			}
			fmt.Printf("Tron Address: %s\n", addr)
			return nil
		},
	}
	cmd.Flags().StringVar(&scheme, "scheme", "",
		"create a TIP-899 post-quantum key instead of ECDSA: fn-dsa-512 or ml-dsa-44 "+
			"(experimental, Nile only; fn-dsa-512 needs a tronctl-pq release build or a tronctl built with -tags falcon)")
	return cmd
}

func keysRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <ACCOUNT_NAME>",
		Short: "Remove a key from the keystore",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return removeAccountByName(args[0], account.RemoveAccount)
		},
	}
	return cmd
}

func keysMnemonicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mnemonic",
		Short: "Compute the bip39 mnemonic for some input entropy",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := mnemonic.Generate()
			if err != nil {
				return err
			}
			fmt.Println(m)
			return nil
		},
	}
	return cmd
}

func keysRecoverMnemonicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recover-from-mnemonic <ACCOUNT_NAME>",
		Short: "Recover account from mnemonic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Account names are unique across ECDSA and PQ accounts.
			if err := checkNewAccountName(args[0]); err != nil {
				return err
			}
			passphrase, err := getPassphraseWithConfirm()
			if err != nil {
				return err
			}
			acc := account.Creation{
				Name:       args[0],
				Passphrase: passphrase,
			}
			fmt.Println("Enter mnemonic to recover keys from")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			m := scanner.Text()
			if err := mnemonic.Validate(m); err != nil {
				return err
			}

			fmt.Println("Enter mnemonic password [optional]")
			scanner.Scan()
			p := scanner.Text()

			acc.Mnemonic = m
			acc.MnemonicPassphrase = p

			if err := account.CreateNewLocalAccount(&acc); err != nil {
				return err
			}
			fmt.Println("Successfully recovered account from mnemonic!")
			addr, err := store.AddressFromAccountName(acc.Name)
			if err != nil {
				return err
			}
			fmt.Printf("Tron Address: %s\n", addr)
			return nil
		},
	}
	return cmd
}

func keysImportKSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-ks <KEYSTORE_FILE_PATH> [ACCOUNT_NAME]",
		Args:  cobra.RangeArgs(1, 2),
		Short: "Import an existing keystore key",
		RunE: func(cmd *cobra.Command, args []string) error {
			userName := ""
			if len(args) == 2 {
				userName = args[1]
			}
			data, err := readImportFile(args[0])
			if err != nil {
				return err
			}
			if isPQKeystore(data) {
				return importPQKeystore(cmd, data, userName)
			}
			if err := requireKeystoreDir(); err != nil {
				return err
			}
			// Account names are unique across ECDSA and PQ accounts.
			if userName == "" {
				if userName, err = generateImportName(); err != nil {
					return err
				}
			} else if store.NameExists(userName) {
				return fmt.Errorf("account %s already exists", userName)
			}
			passphrase, err := getPassphrase()
			if err != nil {
				return err
			}
			name, err := account.ImportKeyStore(args[0], userName, passphrase)
			if !quietImport && err == nil {
				fmt.Printf("Imported keystore given account alias of `%s`\n", name)
				addr, err := store.AddressFromAccountName(name)
				if err != nil {
					return err
				}
				fmt.Printf("Tron Address: %s\n", addr)
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&quietImport, "quiet", false, "do not print out imported account name")
	return cmd
}

func keysImportPKCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-private-key <secp256k1_PRIVATE_KEY> [ACCOUNT_NAME]",
		Short: "Import an existing keystore key (only accept secp256k1 private keys)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Account names are unique across ECDSA and PQ accounts, so
			// pick or check the name here: account.ImportFromPrivateKey
			// only sees ECDSA accounts.
			var userName string
			if len(args) == 2 {
				userName = args[1]
				if err := checkNewAccountName(userName); err != nil {
					return err
				}
			} else {
				var err error
				if userName, err = generateImportName(); err != nil {
					return err
				}
			}
			passphrase, err := getPassphrase()
			if err != nil {
				return err
			}
			name, err := account.ImportFromPrivateKey(args[0], userName, passphrase)
			if !quietImport && err == nil {
				fmt.Printf("Imported keystore given account alias of `%s`\n", name)
				addr, err := store.AddressFromAccountName(name)
				if err != nil {
					return err
				}
				fmt.Printf("Tron Address: %s\n", addr)
			}
			return err
		},
	}
	cmd.Flags().BoolVar(&quietImport, "quiet", false, "do not print out imported account name")
	return cmd
}

func keysExportPKCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "export-private-key <ACCOUNT_ADDRESS>",
		Short:   "Export the secp256k1 private key",
		Args:    cobra.ExactArgs(1),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := refusePQPrivateKeyExport(addr.address); err != nil {
				return err
			}
			passphrase, err := getPassphrase()
			if err != nil {
				return err
			}
			return account.ExportPrivateKey(addr.address, passphrase)
		},
	}
	return cmd
}

func keysExportKSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "export-ks <ACCOUNT_ADDRESS> <OUTPUT_DIRECTORY>",
		Short:   "Export the keystore file contents",
		Args:    cobra.ExactArgs(2),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			pqAcct, isPQ, err := lookupPQAccount(addr.address)
			if err != nil {
				return err
			}
			if isPQ {
				file, err := exportPQKeystore(pqAcct, args[1])
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Exported keystore to", file)
				return nil
			}
			passphrase, err := getPassphrase()
			if err != nil {
				return err
			}
			file, e := account.ExportKeystore(addr.address, args[1], passphrase)
			if file != "" {
				fmt.Println("Exported keystore to", file)
			}
			return e
		},
	}
	return cmd
}

func keysRandomPKCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "random-pk",
		Short: "export a random private key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			privateKey, err := keys.GenerateKey()
			if err != nil {
				return err
			}
			fmt.Println(hex.EncodeToString(privateKey.Serialize()))
			return nil
		},
	}
	return cmd
}

func keysAddressPKCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address-pk",
		Short: "export address from private key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Enter private key hex format:")
			data, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}

			// decode hex
			privateKeyBytes, err := hex.DecodeString(string(data))
			if err != nil {
				return err
			}

			// Same checks as import-private-key: 32 bytes, in [1, n-1].
			sk, err := keys.GetPrivateKeyFromBytes(privateKeyBytes)
			if err != nil {
				return err
			}
			defer keys.ZeroPrivateKey(sk)

			addr := address.PubkeyToAddress(*sk.PubKey().ToECDSA())
			fmt.Println(addr)
			return nil
		},
	}
	return cmd
}

func keysSub() []*cobra.Command {
	return []*cobra.Command{
		keysListCmd(),
		keysLocationCmd(),
		keysAddCmd(),
		keysRemoveCmd(),
		keysMnemonicCmd(),
		keysRecoverMnemonicCmd(),
		keysImportKSCmd(),
		keysImportToolkitCmd(),
		keysImportPKCmd(),
		keysExportKSCmd(),
		keysExportPKCmd(),
		keysRandomPKCmd(),
		keysAddressPKCmd(),
	}
}

func init() {
	cmdKeys := &cobra.Command{
		Use:   "keys",
		Short: "Add or view local private keys",
		Long:  "Manage your local keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmdKeys.AddCommand(keysSub()...)
	RootCmd.AddCommand(cmdKeys)
}
