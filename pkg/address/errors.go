package address

import "errors"

// Sentinel errors for address validation. Callers can test with errors.Is.
var (
	ErrInvalidAddressLength   = errors.New("invalid address length")
	ErrInvalidAddressPrefix   = errors.New("invalid address prefix")
	ErrInvalidAddressChecksum = errors.New("invalid address checksum")
	ErrOversizeBigInt         = errors.New("big.Int too large for address")
	ErrInvalidHex             = errors.New("invalid hex string")
)
