package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/spf13/cobra"
)

// printJSON writes v to the command's output as indented JSON. It returns the
// encoding error instead of printing an empty result, so a command whose
// output cannot be encoded fails with a non-zero exit rather than looking
// successful.
func printJSON(cmd *cobra.Command, v interface{}) error {
	asJSON, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("encode result as JSON: %w", err)
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), common.JSONPrettyFormat(string(asJSON)))
	return err
}
