//go:build windows

package cmd

import (
	"os"
	"syscall"
)

// upgradeSignals cancel a running upgrade (download or version check), so its
// temp file is still removed. Windows has no SIGHUP; closing the console
// arrives as SIGTERM.
func upgradeSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
