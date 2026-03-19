// Package txresult defines shared result types for TRON transaction
// operations, used by both the txbuilder and contract packages.
package txresult

// Receipt holds the result of a broadcast (and optional confirmation) of a
// TRON transaction.
type Receipt struct {
	TxID          string
	BlockNumber   int64
	Confirmed     bool
	EnergyUsed    int64
	BandwidthUsed int64
	Fee           int64  // in SUN
	Result        []byte // contract return data
	Error         string // TRON error message if failed
}
