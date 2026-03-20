// Package ledger provides hardware wallet integration for TRON signing.
package ledger

import (
	"context"
	"fmt"

	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/crypto"
)

// Device abstracts a hardware wallet for testability and multi-device support.
type Device interface {
	// GetAddress returns the TRON address from the device.
	GetAddress() (string, error)
	// SignTransaction signs a raw transaction and returns the signature.
	SignTransaction(ctx context.Context, tx []byte) ([]byte, error)
	// Close releases the device connection.
	Close() error
}

// ensure NanoS implements Device at compile time.
var _ Device = (*NanoS)(nil)

// OpenDevice opens a Ledger Nano S and returns it as a Device.
// This is the recommended entry point for callers.
func OpenDevice() (Device, error) {
	return OpenNanoS()
}

// Close releases the underlying HID device connection.
func (n *NanoS) Close() error {
	// NanoS does not currently hold a closeable reference beyond the
	// apduFramer, which wraps a hidFramer wrapping an io.ReadWriter.
	// The hid.Device returned by Open() implements io.ReadWriteCloser.
	if closer, ok := n.device.hf.rw.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

// SignTransaction signs the given raw transaction bytes using the Ledger device.
// It returns the 65-byte signature after verifying via ecrecover.
func (n *NanoS) SignTransaction(_ context.Context, tx []byte) ([]byte, error) {
	sig, err := n.SignTxn(tx)
	if err != nil {
		return nil, fmt.Errorf("ledger sign: %w", err)
	}

	var hashBytes [32]byte
	hw := sha3.NewLegacyKeccak256()
	hw.Write(tx)
	hw.Sum(hashBytes[:0])

	pubkey, err := crypto.Ecrecover(hashBytes[:], sig[:])
	if err != nil {
		return nil, fmt.Errorf("ecrecover: %w", err)
	}

	if len(pubkey) == 0 || pubkey[0] != 4 {
		return nil, fmt.Errorf("invalid public key from ledger signature")
	}

	return sig[:], nil
}

// GetAddress returns the TRON address from the connected Ledger device.
// Deprecated: Use OpenDevice() and call GetAddress() on the Device instead.
func GetAddress() string {
	dev, err := OpenNanoS()
	if err != nil {
		return ""
	}
	defer func() { _ = dev.Close() }()
	addr, err := dev.GetAddress()
	if err != nil {
		return ""
	}
	return addr
}

// SignTx signs the given transaction bytes using a newly opened Ledger device.
// Deprecated: Use OpenDevice() and call SignTransaction() on the Device instead.
func SignTx(tx []byte) ([]byte, error) {
	dev, err := OpenNanoS()
	if err != nil {
		return nil, fmt.Errorf("open ledger: %w", err)
	}
	defer func() { _ = dev.Close() }()
	return dev.SignTransaction(context.Background(), tx)
}
