package contract

import "errors"

// Sentinel errors for contract call validation. Callers can test with errors.Is.
var (
	ErrNoFromAddress = errors.New("from address required")
	ErrNoContract    = errors.New("contract address required")
	ErrNoMethod      = errors.New("method signature required")
)
