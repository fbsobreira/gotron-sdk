// Package signer defines a signing interface for TRON transactions and
// provides concrete implementations backed by private keys, keystores,
// and hardware wallets.
package signer

import (
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// Signer signs TRON transactions and exposes the associated address.
type Signer interface {
	// Sign appends a signature to the transaction and returns it.
	// Implementations must not modify the transaction's raw data.
	Sign(tx *core.Transaction) (*core.Transaction, error)

	// Address returns the TRON address of the signing key.
	Address() address.Address
}
