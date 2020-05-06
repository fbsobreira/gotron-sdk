package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
)

var (
	issueStartDate string
	issueDuration  uint32
	issueFrozen    []string
	issueDecimals  int32
)

func trc10Sub() []*cobra.Command {
	cmdIssue := &cobra.Command{
		Use:   "issue <NAME> <DESCRIPTION> <SYMBOL> <URL> <TOTAL_SUPPLY> <RATIO>",
		Short: "Check account balance",
		Long:  "Query for the latest account balance given Address",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			trxNum := int64(1)
			tokenNum := int64(1)
			t, err := dateparse.ParseAny(issueStartDate)
			if err != nil {
				return err
			}
			if t.Before(time.Now()) {
				return fmt.Errorf("start date cannot be prior issue")
			}
			if issueDecimals > 6 || issueDecimals < 0 {
				return fmt.Errorf("decimals should be >= 0 &&  <= 6, found %d", issueDecimals)
			}
			if colon := strings.Index(args[5], ":"); colon > -1 {
				if trxNum, err = strconv.ParseInt(args[5][:colon], 10, 32); err != nil {
					return err
				}
				if tokenNum, err = strconv.ParseInt(args[5][colon+1:], 10, 32); err != nil {
					return err
				}
			} else {
				// use as float
				ratio, err := strconv.ParseFloat(args[5], 32)
				if err != nil {
					return err
				}
				// round up to 6 decimals
				p := math.Pow10(6)
				ratio = float64(int(ratio*p)) / p
				for float64(int64(ratio)) != ratio && tokenNum <= int64(math.Pow10(6)) {
					ratio *= 10
					tokenNum *= 10
				}
				if tokenNum > int64(math.Pow10(6)) {
					return fmt.Errorf("invalid ratio")
				}
				trxNum = int64(ratio)
			}

			frozenSupply := make(map[string]string)
			for _, value := range issueFrozen {
				frozenSupplyKeyValue := strings.Split(value, ":")
				if len(frozenSupplyKeyValue) != 2 {
					return fmt.Errorf("invalid frozen supply %s", frozenSupplyKeyValue)
				}
				if len(frozenSupply[frozenSupplyKeyValue[0]]) > 0 {
					return fmt.Errorf("frozen supply date colision %s:%s -> %s", frozenSupplyKeyValue[0], frozenSupply[frozenSupplyKeyValue[0]], value)
				}
				frozenSupply[frozenSupplyKeyValue[0]] = frozenSupplyKeyValue[1]
			}

			totalSupply, err := strconv.ParseInt(args[4], 10, 64)
			if err != nil {
				return err
			}

			tx, err := conn.AssetIssue(signerAddress.String(),
				args[0], // Name
				args[1], // Description
				args[2], // Symbol
				args[3], // URL
				issueDecimals,
				totalSupply,
				t.UTC().Unix()*1000,
				t.Add(time.Duration(issueDuration)*time.Hour*24).UTC().Unix()*1000,
				0, 0, //AssetLimit
				int32(trxNum),
				int32(tokenNum),
				0,            // Vote scores
				frozenSupply, // Frozen list
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
	// Asset issue extras
	cmdIssue.Flags().StringVar(&issueStartDate, "start", time.Now().Add(10*time.Minute).String(), "start time")
	cmdIssue.Flags().Uint32VarP(&issueDuration, "duration", "d", 1, "ico duration in days")
	cmdIssue.Flags().StringSliceVarP(&issueFrozen, "frozen", "0", []string{}, "frozen supply day1:amount1,day2:amount2")
	cmdIssue.Flags().Int32VarP(&issueDecimals, "decimals", "p", 0, "decimals precision (max 6)")

	return []*cobra.Command{cmdIssue}
}

func init() {
	cmdTrc10 := &cobra.Command{
		Use:   "trc10",
		Short: "Assets Manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdTrc10.AddCommand(trc10Sub()...)
	RootCmd.AddCommand(cmdTrc10)
}
