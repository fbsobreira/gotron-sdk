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

// tokenDecimalsFlag overrides the contract's decimals() lookup for the
// trc20 send command. Negative means "query the contract".
var tokenDecimalsFlag int64

// resolveTRC20Amount scales amount into token base units. A non-negative
// override is used as the token's decimals; otherwise lookup queries the
// contract. A wrong scale silently moves the wrong amount, so a failed lookup
// is fatal and points the user at --decimals.
func resolveTRC20Amount(amount string, override int64, lookup func() (*big.Int, error)) (*big.Int, error) {
	tokenDecimals := big.NewInt(override)
	if override < 0 {
		var err error
		tokenDecimals, err = lookup()
		if err != nil {
			return nil, fmt.Errorf("%w (pass --decimals to override for non-standard tokens)", err)
		}
	}
	if !tokenDecimals.IsInt64() || tokenDecimals.Sign() < 0 || tokenDecimals.Int64() > common.MaxTokenDecimals {
		return nil, fmt.Errorf("token decimals %s out of range 0..%d", tokenDecimals, common.MaxTokenDecimals)
	}
	n, err := common.ParseAmountBig(amount, int(tokenDecimals.Int64()))
	if err != nil {
		return nil, fmt.Errorf("AMOUNT: %w", err)
	}
	return n, nil
}

func trc20SendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "send <ADDRESS_TO> <AMOUNT> <CONTRACT_ADDRESS>",
		Short:   "send TRC20 tokens to an address",
		Args:    cobra.ExactArgs(3),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}
			// Validate the amount syntax before any RPC; it is scaled exactly
			// once the token's decimals are known.
			if _, err := common.ParseAmountBig(args[1], common.MaxTokenDecimals); err != nil {
				return fmt.Errorf("AMOUNT: %w", err)
			}

			// get contract address
			contract, err := findAddress(args[2])
			if err != nil {
				return err
			}
			amount, err := resolveTRC20Amount(args[1], tokenDecimalsFlag, func() (*big.Int, error) {
				d, err := conn.TRC20GetDecimals(contract.String())
				if err != nil {
					return nil, fmt.Errorf("cannot read decimals() for %s: %w", contract.String(), err)
				}
				return d, nil
			})
			if err != nil {
				return err
			}
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
	cmd.Flags().Int64Var(&feeLimit, "feeLimit", 10000000, "fee limit")
	cmd.Flags().Int64Var(&tokenDecimalsFlag, "decimals", -1,
		"token decimals override; default queries the contract's decimals()")
	return cmd
}

func trc20BalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
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
	return cmd
}

func trc20Sub() []*cobra.Command {
	return []*cobra.Command{
		trc20SendCmd(),
		trc20BalanceCmd(),
	}
}

func init() {
	cmdTrc20 := &cobra.Command{
		Use:   "trc20",
		Short: "TRC20 Manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmdTrc20.AddCommand(trc20Sub()...)
	RootCmd.AddCommand(cmdTrc20)
}
