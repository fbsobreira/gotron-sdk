package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/store"

	"github.com/spf13/cobra"
)

var (
	abiSTR       string
	abiFile      string
	bcSTR        string
	bcFile       string
	feeLimit     int64
	curPercent   int64
	oeLimit      int64
	tAmount      float64
	tTokenID     string
	tTokenAmount float64
)

func contractSub() []*cobra.Command {
	cmdDeploy := &cobra.Command{
		Use:   "deploy <CONTRACT_NAME>",
		Short: "deploy smart contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {

			if abiSTR == "" {
				if abiFile != "" {
					abiBytes, err := ioutil.ReadFile(abiFile)
					if err != nil {
						return fmt.Errorf("cannot read ABI file: %s %v", abiFile, err)
					}
					abiSTR = string(abiBytes)
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
					bcBytes, err := ioutil.ReadFile(bcFile)
					if err != nil {
						return fmt.Errorf("cannot read Bytecode file: %s %v", bcFile, err)
					}
					bcSTR = string(bcBytes)
				} else {
					return fmt.Errorf("no Bytecode string or Bytecode file specified")
				}
			}

			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			// TODO: add constructor arguments
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

	cmdDeploy.Flags().StringVar(&abiSTR, "abi", "", "abi JSON string")
	cmdDeploy.Flags().StringVar(&abiFile, "abiFile", "", "abi file location")
	cmdDeploy.Flags().StringVar(&bcSTR, "bc", "", "bytecode HEX string")
	cmdDeploy.Flags().StringVar(&bcFile, "bcFile", "", "bytecode file location")
	cmdDeploy.Flags().Int64Var(&feeLimit, "feeLimit", 1000000000, "fee limit")
	cmdDeploy.Flags().Int64Var(&curPercent, "curPercent", 100, "consome user resource percentage")
	cmdDeploy.Flags().Int64Var(&oeLimit, "oeLimit", 1000000, "origin energy limit")

	cmdConstant := &cobra.Command{
		Use:     "constant <CONTRACT_ADDRESS> <METHOD> [PARAMETER]",
		Short:   "constantTrigger contract",
		Args:    cobra.RangeArgs(2, 3),
		PreRunE: validateAddress,
		RunE: func(cmd *cobra.Command, args []string) error {
			if signerAddress.String() == "" {
				return fmt.Errorf("no signer specified")
			}

			param := ""
			if len(args) == 3 {
				param = args[2]
			}

			tx, err := conn.TriggerConstantContract(
				signerAddress.String(),
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
			//TODO: parse based on contract ABI
			result["Result"] = common.ToHex(cResult[0])

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))

			return nil
		},
	}

	cmdTrigger := &cobra.Command{
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
	cmdTrigger.Flags().Int64Var(&feeLimit, "feeLimit", 10000000, "fee limit")
	cmdTrigger.Flags().Float64Var(&tAmount, "value", 0, "trx amount")
	cmdTrigger.Flags().StringVar(&tTokenID, "token", "", "token id")
	cmdTrigger.Flags().Float64Var(&tTokenAmount, "tokenValue", 0, "token amount")

	return []*cobra.Command{cmdDeploy, cmdConstant, cmdTrigger}
}

func init() {
	cmdContract := &cobra.Command{
		Use:   "contract",
		Short: "SmartContract actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdContract.AddCommand(contractSub()...)
	RootCmd.AddCommand(cmdContract)
}
