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

var ()

func exchangeSub() []*cobra.Command {
	cmdCreate := &cobra.Command{
		Use:   "create <TOKEN1> <AMOUNT1> <TOKEN2> <AMOUNT2>",
		Short: "Check account balance",
		Long:  "Query for the latest account balance given Address",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			tokenID1 := args[0]
			// get amount
			tokenValue1, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return err
			}
			tokenID2 := args[2]
			// get amount
			tokenValue2, err := strconv.ParseFloat(args[3], 64)
			if err != nil {
				return err
			}
			if tokenID1 == tokenID2 {
				return fmt.Errorf("token ID cannot be the same")
			}
			if tokenValue1 <= 0 || tokenValue2 <= 0 {
				return fmt.Errorf("invalid token amount")
			}

			if tokenID1 == "TRX" || tokenID1 == "0" {
				tokenID1 = "_"
				tokenValue1 = tokenValue1 * math.Pow10(6)
			}
			if tokenID2 == "TRX" || tokenID2 == "0" {
				tokenID2 = "_"
				tokenValue2 = tokenValue2 * math.Pow10(6)
			}

			// Get asset information
			// check if possible id
			if tokenID1 != "_" {
				if asset, err := conn.GetAssetIssueByID(tokenID1); err == nil {
					tokenValue1 = tokenValue1 * math.Pow10(int(asset.Precision))
				} else {
					return fmt.Errorf("TRC10 not found: %s", tokenID1)
				}
			}
			if tokenID2 != "_" {
				if asset, err := conn.GetAssetIssueByID(tokenID2); err == nil {
					tokenValue2 = tokenValue2 * math.Pow10(int(asset.Precision))
				} else {
					return fmt.Errorf("TRC10 not found: %s", tokenID2)
				}
			}

			tx, err := conn.ExchangeCreate(
				signerAddress.String(),
				tokenID1,
				int64(tokenValue1),
				tokenID2,
				int64(tokenValue2),
			)
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
			if err = ctrlr.ExecuteTransaction(); err != nil {
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
	return []*cobra.Command{cmdCreate}
}

func init() {
	cmdExchange := &cobra.Command{
		Use:   "exchange",
		Short: "Bancos Exchange Actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdExchange.AddCommand(exchangeSub()...)
	RootCmd.AddCommand(cmdExchange)
}
