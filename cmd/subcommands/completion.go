package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	cmdCompletion := &cobra.Command{
		Use:   "completion",
		Short: "Generates bash completion scripts",
		Long: `To load completion, run:

    . <(tronctl completion)

Add the line to your ~/.bashrc to enable completion for each bash session.
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			RootCmd.GenBashCompletion(os.Stdout)
			return nil
		},
	}
	RootCmd.AddCommand(cmdCompletion)
}
