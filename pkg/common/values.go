package common

import (
	"errors"
	"os"
)

const (
	// DefaultConfigAccountAliasesDirName fro accounts
	DefaultConfigAccountAliasesDirName = "account-keys"
	// DefaultPassphrase for accounts
	DefaultPassphrase = ""
	// Secp256k1PrivateKeyBytesLength privete key
	Secp256k1PrivateKeyBytesLength = 32
	// AmountDecimalPoint TRX decimal point
	AmountDecimalPoint = 6
)

var (
	// DefaultConfigDirName for wallets
	DefaultConfigDirName = ".tronctl"
	DebugGRPC            = false
	DebugTransaction     = false
	ErrNotAbsPath        = errors.New("keypath is not absolute path")
	ErrBadKeyLength      = errors.New("invalid private key (wrong length)")
	ErrFoundNoPass       = errors.New("found no passphrase file")
)

func init() {
	if _, enabled := os.LookupEnv("TRONCTL_GRPC_DEBUG"); enabled {
		DebugGRPC = true
	}
	if _, enabled := os.LookupEnv("TRONCTL_TX_DEBUG"); enabled {
		DebugTransaction = true
	}
	if _, enabled := os.LookupEnv("TRONCTL_ALL_DEBUG"); enabled {
		EnableAllVerbose()
	}
}

// EnableAllVerbose sets debug vars to true
func EnableAllVerbose() {
	DebugGRPC = true
	DebugTransaction = true
}
