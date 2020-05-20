package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
)

var (
	expectedAmount float64
)

func exchangeSub() []*cobra.Command {
	cmdCreate := &cobra.Command{
		Use:   "create <TOKEN1> <AMOUNT1> <TOKEN2> <AMOUNT2>",
		Short: "Create bancor exchange for a token pair",
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

	cmdInject := &cobra.Command{
		Use:   "inject <EXCHANGE_ID> <TOKEN_ID> <AMOUNT>",
		Short: "inject tokens into bancor exchange",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			exchangeID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}

			tokenID1 := args[1]
			// get amount
			tokenValue1, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
			if tokenValue1 <= 0 {
				return fmt.Errorf("invalid token amount")
			}

			if tokenID1 == "TRX" || tokenID1 == "0" {
				tokenID1 = "_"
				tokenValue1 = tokenValue1 * math.Pow10(6)
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

			tx, err := conn.ExchangeInject(
				signerAddress.String(),
				exchangeID,
				tokenID1,
				int64(tokenValue1),
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
				"fee":          ctrlr.Receipt.Fee,
				"netFee":       ctrlr.Receipt.Receipt.NetFee,
				"netUsage":     ctrlr.Receipt.Receipt.NetUsage,
				"TokenAmount1": int64(tokenValue1),
				"TokenAmount2": ctrlr.Receipt.ExchangeInjectAnotherAmount,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdWithdraw := &cobra.Command{
		Use:   "withdraw <EXCHANGE_ID> <TOKEN_ID> <AMOUNT>",
		Short: "withdraw tokens from bancor exchange",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			exchangeID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}

			tokenID1 := args[1]
			// get amount
			tokenValue1, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
			if tokenValue1 <= 0 {
				return fmt.Errorf("invalid token amount")
			}

			if tokenID1 == "TRX" || tokenID1 == "0" {
				tokenID1 = "_"
				tokenValue1 = tokenValue1 * math.Pow10(6)
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

			tx, err := conn.ExchangeWithdraw(
				signerAddress.String(),
				exchangeID,
				tokenID1,
				int64(tokenValue1),
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
				"fee":          ctrlr.Receipt.Fee,
				"netFee":       ctrlr.Receipt.Receipt.NetFee,
				"netUsage":     ctrlr.Receipt.Receipt.NetUsage,
				"TokenAmount1": int64(tokenValue1),
				"TokenAmount2": ctrlr.Receipt.ExchangeWithdrawAnotherAmount,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdList := &cobra.Command{
		Use:   "list",
		Short: "List TRC10 bancor exchange",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {

			list, err := conn.ExchangeList(-1)
			if err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(list.Exchanges)
				return nil
			}

			result := make(map[string]interface{})
			result["total"] = len(list.Exchanges)
			result["list"] = make([]map[string]interface{}, 0)
			for _, e := range list.Exchanges {
				data := map[string]interface{}{
					"ID":            e.ExchangeId,
					"Owner":         address.Address(e.CreatorAddress).String(),
					"StartAt":       time.Unix(e.CreateTime/1000, 0),
					"Token1":        string(e.FirstTokenId),
					"Token1Balance": e.FirstTokenBalance,
					"Token2":        string(e.SecondTokenId),
					"Token2Balance": e.SecondTokenBalance,
				}
				result["list"] = append(result["list"].([]map[string]interface{}), data)
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdTrade := &cobra.Command{
		Use:   "trade <EXCHANGE_ID> <TOKEN_ID> <AMOUNT>",
		Short: "Trade token using TRC10 bancor exchange",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			exchangeID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return err
			}

			tokenID1 := args[1]
			// get amount
			tokenValue1, err := strconv.ParseFloat(args[2], 64)
			if err != nil {
				return err
			}
			if tokenValue1 <= 0 {
				return fmt.Errorf("invalid token amount")
			}

			if tokenID1 == "TRX" || tokenID1 == "0" {
				tokenID1 = "_"
				tokenValue1 = tokenValue1 * math.Pow10(6)
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

			// compute expected amount
			if e, err := conn.ExchangeByID(exchangeID); err == nil {
				tokenDecimal := 6
				T1 := string(e.FirstTokenId)
				T2 := string(e.SecondTokenId)
				ratio := float64(0)
				switch tokenID1 {
				case T1:
					if T2 != "_" {
						// get other token decimals
						if asset, err := conn.GetAssetIssueByID(T2); err == nil {
							tokenDecimal = int(asset.Precision)
						}
					}
					ratio = (float64(e.FirstTokenBalance) + tokenValue1) / float64(e.SecondTokenBalance)
				case T2:
					if T1 != "_" {
						// get other token decimals
						if asset, err := conn.GetAssetIssueByID(T1); err == nil {
							tokenDecimal = int(asset.Precision)
						}
					}
					ratio = (float64(e.SecondTokenBalance) + tokenValue1) / float64(e.FirstTokenBalance)
				default:
					return fmt.Errorf("Token ID provided does not match excahnge %s/%s", T1, T2)
				}
				if expectedAmount != 0 {
					expectedAmount = expectedAmount * math.Pow10(tokenDecimal)
				} else {
					expectedAmount = math.Floor(tokenValue1/ratio + 0.5)
				}
			} else {
				return fmt.Errorf("Cannot fetch echange info: %+v", err)
			}

			tx, err := conn.ExchangeTrade(
				signerAddress.String(),
				exchangeID,
				tokenID1,
				int64(tokenValue1),
				int64(expectedAmount),
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
				"fee":           ctrlr.Receipt.Fee,
				"netFee":        ctrlr.Receipt.Receipt.NetFee,
				"netUsage":      ctrlr.Receipt.Receipt.NetUsage,
				"TokenAmount1":  int64(tokenValue1),
				"TokenAmount2":  ctrlr.Receipt.ExchangeReceivedAmount,
				"TokenExpected": int64(expectedAmount),
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}
	cmdTrade.Flags().Float64VarP(&expectedAmount, "expected", "x", 0, "especify expected amount in return")

	return []*cobra.Command{cmdCreate, cmdInject, cmdWithdraw, cmdList, cmdTrade}
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
