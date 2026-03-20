package ledger

import "context"

// MockDevice is a test double that implements the Device interface.
// Callers can set the function fields to control return values;
// unset fields return zero values.
type MockDevice struct {
	// GetAddressFn is called by GetAddress. When nil, the stored Address
	// and Err fields are returned instead.
	GetAddressFn func() (string, error)
	// SignTransactionFn is called by SignTransaction. When nil, the stored
	// Signature and Err fields are returned instead.
	SignTransactionFn func(ctx context.Context, tx []byte) ([]byte, error)
	// CloseFn is called by Close. When nil, nil is returned.
	CloseFn func() error

	// Address is returned by GetAddress when GetAddressFn is nil.
	Address string
	// Signature is returned by SignTransaction when SignTransactionFn is nil.
	Signature []byte
	// Err is returned by both GetAddress and SignTransaction when their
	// respective function fields are nil.
	Err error
}

// GetAddress returns the configured address or calls GetAddressFn.
func (m *MockDevice) GetAddress() (string, error) {
	if m.GetAddressFn != nil {
		return m.GetAddressFn()
	}
	return m.Address, m.Err
}

// SignTransaction returns the configured signature or calls SignTransactionFn.
func (m *MockDevice) SignTransaction(ctx context.Context, tx []byte) ([]byte, error) {
	if m.SignTransactionFn != nil {
		return m.SignTransactionFn(ctx, tx)
	}
	return m.Signature, m.Err
}

// Close calls CloseFn if set, otherwise returns nil.
func (m *MockDevice) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}
