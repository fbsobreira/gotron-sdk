package cmd

import (
	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	pkgsigner "github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
)

// resolveSigner returns the signer for addr: the Ledger device when --ledger is
// set, otherwise the unlocked keystore account.
func resolveSigner(addr tronAddress) (pkgsigner.Signer, error) {
	if useLedgerWallet {
		return pkgsigner.NewLedgerSignerWithAddress(addr.GetAddress()), nil
	}
	ks, acct, err := store.UnlockedKeystore(addr.String(), passphrase)
	if err != nil {
		return nil, err
	}
	return pkgsigner.NewKeystoreSigner(ks, *acct), nil
}

// newController builds a Controller that signs tx as addr with the configured
// signer and applies the global runtime options (dry run, confirmation wait).
func newController(addr tronAddress, tx *core.Transaction) (*transaction.Controller, error) {
	s, err := resolveSigner(addr)
	if err != nil {
		return nil, err
	}
	return transaction.NewController(conn, nil, nil, tx, transaction.WithSigner(s), opts), nil
}
