package client

import "errors"

// Sentinel errors for client operations. Callers can test with errors.Is.
var (
	// ErrEstimateEnergyNotSupported is returned when the connected TRON node
	// does not support the EstimateEnergy RPC.
	ErrEstimateEnergyNotSupported = errors.New("this node does not support estimate energy")
	// ErrNotConnected is returned when a client method is called without an active connection.
	ErrNotConnected = errors.New("client not connected")
	// ErrInvalidAddress is returned when an address parameter is malformed.
	ErrInvalidAddress = errors.New("invalid address")
	// ErrTimeout is returned when an operation exceeds its deadline.
	ErrTimeout = errors.New("operation timed out")
	// ErrZeroAmount is returned when an amount parameter is zero or negative.
	ErrZeroAmount = errors.New("amount must be greater than zero")
	// ErrInsufficientFee is returned when the fee limit is zero or negative.
	ErrInsufficientFee = errors.New("fee limit must be greater than zero")
	// ErrNoSigner is returned when a signing operation has no signer address.
	ErrNoSigner = errors.New("signer address required")
)
