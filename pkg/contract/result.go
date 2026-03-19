package contract

// CallResult holds the decoded response from a constant (read-only) contract call.
type CallResult struct {
	// RawResults contains the raw byte slices returned by the contract call.
	// Each entry corresponds to one return value from GetConstantResult().
	RawResults [][]byte
	// EnergyUsed is the energy consumed by the constant call, as reported
	// in the transaction's energy usage field.
	EnergyUsed int64
}
