package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/store"

	"github.com/spf13/cobra"
)

var (
	abiSTR            string
	abiFile           string
	bcSTR             string
	bcFile            string
	feeLimit          int64
	curPercent        int64
	oeLimit           int64
	tAmount           float64
	tTokenID          string
	tTokenAmount      float64
	estimate          bool
	constructorParams string
)

func contractDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy <CONTRACT_NAME>",
		Short: "deploy smart contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			if abiSTR == "" {
				if abiFile != "" {
					abiBytes, err := os.ReadFile(abiFile)
					if err != nil {
						return fmt.Errorf("cannot read ABI file: %s %v", abiFile, err)
					}
					abiSTR = strings.TrimSpace(string(abiBytes))
				} else {
					return fmt.Errorf("no ABI string or ABI file specified")
				}
			}
			ABI, err := contract.JSONtoABI(abiSTR)
			if err != nil {
				return fmt.Errorf("cannot parse ABI: %v", err)
			}

			if bcSTR == "" {
				if bcFile != "" {
					bcBytes, err := os.ReadFile(bcFile)
					if err != nil {
						return fmt.Errorf("cannot read Bytecode file: %s %v", bcFile, err)
					}
					bcSTR = strings.TrimSpace(string(bcBytes))
				} else {
					return fmt.Errorf("no Bytecode string or Bytecode file specified")
				}
			}

			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			// Encode constructor arguments if provided
			if constructorParams != "" {
				found := false
				for _, entry := range ABI.Entrys {
					if entry.Type != core.SmartContract_ABI_Entry_Constructor {
						continue
					}
					found = true
					// Build constructor signature: constructor(type1,type2,...)
					types := make([]string, len(entry.Inputs))
					for i, input := range entry.Inputs {
						types[i] = input.Type
					}
					sig := fmt.Sprintf("constructor(%s)", strings.Join(types, ","))
					params, err := abi.LoadFromJSONWithMethod(sig, constructorParams)
					if err != nil {
						return fmt.Errorf("parse constructor params: %w", err)
					}
					encoded, err := abi.GetPaddedParam(params)
					if err != nil {
						return fmt.Errorf("encode constructor args: %w", err)
					}
					bcSTR += fmt.Sprintf("%x", encoded)
					break
				}
				if !found {
					return fmt.Errorf("--params provided but ABI has no constructor")
				}
			}

			tx, err := conn.DeployContract(signerAddress.String(), args[0],
				ABI, bcSTR, feeLimit, curPercent, oeLimit)
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

	cmd.Flags().StringVar(&abiSTR, "abi", "", "abi JSON string")
	cmd.Flags().StringVar(&abiFile, "abiFile", "", "abi file location")
	cmd.Flags().StringVar(&bcSTR, "bc", "", "bytecode HEX string")
	cmd.Flags().StringVar(&bcFile, "bcFile", "", "bytecode file location")
	cmd.Flags().StringVar(&constructorParams, "params", "", "constructor parameters as JSON (e.g. '[1000000]')")
	cmd.Flags().Int64Var(&feeLimit, "feeLimit", 1000000000, "fee limit")
	cmd.Flags().Int64Var(&curPercent, "curPercent", 100, "consume user resource percentage")
	cmd.Flags().Int64Var(&oeLimit, "oeLimit", 1000000, "origin energy limit")

	return cmd
}

func contractConstantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "constant <CONTRACT_ADDRESS> <METHOD> [PARAMETER]",
		Short:   "constant (read-only) contract call",
		Args:    cobra.RangeArgs(2, 3),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			from := signerAddress.String()

			param := ""
			if len(args) == 3 {
				param = args[2]
			}

			tx, err := conn.TriggerConstantContract(
				from,
				addr.String(),
				args[1],
				param,
			)
			if err != nil {
				return err
			}

			cResult := tx.GetConstantResult()

			if noPrettyOutput {
				fmt.Println(cResult)
				return nil
			}

			result := make(map[string]interface{})
			if len(cResult) == 0 {
				result["Result"] = ""
			} else {
				result["Result"] = common.BytesToHexString(cResult[0])

				// Try to auto-decode common return types
				if len(cResult[0]) >= 32 {
					decoded := tryDecodeResult(cResult[0])
					for k, v := range decoded {
						result[k] = v
					}
				}
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))

			return nil
		},
	}
	return cmd
}

