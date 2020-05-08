package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
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
				data := map[string]interface{}{
					"address":        address.Address(witness.Address).String(),
					"votes":          witness.VoteCount,
					"elected":        witness.IsJobs,
					"blocksMissed":   witness.TotalMissed,
					"blocksProduced": witness.TotalProduced,
					"productivity":   (float64(witness.TotalProduced) / float64(witness.TotalProduced+witness.TotalMissed)) * 100,
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

	return []*cobra.Command{cmdList}
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
