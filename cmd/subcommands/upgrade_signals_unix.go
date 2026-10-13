//go:build !windows

package cmd

import (
	"os"
	"syscall"
)

// upgradeSignals cancel a running upgrade (download or version check), so its
// temp file is still removed: Ctrl-C, a kill, or the terminal closing.
func upgradeSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}
