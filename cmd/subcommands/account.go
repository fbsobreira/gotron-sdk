package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
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
			valueInt := int64(value * math.Pow10(6))
			tx, err := conn.Transfer(signerAddress.String(), addr.String(), int64(valueInt))
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
			result["to"] = addr.String()
			result["amount"] = value
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

	cmdResources := &cobra.Command{
		Use:     "resources <ACCOUNT_NAME>",
		Short:   "Check account resources",
		Args:    cobra.ExactArgs(1),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			ar, err := conn.GetAccountResource(addr.String())
			if err != nil {
				return err
			}
			ard, err := conn.GetDelegatedResources(addr.String())
			if err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(ar, ard)
				return nil
			}

			result := make(map[string]interface{})
			result["address"] = addr.String()
			result["freeNetLimit"] = ar.GetFreeNetLimit()
			result["netLimit"] = ar.GetNetLimit()
			result["totalNetLimit"] = ar.GetTotalNetLimit()
			result["totalNetWeight"] = ar.GetTotalNetWeight()
			result["energyLimit"] = ar.GetEnergyLimit()
			result["totalEnergyLimit"] = ar.GetTotalEnergyLimit()
			result["totalEnergyWeight"] = ar.GetTotalEnergyWeight()

			delegated := make([]map[string]interface{}, 0)
			for _, d := range ard {
				for _, r := range d.DelegatedResource {
					data := make(map[string]interface{})
					data["from"] = address.Address(r.GetFrom()).String()
					data["to"] = address.Address(r.GetTo()).String()
					data["bw"] = float64(r.GetFrozenBalanceForBandwidth()) / 1000000
					data["energy"] = float64(r.GetFrozenBalanceForEnergy()) / 1000000
					data["bwExpire"] = r.GetExpireTimeForBandwidth()
					data["energyExpire"] = r.GetExpireTimeForEnergy()
					delegated = append(delegated, data)
				}
			}
			result["delegated"] = delegated

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdWithdraw := &cobra.Command{
		Use:   "withdraw",
		Short: "claim rewards",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			tx, err := conn.WithdrawBalance(signerAddress.String())
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
			result["address"] = addr.String()
			result["txID"] = common.BytesToHexString(tx.GetTxid())
			result["amount"] = addr.String()
			result["blockNumber"] = ctrlr.Receipt.BlockNumber
			result["message"] = string(ctrlr.Result.Message)
			result["amount"] = float64(ctrlr.Receipt.WithdrawAmount) / 1000000
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

	return []*cobra.Command{cmdBalance, cmdActivate, cmdSend, cmdAddress, cmdResources, cmdWithdraw}
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
