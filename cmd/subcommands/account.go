package cmd

import (
	"encoding/json"
	"fmt"

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
			result["receipt"] = map[string]interface{}{
				"message": string(ctrlr.Receipt.Message),
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	return []*cobra.Command{cmdBalance, cmdActivate}
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
