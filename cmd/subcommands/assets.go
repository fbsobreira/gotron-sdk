package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/spf13/cobra"
)

func assetsSub() []*cobra.Command {
	cmdIssue := &cobra.Command{
		Use:     "issue <ACCOUNT_NAME>",
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

	return []*cobra.Command{cmdIssue}
}

func init() {
	cmdAssets := &cobra.Command{
		Use:   "assets",
		Short: "Assets Manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdAssets.AddCommand(assetsSub()...)
	RootCmd.AddCommand(cmdAssets)
}
