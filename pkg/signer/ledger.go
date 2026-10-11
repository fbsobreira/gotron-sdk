package signer

import (
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/ledger"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "google.golang.org/protobuf/proto"
)

// ledgerSigner implements Signer using a Ledger hardware wallet.
type ledgerSigner struct {
	addr address.Address
}

var _ Signer = (*ledgerSigner)(nil)

// NewLedgerSigner creates a Signer backed by a connected Ledger device.
// It immediately queries the device for its address.
func NewLedgerSigner() (Signer, error) {
	addrStr := ledger.GetAddress()
	addr, err := address.Base58ToAddress(addrStr)
	if err != nil {
		return nil, err
	}
	return &ledgerSigner{addr: addr}, nil
}

// NewLedgerSignerWithAddress creates a Signer backed by a Ledger device whose
// address the caller already knows. Unlike NewLedgerSigner it does no device
// I/O at construction; the device is opened only when Sign is called. The
// address is reported by Address and is not checked against the device.
func NewLedgerSignerWithAddress(addr address.Address) Signer {
	return &ledgerSigner{addr: append(address.Address(nil), addr...)}
}

// Sign signs the transaction using the connected Ledger device.
func (s *ledgerSigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return nil, err
	}

	sig, err := ledger.SignTx(rawData)
	if err != nil {
		return nil, err
	}

	tx.Signature = append(tx.Signature, sig)
	return tx, nil
}

// Address returns the TRON address of the Ledger device.
func (s *ledgerSigner) Address() address.Address {
	return s.addr
}
