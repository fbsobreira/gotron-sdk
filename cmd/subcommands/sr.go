package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
)

var (
	electedOnly bool
	brokerage   bool
)

func srSub() []*cobra.Command {
	cmdList := &cobra.Command{
		Use:   "list",
		Short: "List network witnesses",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			list, err := conn.ListWitnesses()
			if err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(list.Witnesses)
				return nil
			}

			result := make(map[string]interface{})

			wList := make([]map[string]interface{}, 0)
			for _, witness := range list.Witnesses {
				if electedOnly && !witness.IsJobs {
					continue
				}
				prod := float64(0)
				if witness.TotalProduced+witness.TotalMissed > 0 {
					prod = (float64(witness.TotalProduced) / float64(witness.TotalProduced+witness.TotalMissed)) * 100
				}
				data := map[string]interface{}{
					"address":        address.Address(witness.Address).String(),
					"votes":          witness.VoteCount,
					"elected":        witness.IsJobs,
					"blocksMissed":   witness.TotalMissed,
					"blocksProduced": witness.TotalProduced,
					"productivity":   prod,
					"url":            witness.Url,
				}
				if brokerage {
					value := float64(10)
					distType := "need withdraw"
					if data["address"].(string) == "TKSXDA8HfE9E1y39RczVQ1ZascUEtaSToF" {
						distType = "directly to wallet"
					} else {
						value, err = conn.GetWitnessBrokerage(data["address"].(string))
						if err != nil {
							return fmt.Errorf("fetching brokerage from %s", data["address"])
						}
					}
					data["brokerage"] = value
					data["distribution"] = 100 - value
					data["distribution"] = distType
				}
				wList = append(wList, data)
			}
			result["totalCount"] = len(list.Witnesses)
			result["filterCount"] = len(wList)
			result["witnesses"] = wList
			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}
	cmdList.Flags().BoolVar(&electedOnly, "elected", false, "if true return elected only")
	cmdList.Flags().BoolVar(&brokerage, "brokerage", false, "add brokerage result")

	cmdCreate := &cobra.Command{
		Use:   "create <URL>",
		Short: "create new SR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}
			tx, err := conn.CreateWitness(signerAddress.String(), args[0])
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
				fmt.Println(tx, ctrlr.Receipt, ctrlr.Result)
				return nil
			}

			result := make(map[string]interface{})
			result["from"] = signerAddress.String()
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

	cmdUpdateBrokerage := &cobra.Command{
		Use:   "brokerage <new_comission>",
		Short: "up SR brokerage comission",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}
			value, err := strconv.ParseInt(args[0], 10, 32)
			if err != nil {
				return err
			}
			if value < 0 || value > 100 {
				return fmt.Errorf("Invalud Brokerage rande 0 > X < 100")
			}
			tx, err := conn.UpdateBrokerage(signerAddress.String(), int32(value))
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
				fmt.Println(tx, ctrlr.Receipt, ctrlr.Result)
				return nil
			}

			result := make(map[string]interface{})
			result["from"] = signerAddress.String()
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

	return []*cobra.Command{cmdList, cmdCreate, cmdUpdateBrokerage}
}

func init() {
	cmdSR := &cobra.Command{
		Use:   "sr",
		Short: "SR Actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdSR.AddCommand(srSub()...)
	RootCmd.AddCommand(cmdSR)
}
