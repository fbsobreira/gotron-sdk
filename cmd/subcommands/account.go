package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
)

var (
	balanceDetails bool
)

func accountSub() []*cobra.Command {
	cmdBalance := &cobra.Command{
		Use:     "balance <ACCOUNT_NAME>",
		Short:   "Check account balance",
		Long:    "Query for the latest account balance given Address",
		Args:    cobra.ExactArgs(1),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			acc, err := conn.GetAccount(addr.String())
			if err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(acc)
				return nil
			}

			result := make(map[string]interface{})
			result["address"] = addr.String()
			result["type"] = acc.GetType()
			result["name"] = acc.GetAccountName()
			result["balance"] = float64(acc.GetBalance()) / 1000000
			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}
	cmdBalance.Flags().BoolVar(&balanceDetails, "details", false, "")

	cmdActivate := &cobra.Command{
		Use:     "activate <ADDRESS_TO_ACTIVATE>",
		Short:   "activate an address",
		Args:    cobra.ExactArgs(1),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}
			tx, err := conn.CreateAccount(signerAddress.String(), addr.String())
			if err != nil {
				return err
			}

			var ctrlr *transaction.Controller
			if useLedgerWallet {
				account := keystore.Account{Address: signerAddress.GetAddress()}
				ctrlr = transaction.NewController(conn, nil, &account, tx.Transaction, opts)
			} else {
				ks, acct, err := store.UnlockedKeystore(signerAddress.String(), passphrase)
				if err != nil {
					return err
				}
				ctrlr = transaction.NewController(conn, ks, acct, tx.Transaction, opts)
			}
			if err = ctrlr.ExecuteTransaction(0); err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(tx)
				return nil
			}

			result := make(map[string]interface{})
			result["txID"] = common.BytesToHexString(tx.GetTxid())
			result["blockNumber"] = ctrlr.Receipt.BlockNumber
			result["message"] = string(ctrlr.Result.Message)
			result["receipt"] = map[string]interface{}{
				"fee":      ctrlr.Receipt.Fee,
				"netFee":   ctrlr.Receipt.Receipt.NetFee,
				"netUsage": ctrlr.Receipt.Receipt.NetUsage,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdSend := &cobra.Command{
		Use:     "send <ADDRESS_TO> <AMOUNT>",
		Short:   "send TRX to an address",
		Args:    cobra.ExactArgs(2),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}
			// get amount
			value, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return err
			}
			value = value * math.Pow10(6)
			tx, err := conn.Transfer(signerAddress.String(), addr.String(), int64(value))
			if err != nil {
				return err
			}

			var ctrlr *transaction.Controller
			if useLedgerWallet {
				account := keystore.Account{Address: signerAddress.GetAddress()}
				ctrlr = transaction.NewController(conn, nil, &account, tx.Transaction, opts)
			} else {
				ks, acct, err := store.UnlockedKeystore(signerAddress.String(), passphrase)
				if err != nil {
					return err
				}
				ctrlr = transaction.NewController(conn, ks, acct, tx.Transaction, opts)
			}
			if err = ctrlr.ExecuteTransaction(0); err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(tx)
				return nil
			}

			result := make(map[string]interface{})
			result["txID"] = common.BytesToHexString(tx.GetTxid())
			result["blockNumber"] = ctrlr.Receipt.BlockNumber
			result["message"] = string(ctrlr.Result.Message)
			result["receipt"] = map[string]interface{}{
				"fee":      ctrlr.Receipt.Fee,
				"netFee":   ctrlr.Receipt.Receipt.NetFee,
				"netUsage": ctrlr.Receipt.Receipt.NetUsage,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdAddress := &cobra.Command{
		Use:   "address [ACC_NAME]",
		Short: "retrive address of account by name",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := ""
			if len(args) == 0 {
				if signerAddress.String() == "" {
					return fmt.Errorf("no signer or account name specified")
				}
				address = signerAddress.String()
			} else {
				if err := validateAddress(cmd, args); err != nil {
					return err
				}
				address = addr.String()
			}

			if noPrettyOutput {
				fmt.Println(address)
				return nil
			}

			result := make(map[string]interface{})
			result["address"] = address

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	return []*cobra.Command{cmdBalance, cmdActivate, cmdSend, cmdAddress}
}

func init() {
	cmdAccount := &cobra.Command{
		Use:   "account",
		Short: "Check account balance",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdAccount.AddCommand(accountSub()...)
	RootCmd.AddCommand(cmdAccount)
}
