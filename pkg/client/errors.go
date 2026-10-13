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
	// ErrNoStoredPermissions is returned by PermissionUpdateMaps when the account
	// has no stored owner permission or no active permission, as getaccount
	// reports for accounts never updated since multi-sign was enabled. The node
	// applies implicit defaults to such an account, so there is nothing to read
	// back; build the update yourself instead: an owner with threshold 1 and the
	// account address at weight 1, and one active permission named "active"
	// (id 2, threshold 1, the same key) with the chain's default active
	// operations (chain parameter getActiveDefaultOperations). A
	// DefaultPermissionUpdateMaps helper is planned.
	ErrNoStoredPermissions = errors.New("account has no stored owner or active permission")
)
