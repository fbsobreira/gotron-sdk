package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
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

	return []*cobra.Command{cmdNode, cmdMT}
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
