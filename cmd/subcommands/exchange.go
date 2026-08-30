package cmd

import (
	"encoding/json"
	"fmt"
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
	expectedAmount string
)

// parseExchangeTokenAmount scales a TRX or TRC10 amount into base units.
// "TRX" and "0" are normalized to the exchange's "_" token id.
func parseExchangeTokenAmount(tokenID, amount, argName string) (string, int64, error) {
	decimals := common.AmountDecimalPoint
	id := tokenID
	if tokenID == "TRX" || tokenID == "0" {
		id = "_"
	} else {
		asset, err := conn.GetAssetIssueByID(tokenID)
		if err != nil || asset == nil {
			return "", 0, fmt.Errorf("TRC10 not found: %s", tokenID)
		}
		decimals = int(asset.GetPrecision())
	}
	v, err := parseAmountArg(amount, argName, decimals)
	if err != nil {
		return "", 0, err
	}
	return id, v, nil
}

func exchangeCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <TOKEN1> <AMOUNT1> <TOKEN2> <AMOUNT2>",
		Short: "Create bancor exchange for a token pair",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			if args[0] == args[2] {
				return fmt.Errorf("token ID cannot be the same")
			}
			tokenID1, tokenValue1, err := parseExchangeTokenAmount(args[0], args[1], "AMOUNT1")
			if err != nil {
				return err
			}
			tokenID2, tokenValue2, err := parseExchangeTokenAmount(args[2], args[3], "AMOUNT2")
			if err != nil {
				return err
			}
			if tokenValue1 <= 0 || tokenValue2 <= 0 {
				return fmt.Errorf("invalid token amount")
			}

			tx, err := conn.ExchangeCreate(
				signerAddress.String(),
				tokenID1,
				tokenValue1,
				tokenID2,
				tokenValue2,
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
}

func exchangeInjectCmd() *cobra.Command {
	return &cobra.Command{
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

			tokenID1, tokenValue1, err := parseExchangeTokenAmount(args[1], args[2], "AMOUNT")
			if err != nil {
				return err
			}
			if tokenValue1 <= 0 {
				return fmt.Errorf("invalid token amount")
			}

			tx, err := conn.ExchangeInject(
				signerAddress.String(),
				exchangeID,
				tokenID1,
				tokenValue1,
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
				"TokenAmount1": tokenValue1,
				"TokenAmount2": ctrlr.Receipt.ExchangeInjectAnotherAmount,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}
}

func exchangeWithdrawCmd() *cobra.Command {
	return &cobra.Command{
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

			tokenID1, tokenValue1, err := parseExchangeTokenAmount(args[1], args[2], "AMOUNT")
			if err != nil {
				return err
			}
			if tokenValue1 <= 0 {
				return fmt.Errorf("invalid token amount")
			}

			tx, err := conn.ExchangeWithdraw(
				signerAddress.String(),
				exchangeID,
				tokenID1,
				tokenValue1,
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
				"TokenAmount1": tokenValue1,
				"TokenAmount2": ctrlr.Receipt.ExchangeWithdrawAnotherAmount,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}
}

func exchangeListCmd() *cobra.Command {
	return &cobra.Command{
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
}

func exchangeTradeCmd() *cobra.Command {
	cmd := &cobra.Command{
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

			tokenID1, tokenValue1, err := parseExchangeTokenAmount(args[1], args[2], "AMOUNT")
			if err != nil {
				return err
			}
			if tokenValue1 <= 0 {
				return fmt.Errorf("invalid token amount")
			}

			e, err := conn.ExchangeByID(exchangeID)
			if err != nil {
				return fmt.Errorf("Cannot fetch exchange info: %+v", err)
			}
			if e == nil {
				return fmt.Errorf("Cannot fetch exchange info: empty response")
			}
			T1 := string(e.GetFirstTokenId())
			T2 := string(e.GetSecondTokenId())
			var reserveIn, reserveOut int64
			// outID is the token received; only its precision can scale --expected.
			outID := ""
			switch tokenID1 {
			case T1:
				outID = T2
				reserveIn = e.GetFirstTokenBalance()
				reserveOut = e.GetSecondTokenBalance()
			case T2:
				outID = T1
				reserveIn = e.GetSecondTokenBalance()
				reserveOut = e.GetFirstTokenBalance()
			default:
				return fmt.Errorf("Token ID provided does not match exchange %s/%s", T1, T2)
			}

			var expectedInt int64
			if expectedAmount != "" && !isDecimalZero(expectedAmount) {
				// Resolve the received token's precision only when an explicit
				// --expected has to be scaled; the auto-quote below is already
				// in base units and must not depend on this lookup.
				tokenDecimal := common.AmountDecimalPoint
				if outID != "_" {
					asset, err := conn.GetAssetIssueByID(outID)
					if err != nil || asset == nil {
						return fmt.Errorf("TRC10 not found: %s", outID)
					}
					tokenDecimal = int(asset.GetPrecision())
				}
				expectedInt, err = parseAmountArg(expectedAmount, "--expected", tokenDecimal)
				if err != nil {
					return err
				}
			} else {
				expectedInt, err = roundBancorQuote(tokenValue1, reserveIn, reserveOut)
				if err != nil {
					return err
				}
			}

			tx, err := conn.ExchangeTrade(
				signerAddress.String(),
				exchangeID,
				tokenID1,
				tokenValue1,
				expectedInt,
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
				"TokenAmount1":  tokenValue1,
				"TokenAmount2":  ctrlr.Receipt.ExchangeReceivedAmount,
				"TokenExpected": expectedInt,
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}
	cmd.Flags().StringVarP(&expectedAmount, "expected", "x", "0", "specify expected amount in return")
	return cmd
}

func exchangeSub() []*cobra.Command {
	return []*cobra.Command{
		exchangeCreateCmd(),
		exchangeInjectCmd(),
		exchangeWithdrawCmd(),
		exchangeListCmd(),
		exchangeTradeCmd(),
	}
}

func init() {
	cmdExchange := &cobra.Command{
		Use:   "exchange",
		Short: "Bancos Exchange Actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmdExchange.AddCommand(exchangeSub()...)
	RootCmd.AddCommand(cmdExchange)
}
