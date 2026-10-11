package cmd

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	pkgsigner "github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/spf13/cobra"
)

// resolveSigner returns the signer for addr: the Ledger device when --ledger is
// set, the decrypted TIP-899 post-quantum key when addr is a PQ account,
// otherwise the unlocked keystore account. A PQ key stays decrypted until
// destroyPQSigningKeys runs at the end of the command.
//
// A failed PQ lookup (pq-keys unreadable, or a stray file in its place) does
// not block ECDSA accounts: the keystore is tried next, and only if that
// fails too is the error returned, carrying both causes.
func resolveSigner(addr tronAddress) (pkgsigner.Signer, error) {
	if useLedgerWallet {
		return pkgsigner.NewLedgerSignerWithAddress(addr.GetAddress()), nil
	}
	pqAcct, isPQ, pqErr := lookupPQAccount(addr.String())
	if pqErr == nil && isPQ {
		return resolvePQSigner(pqAcct)
	}
	ks, acct, err := store.UnlockedKeystore(addr.String(), passphrase)
	if err != nil {
		if pqErr != nil {
			return nil, fmt.Errorf("%w; post-quantum account lookup also failed: %w", err, pqErr)
		}
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

// executeOrDryRun runs ctrlr.ExecuteTransaction. Without --dry-run it prints
// nothing and returns stop=false, so the caller prints the broadcast result.
//
// With --dry-run the Controller signs but neither broadcasts nor confirms, so
// Result and Receipt stay nil and the caller must not read them. Instead it
// prints the signed transaction as JSON (transaction.ToJSON, indented unless
// --no-pretty) to stdout, and the txID with a note that nothing was broadcast
// to stderr, keeping stdout machine-readable. It returns stop=true; the caller
// must then return err, which is non-nil only if printing failed.
//
// The dry-run decision reads ctrlr.Behavior.DryRun, which opts copies from
// --dry-run, because that is the field the Controller itself checks before
// broadcasting.
func executeOrDryRun(cmd *cobra.Command, ctrlr *transaction.Controller) (stop bool, err error) {
	if err := ctrlr.ExecuteTransaction(); err != nil {
		return false, err
	}
	if !ctrlr.Behavior.DryRun {
		return false, nil
	}
	asJSON, err := transaction.ToJSON(ctrlr.Transaction())
	if err != nil {
		return true, fmt.Errorf("encode signed transaction: %w", err)
	}
	txID, err := ctrlr.TransactionHash()
	if err != nil {
		return true, fmt.Errorf("compute transaction ID: %w", err)
	}
	out := string(asJSON)
	if !noPrettyOutput {
		out = common.JSONPrettyFormat(out)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), out); err != nil {
		return true, err
	}
	if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "dry run: transaction %s signed but NOT broadcast\n", txID); err != nil {
		return true, err
	}
	return true, nil
}
