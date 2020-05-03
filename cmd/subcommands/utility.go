package cmd

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/spf13/cobra"
)

func init() {
	cmdUtilities := &cobra.Command{
		Use:   "utility",
		Short: "common tron utilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Help()
			return nil
		},
	}

	cmdUtilities.AddCommand([]*cobra.Command{{
		Use:   "metadata",
		Short: "data includes network specific values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}, {
		Use:   "metrics",
		Short: "mostly in-memory fluctuating values",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}, {
		Use:   "base58-to-addr",
		Args:  cobra.ExactArgs(1),
		Short: "0x Address of a base58 one-address",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := address.Base58ToAddress(args[0])
			if err != nil {
				return err
			}
			fmt.Println(addr.Hex())
			return nil
		},
	}, {
		Use:   "addr-to-base58",
		Args:  cobra.ExactArgs(1),
		Short: "base58 tron-address of an 0x address",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(address.HexToAddress(args[0]))
			return nil
		},
	}}...)

	RootCmd.AddCommand(cmdUtilities)
}
