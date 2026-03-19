package common

import (
	"errors"
	"os"
)

const (
	// DefaultConfigAccountAliasesDirName is the directory name for account key aliases.
	DefaultConfigAccountAliasesDirName = "account-keys"
	// DefaultPassphrase is the default passphrase used when none is provided.
	DefaultPassphrase = ""
	// Secp256k1PrivateKeyBytesLength is the required byte length of a secp256k1 private key.
	Secp256k1PrivateKeyBytesLength = 32
	// AmountDecimalPoint is the number of decimal places in TRX (1 TRX = 10^6 SUN).
	AmountDecimalPoint = 6
)

var (
	// DefaultConfigDirName is the default directory name for tronctl configuration.
	DefaultConfigDirName = ".tronctl"
	// DebugGRPC enables verbose gRPC request/response logging when true.
	DebugGRPC = false
	// DebugTransaction enables verbose transaction logging when true.
	DebugTransaction = false
	// ErrNotAbsPath is returned when a keypath is not an absolute path.
	ErrNotAbsPath = errors.New("keypath is not absolute path")
	// ErrBadKeyLength is returned when a private key has an invalid length.
	ErrBadKeyLength = errors.New("ivalid private key (wrong length)")
	// ErrFoundNoPass is returned when no passphrase file is found.
	ErrFoundNoPass = errors.New("found no passphrase file")
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

// EnableAllVerbose enables all debug logging flags (gRPC and transaction).
func EnableAllVerbose() {
	DebugGRPC = true
	DebugTransaction = true
}
