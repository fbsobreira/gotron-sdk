package main

import (
	"fmt"
	"os"
	"path"

	cmd "github.com/fbsobreira/gotron-sdk/cmd/subcommands"
	// Need this side effect
	_ "github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
)

var (
	version string
	commit  string
	builtAt string
	builtBy string
)

func main() {
	// HACK Force usage of go implementation rather than the C based one. Do the right way, see the
	// notes one line 66,67 of https://golang.org/src/net/net.go that say can make the decision at
	// build time.
	os.Setenv("GODEBUG", "netdns=go")
	cmd.VersionWrapDump = version + "-" + commit
	cmd.RootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr,
				"TronCTL. %v version %v-%v (%v %v)\n",
				path.Base(os.Args[0]), version, commit, builtBy, builtAt)
			os.Exit(0)
			return nil
		},
	})
	cmd.Execute()
}
