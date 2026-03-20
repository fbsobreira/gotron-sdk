package txbuilder

import "errors"

// Sentinel errors for transaction builder validation. Callers can test with errors.Is.
var (
	ErrZeroAmount     = errors.New("amount must be greater than zero")
	ErrInvalidAddress = errors.New("invalid address")
	ErrMissingRawData = errors.New("invalid transaction: missing raw data")
	ErrAlreadyBuilt   = errors.New("txbuilder: Tx has already been built; create a new one")
)
