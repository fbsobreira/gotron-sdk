package cmd

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fatih/color"
	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	c "github.com/fbsobreira/gotron-sdk/pkg/common"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/fbsobreira/gotron-sdk/pkg/ledger"
	"github.com/fbsobreira/gotron-sdk/pkg/mnemonic"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
	"github.com/tyler-smith/go-bip39"
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

func keysSub() []*cobra.Command {
	cmdList := &cobra.Command{
		Use:   "list",
		Short: "List all the local accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if useLedgerWallet {
				ledger.ProcessAddressCommand()
				return nil
			}
			store.DescribeLocalAccounts()
			return nil
		},
	}

	cmdLocation := &cobra.Command{
		Use:   "location",
		Short: "Show where `tronctl` keeps accounts & their keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(store.DefaultLocation())
			return nil
		},
	}

	cmdAdd := &cobra.Command{
		Use:   "add <ACCOUNT_NAME>",
		Short: "Create a new keystore key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if store.DoesNamedAccountExist(args[0]) {
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
			addr, _ := store.AddressFromAccountName(acc.Name)
			fmt.Printf("Tron Address: %s\n", addr)
			return nil
		},
	}

	cmdRemove := &cobra.Command{
		Use:   "remove <ACCOUNT_NAME>",
		Short: "Remove a key from the keystore",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := account.RemoveAccount(args[0]); err != nil {
				return err
			}
			return nil
		},
	}

	cmdMnemonic := &cobra.Command{
		Use:   "mnemonic",
		Short: "Compute the bip39 mnemonic for some input entropy",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(mnemonic.Generate())
			return nil
		},
	}

	cmdRecoverMnemonic := &cobra.Command{
		Use:   "recover-from-mnemonic [ACCOUNT_NAME]",
		Short: "Recover account from mnemonic",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if store.DoesNamedAccountExist(args[0]) {
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
			fmt.Println("Enter mnemonic to recover keys from")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			m := scanner.Text()
			if !bip39.IsMnemonicValid(m) {
				return fmt.Errorf("invalid mnemonic given")
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
			addr, _ := store.AddressFromAccountName(acc.Name)
			fmt.Printf("Tron Address: %s\n", addr)
			return nil
		},
	}

	cmdImportKS := &cobra.Command{
		Use:   "import-ks <KEYSTORE_FILE_PATH> [ACCOUNT_NAME]",
		Args:  cobra.RangeArgs(1, 2),
		Short: "Import an existing keystore key",
		RunE: func(cmd *cobra.Command, args []string) error {
			userName := ""
			if len(args) == 2 {
				userName = args[1]
			}
			passphrase, err := getPassphrase()
			if err != nil {
				return err
			}
			name, err := account.ImportKeyStore(args[0], userName, passphrase)
			if !quietImport && err == nil {
				fmt.Printf("Imported keystore given account alias of `%s`\n", name)
				addr, _ := store.AddressFromAccountName(name)
				fmt.Printf("Tron Address: %s\n", addr)
			}
			return err
		},
	}
	cmdImportKS.Flags().BoolVar(&quietImport, "quiet", false, "do not print out imported account name")

	cmdImportPK := &cobra.Command{
		Use:   "import-private-key <secp256k1_PRIVATE_KEY> [ACCOUNT_NAME]",
		Short: "Import an existing keystore key (only accept secp256k1 private keys)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			userName := ""
			if len(args) == 2 {
				userName = args[1]
			}
			passphrase, err := getPassphrase()
			if err != nil {
				return err
			}
			name, err := account.ImportFromPrivateKey(args[0], userName, passphrase)
			if !quietImport && err == nil {
				fmt.Printf("Imported keystore given account alias of `%s`\n", name)
				addr, _ := store.AddressFromAccountName(name)
				fmt.Printf("Tron Address: %s\n", addr)
			}
			return err
		},
	}
	cmdImportPK.Flags().BoolVar(&quietImport, "quiet", false, "do not print out imported account name")

	cmdExportPK := &cobra.Command{
		Use:     "export-private-key <ACCOUNT_ADDRESS>",
		Short:   "Export the secp256k1 private key",
		Args:    cobra.ExactArgs(1),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			passphrase, err := getPassphrase()
			if err != nil {
				return err
			}
			return account.ExportPrivateKey(addr.address, passphrase)
		},
	}

	cmdExportKS := &cobra.Command{
		Use:     "export-ks <ACCOUNT_ADDRESS> <OUTPUT_DIRECTORY>",
		Short:   "Export the keystore file contents",
		Args:    cobra.ExactArgs(2),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
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

	randomPrivateKey := &cobra.Command{
		Use:   "random-pk",
		Short: "export a random private key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := bip39.NewEntropy(256)
			if err != nil {
				return err
			}
			fmt.Println(hex.EncodeToString(key))
			return nil
		},
	}

	addressFromPrivateKey := &cobra.Command{
		Use:   "address-pk",
		Short: "export address from private key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Enter privete key hex format:")
			data, err := terminal.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				return err
			}

			// decode hex
			privateKeyBytes, err := hex.DecodeString(string(data))
			if err != nil {
				return err
			}

			if len(privateKeyBytes) != common.Secp256k1PrivateKeyBytesLength {
				return common.ErrBadKeyLength
			}

			// btcec.PrivKeyFromBytes only returns a secret key and public key
			sk, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

			addr := address.PubkeyToAddress(*sk.PubKey().ToECDSA())
			fmt.Println(addr)
			return nil
		},
	}

	return []*cobra.Command{cmdList, cmdLocation, cmdAdd, cmdRemove, cmdMnemonic, cmdRecoverMnemonic, cmdImportKS, cmdImportPK,
		cmdExportKS, cmdExportPK, randomPrivateKey, addressFromPrivateKey}
}

func init() {
	cmdKeys := &cobra.Command{
		Use:   "keys",
		Short: "Add or view local private keys",
		Long:  "Manage your local keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdKeys.AddCommand(keysSub()...)
	RootCmd.AddCommand(cmdKeys)
}