// tryDecodeResult attempts to auto-decode ABI-encoded return data into
// human-readable values. It handles the most common return types:
// uint256, bool, string, and address.
func tryDecodeResult(data []byte) map[string]interface{} {
	result := make(map[string]interface{})

	if len(data) < 32 {
		return result
	}

	first32 := data[:32]
	val := new(big.Int).SetBytes(first32)

	// 1. Try ABI-encoded string first (offset=32 at first word)
	if len(data) >= 64 && val.IsUint64() && val.Uint64() == 32 {
		lengthWord := new(big.Int).SetBytes(data[32:64])
		if lengthWord.IsUint64() {
			strLen := lengthWord.Uint64()
			if strLen > 0 && strLen < 1024 && 64+strLen <= uint64(len(data)) {
				result["asString"] = string(data[64 : 64+strLen])
				return result
			}
		}
	}

	// 2. Try TRON address (first 12 bytes zero, 160-bit payload)
	allZero := true
	for i := 0; i < 12; i++ {
		if first32[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero && val.BitLen() > 64 && val.BitLen() <= 160 {
		evmAddr := first32[12:]
		tronAddr := make([]byte, 21)
		tronAddr[0] = 0x41
		copy(tronAddr[1:], evmAddr)
		result["asAddress"] = address.Address(tronAddr).String()
		return result
	}

	// 3. Try bool (0 or 1)
	if val.IsUint64() && (val.Uint64() == 0 || val.Uint64() == 1) {
		result["asBool"] = val.Uint64() == 1
	}

	// 4. Try number (full uint256)
	result["asNumber"] = val.String()

	return result
}

func contractTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "trigger <CONTRACT_ADDRESS> <METHOD> [PARAMETER]",
		Short:   "trigger smartcontract",
		Args:    cobra.RangeArgs(2, 3),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}
			// get amount
			valueInt := int64(0)
			if tAmount > 0 {
				valueInt = int64(tAmount * math.Pow10(6))
			}
			tokenInt := int64(0)
			if tTokenAmount > 0 {
				// get token info
				info, err := conn.GetAssetIssueByID(tTokenID)
				if err != nil {
					return err
				}
				tokenInt = int64(tAmount * math.Pow10(int(info.Precision)))
			}

			param := ""
			if len(args) == 3 {
				param = args[2]
			}

			if estimate {
				estimate, err := conn.EstimateEnergy(
					signerAddress.String(),
					addr.String(),
					args[1],
					param,
					valueInt,
					tTokenID,
					tokenInt,
				)

				if err != nil {
					return err
				}

				if noPrettyOutput {
					fmt.Println(estimate)
					return nil
				}

				result := make(map[string]interface{})
				result["EnergyRequired"] = estimate.EnergyRequired
				result["result"] = map[string]interface{}{
					"code":    estimate.Result.Code.String(),
					"message": string(estimate.Result.Message),
					"result":  estimate.Result.Result,
				}

				asJSON, _ := json.Marshal(result)
				fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			}

			tx, err := conn.TriggerContract(
				signerAddress.String(),
				addr.String(),
				args[1],
				param,
				feeLimit,
				valueInt,
				tTokenID,
				tokenInt,
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
	cmd.Flags().Float64Var(&tAmount, "value", 0, "trx amount")
	cmd.Flags().StringVar(&tTokenID, "token", "", "token id")
	cmd.Flags().Float64Var(&tTokenAmount, "tokenValue", 0, "token amount")
	cmd.Flags().BoolVar(&estimate, "estimate", false, "estimate energy required")
	return cmd
}

func contractSub() []*cobra.Command {
	return []*cobra.Command{
		contractDeployCmd(),
		contractConstantCmd(),
		contractTriggerCmd(),
	}
}

func init() {
	cmdContract := &cobra.Command{
		Use:   "contract",
		Short: "SmartContract actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmdContract.AddCommand(contractSub()...)
	RootCmd.AddCommand(cmdContract)
}
