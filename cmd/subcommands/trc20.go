package cmd

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/common/decimals"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
)

func trc20Sub() []*cobra.Command {
	cmdSend := &cobra.Command{
		Use:     "send <ADDRESS_TO> <AMOUNT> <CONTRACT_ADDRESS> ",
		Short:   "send TRC20 tokens to an address",
		Args:    cobra.ExactArgs(3),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}
			// get amount
			value, ok := decimals.FromString(args[1])
			if !ok {
				return fmt.Errorf("cannot parse value %s", args[1])
			}

			// get contract address
			contract, err := findAddress(args[2])
			if err != nil {
				return err
			}
			// get contract decimals if any
			tokenDecimals, err := conn.TRC20GetDecimals(contract.String())
			if err != nil {
				tokenDecimals = big.NewInt(0)
			}

			amount, _ := decimals.ApplyDecimals(value, tokenDecimals.Int64())
			tx, err := conn.TRC20Send(signerAddress.String(), addr.String(), contract.String(), amount, feeLimit)
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

			addrResult := address.Address(ctrlr.Receipt.ContractAddress).String()

			result := make(map[string]interface{})
			result["txID"] = common.BytesToHexString(tx.GetTxid())
			result["blockNumber"] = ctrlr.Receipt.BlockNumber
			result["message"] = string(ctrlr.Result.Message)
			result["contractAddress"] = addrResult
			result["success"] = ctrlr.GetResultError() == nil
			result["resMessage"] = string(ctrlr.Receipt.ResMessage)
			result["receipt"] = map[string]interface{}{
				"fee":               ctrlr.Receipt.Fee,
				"energyFee":         ctrlr.Receipt.Receipt.EnergyFee,
				"energyUsage":       ctrlr.Receipt.Receipt.EnergyUsage,
				"originEnergyUsage": ctrlr.Receipt.Receipt.OriginEnergyUsage,
				"energyUsageTotal":  ctrlr.Receipt.Receipt.EnergyUsageTotal,
				"netFee":            ctrlr.Receipt.Receipt.NetFee,
				"netUsage":          ctrlr.Receipt.Receipt.NetUsage,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdBalance := &cobra.Command{
		Use:     "balance <ADDRESS_TO> <CONTRACT_ADDRESS> ",
		Short:   "get TRC20 balance from contract",
		Args:    cobra.ExactArgs(2),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			// get contract address
			contract, err := findAddress(args[1])
			if err != nil {
				return err
			}

			// get contract decimals if any
			tokenDecimals, err := conn.TRC20GetDecimals(contract.String())
			if err != nil {
				tokenDecimals = big.NewInt(0)
			}

			// get contract decimals if any
			symbol, err := conn.TRC20GetSymbol(contract.String())
			if err != nil {
				symbol = ""
			}

			value, err := conn.TRC20ContractBalance(addr.String(), contract.String())
			if err != nil {
				return err
			}

			amount := decimals.RemoveDecimals(value, tokenDecimals.Int64())

			if noPrettyOutput {
				fmt.Println(amount.String())
				return nil
			}

			result := make(map[string]interface{})
			result["balance"] = fmt.Sprintf("%s %s", amount.String(), symbol)

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	return []*cobra.Command{cmdSend, cmdBalance}
}

func init() {
	cmdTrc20 := &cobra.Command{
		Use:   "trc20",
		Short: "TRC20 Manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdTrc20.AddCommand(trc20Sub()...)
	RootCmd.AddCommand(cmdTrc20)
}
