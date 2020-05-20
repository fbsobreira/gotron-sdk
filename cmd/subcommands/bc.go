package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/cobra"
)

var ()

func bcSub() []*cobra.Command {
	cmdNode := &cobra.Command{
		Use:   "node",
		Short: "get node metrics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := conn.GetNodeInfo()
			if err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(info)
				return nil
			}

			asJSON, _ := json.Marshal(info)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdMT := &cobra.Command{
		Use:   "mt",
		Short: "get network next maintainance time",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := conn.GetNextMaintenanceTime()
			if err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(info)
				return nil
			}

			t := time.Unix(info.GetNum()/1000, 0)
			result := make(map[string]interface{})
			result["nextTimestamp"] = info.GetNum()
			result["date"] = t.UTC().Format(time.RFC3339)

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	cmdTX := &cobra.Command{
		Use:   "tx <HASH>",
		Short: "get tx info by hash",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tx, err := conn.GetTransactionByID(args[0])
			if err != nil {
				return err
			}
			contracts := tx.GetRawData().GetContract()
			if len(contracts) != 1 {
				return fmt.Errorf("invalid contracts")
			}
			contract := contracts[0]

			info, err := conn.GetTransactionInfoByID(args[0])
			if err != nil {
				return err
			}

			if noPrettyOutput {
				fmt.Println(tx, info)
				return nil
			}

			result := make(map[string]interface{})
			t := time.Unix(info.GetBlockTimeStamp()/1000, 0)
			result["txID"] = common.BytesToHexString(info.Id)
			result["block"] = info.GetBlockNumber()
			result["timestamp"] = info.GetBlockTimeStamp()
			result["date"] = t.UTC().Format(time.RFC3339)

			result["receipt"] = map[string]interface{}{
				"fee":               info.GetFee(),
				"energyFee":         info.GetReceipt().GetEnergyFee(),
				"energyUsage":       info.GetReceipt().GetEnergyUsage(),
				"originEnergyUsage": info.GetReceipt().GetOriginEnergyUsage(),
				"energyUsageTotal":  info.GetReceipt().GetEnergyUsageTotal(),
				"netFee":            info.GetReceipt().GetNetFee(),
				"netUsage":          info.GetReceipt().GetNetUsage(),
			}

			result["contractName"] = contract.Type.String()
			//parse contract
			switch contract.Type {
			case core.Transaction_Contract_AccountCreateContract:
				var c core.AccountCreateContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_TransferContract:
				var c core.TransferContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_TransferAssetContract:
				var c core.TransferAssetContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_VoteWitnessContract:
				var c core.VoteWitnessContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_WitnessCreateContract:
				var c core.WitnessCreateContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_AssetIssueContract:
				var c core.AssetIssueContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_ParticipateAssetIssueContract:
				var c core.ParticipateAssetIssueContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_AccountUpdateContract:
				var c core.AccountUpdateContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_FreezeBalanceContract:
				var c core.FreezeBalanceContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_UnfreezeBalanceContract:
				var c core.UnfreezeBalanceContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_WithdrawBalanceContract:
				var c core.WithdrawBalanceContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_UnfreezeAssetContract:
				var c core.UnfreezeAssetContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_UpdateAssetContract:
				var c core.UpdateAssetContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)

			case core.Transaction_Contract_ProposalCreateContract:
				var c core.ProposalCreateContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_ProposalApproveContract:
				var c core.ProposalApproveContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_ProposalDeleteContract:
				var c core.ProposalDeleteContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_SetAccountIdContract:
				var c core.SetAccountIdContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_CustomContract:
				return fmt.Errorf("Tx inconsistent")
			case core.Transaction_Contract_CreateSmartContract:
				var c core.CreateSmartContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_TriggerSmartContract:
				var c core.TriggerSmartContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_UpdateSettingContract:
				var c core.UpdateSettingContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_ExchangeCreateContract:
				var c core.ExchangeCreateContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_ExchangeInjectContract:
				var c core.ExchangeInjectContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_ExchangeWithdrawContract:
				var c core.ExchangeWithdrawContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_ExchangeTransactionContract:
				var c core.ExchangeTransactionContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_UpdateEnergyLimitContract:
				var c core.UpdateEnergyLimitContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_AccountPermissionUpdateContract:
				var c core.AccountPermissionUpdateContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_ClearABIContract:
				var c core.ClearABIContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_UpdateBrokerageContract:
				var c core.UpdateBrokerageContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			case core.Transaction_Contract_ShieldedTransferContract:
				var c core.ShieldedTransferContract
				if err = ptypes.UnmarshalAny(contract.GetParameter(), &c); err != nil {
					return fmt.Errorf("Tx inconsistent")
				}
				result["contract"] = structs.Map(c)
			default:
				return fmt.Errorf("Tx inconsistent")
			}

			c := result["contract"].(map[string]interface{})
			delete(c, "XXX_NoUnkeyedLiteral")
			delete(c, "XXX_sizecache")
			delete(c, "XXX_unrecognized")
			if v, ok := c["OwnerAddress"]; ok && len(v.([]uint8)) > 0 {
				c["OwnerAddress"] = address.Address(v.([]uint8)).String()
			}
			if v, ok := c["ReceiverAddress"]; ok && len(v.([]uint8)) > 0 {
				c["ReceiverAddress"] = address.Address(v.([]uint8)).String()
			}
			if v, ok := c["ToAddress"]; ok && len(v.([]uint8)) > 0 {
				c["ToAddress"] = address.Address(v.([]uint8)).String()
			}

			if v, ok := c["Votes"]; ok {
				votes := make(map[string]int64)
				for _, d := range v.([]interface{}) {
					dP := d.(map[string]interface{})
					votes[address.Address(dP["VoteAddress"].([]uint8)).String()] = dP["VoteCount"].(int64)
				}
				c["Votes"] = votes
			}

			asJSON, _ := json.Marshal(result)
			fmt.Println(common.JSONPrettyFormat(string(asJSON)))
			return nil
		},
	}

	return []*cobra.Command{cmdNode, cmdMT, cmdTX}
}

func init() {
	cmdBC := &cobra.Command{
		Use:   "bc",
		Short: "Blockchain Actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdBC.AddCommand(bcSub()...)
	RootCmd.AddCommand(cmdBC)
}
