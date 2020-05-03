package transaction

type SignerImpl int

const (
	Software SignerImpl = iota
	Ledger
)
