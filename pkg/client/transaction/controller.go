// Package transaction provides transaction signing and submission for TRON.
package transaction

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/txcore"
	proto "google.golang.org/protobuf/proto"
)

var (
	// ErrBadTransactionParam is returned when invalid params are given to the
	// controller upon execution of a transaction.
	ErrBadTransactionParam = errors.New("transaction has bad parameters")
)

type sender struct {
	ks      *keystore.KeyStore
	account *keystore.Account
}

// Controller drives the transaction signing process.
//
// The preferred way to choose how a transaction is signed is the WithSigner
// option, which accepts any signer.Signer (keystore, private key, Ledger, or
// a TIP-899 post-quantum signer). Without it the Controller falls back to
// Behavior.SigningImpl and the keystore and account given to NewController.
type Controller struct {
	executionError error
	resultError    error
	client         *client.GrpcClient
	tx             *core.Transaction
	sender         sender
	signer         signer.Signer
	Behavior       behavior
	Result         *api.Return
	Receipt        *core.TransactionInfo
}

type behavior struct {
	DryRun bool
	// SigningImpl selects the fallback signer when WithSigner is not used.
	// It is read when ExecuteTransaction runs, so it may be set after
	// NewController.
	SigningImpl          SignerImpl
	ConfirmationWaitTime uint32
	PermissionID         *int32
}

// NewController initializes a Controller, caller can control behavior via options
func NewController(
	client *client.GrpcClient,
	senderKs *keystore.KeyStore,
	senderAcct *keystore.Account,
	tx *core.Transaction,
	options ...func(*Controller),
) *Controller {

	ctrlr := &Controller{
		executionError: nil,
		resultError:    nil,
		client:         client,
		sender: sender{
			ks:      senderKs,
			account: senderAcct,
		},
		tx: tx,
		Behavior: behavior{
			SigningImpl: Software,
		},
	}
	for _, option := range options {
		option(ctrlr)
	}
	return ctrlr
}

// WithPermissionID sets the permission ID for multi-signature transactions.
// PermissionID = 0 is the owner permission (default), PermissionID = 2 is
// commonly used for active permissions in multi-sig setups.
func WithPermissionID(id int32) func(*Controller) {
	return func(c *Controller) {
		c.Behavior.PermissionID = &id
	}
}

// WithSigner sets the signer used by ExecuteTransaction. It is the preferred
// way to choose a signer; when set it takes precedence over
// Behavior.SigningImpl and the keystore and account passed to NewController,
// which are then ignored for signing. A nil signer is ignored.
//
// If s signs with a TIP-899 post-quantum key (it implements
// txcore.PQSchemeSigner, as signer.NewPQSigner does), ExecuteTransaction
// first asks the node whether the scheme is enabled and fails with an error
// wrapping txcore.ErrPQSchemeDisabled if it is not, without signing or
// broadcasting. This check also runs in DryRun mode.
//
// A typed-nil Signer (a nil pointer wrapped in a non-nil interface) is the
// caller's bug and is not filtered; only a nil interface is ignored.
func WithSigner(s signer.Signer) func(*Controller) {
	return func(c *Controller) {
		if s != nil {
			c.signer = s
		}
	}
}

func setPermissionID(tx *core.Transaction, id int32) {
	for _, contract := range tx.GetRawData().GetContract() {
		contract.PermissionId = id //nolint:staticcheck // ST1003: proto generated field name
	}
}

func (C *Controller) applyPermissionID() {
	if C.Behavior.PermissionID != nil {
		setPermissionID(C.tx, *C.Behavior.PermissionID)
	}
}

// softwareSigner returns the keystore signer used by the Software fallback.
func (C *Controller) softwareSigner() (signer.Signer, error) {
	if C.sender.ks == nil || C.sender.account == nil {
		return nil, fmt.Errorf("%w: software signing needs a keystore and an account", ErrBadTransactionParam)
	}
	return signer.NewKeystoreSigner(C.sender.ks, *C.sender.account), nil
}

// ledgerSigner returns the Ledger signer used by the Ledger fallback. It
// reuses the sender address and does not touch the device until Sign.
func (C *Controller) ledgerSigner() signer.Signer {
	var addr []byte
	if C.sender.account != nil {
		addr = C.sender.account.Address
	}
	return signer.NewLedgerSignerWithAddress(addr)
}

// resolveSigner picks the signer for ExecuteTransaction: the WithSigner
// signer if set, otherwise the one Behavior.SigningImpl selects. It returns
// a nil signer and no error for an unrecognised SigningImpl, which signs
// nothing, as before signers were introduced.
func (C *Controller) resolveSigner() (signer.Signer, error) {
	if C.signer != nil {
		return C.signer, nil
	}
	switch C.Behavior.SigningImpl {
	case Software:
		return C.softwareSigner()
	case Ledger:
		return C.ledgerSigner(), nil
	}
	return nil, nil
}

// signTx signs C.tx with the resolved signer.
func (C *Controller) signTx() {
	if C.executionError != nil {
		return
	}
	s, err := C.resolveSigner()
	if err != nil {
		C.executionError = err
		return
	}
	if s == nil {
		return
	}
	C.signWith(s)
}

// signWith signs C.tx with s. It refuses a nil transaction or nil or empty
// raw data before calling any signer, so no signer is asked to sign a payload
// other than the transaction. For a post-quantum signer it then checks that
// the node has the scheme enabled.
func (C *Controller) signWith(s signer.Signer) {
	if C.executionError != nil {
		return
	}
	if _, err := C.GetRawData(); err != nil {
		C.executionError = fmt.Errorf("get raw data for signing: %w", err)
		return
	}
	if err := C.checkPQScheme(s); err != nil {
		C.executionError = err
		return
	}
	signed, err := s.Sign(C.tx)
	if err != nil {
		C.executionError = err
		return
	}
	if signed == nil {
		C.executionError = errors.New("signer returned a nil transaction")
		return
	}
	C.tx = signed
}

// checkPQScheme asks the node whether a post-quantum signer's scheme is
// enabled. It returns nil without a network call for any other signer. The
// errors match txcore.Send.
func (C *Controller) checkPQScheme(s signer.Signer) error {
	pqSigner, ok := s.(txcore.PQSchemeSigner)
	if !ok {
		return nil
	}
	if C.client == nil {
		return fmt.Errorf("checking post-quantum capability: %w", errors.New("no client to query the node"))
	}
	scheme := pqSigner.PQScheme()
	allowed, err := C.client.PQSchemeAllowed(scheme)
	if err != nil {
		return fmt.Errorf("checking post-quantum capability: %w", err)
	}
	if !allowed {
		return fmt.Errorf("%w: %s", txcore.ErrPQSchemeDisabled, scheme)
	}
	return nil
}

// TransactionHash extract hash from TX
func (C *Controller) TransactionHash() (string, error) {
	rawData, err := C.GetRawData()
	if err != nil {
		return "", err
	}
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	return common.BytesToHexString(hash), nil
}

func (C *Controller) txConfirmation() {
	if C.executionError != nil || C.Behavior.DryRun {
		return
	}
	if C.Behavior.ConfirmationWaitTime > 0 {
		txHash, err := C.TransactionHash()
		if err != nil {
			C.executionError = fmt.Errorf("could not get tx hash: %w", err)
			return
		}
		//fmt.Printf("TX hash: %s\nWaiting for confirmation....", txHash)
		start := int(C.Behavior.ConfirmationWaitTime)
		for {
			// GETTX by ID
			if txi, err := C.client.GetTransactionInfoByID(txHash); err == nil {
				// check receipt
				if txi.Result != 0 {
					C.resultError = fmt.Errorf("%s", txi.ResMessage)
				}
				// Add receipt
				C.Receipt = txi
				return
			}
			if start < 0 {
				C.executionError = fmt.Errorf("could not confirm transaction after %d seconds", C.Behavior.ConfirmationWaitTime)
				return
			}
			time.Sleep(time.Second)
			start--
		}
	} else {
		C.Receipt = &core.TransactionInfo{}
		C.Receipt.Receipt = &core.ResourceReceipt{}
	}

}

// Transaction returns the Controller's transaction. After ExecuteTransaction
// it is the signed transaction, carrying its ECDSA signature or TIP-899
// pq_auth_sig, which in DryRun mode is the only output since nothing is
// broadcast and Result and Receipt stay nil. The returned value is the
// Controller's own instance, not a copy.
func (C *Controller) Transaction() *core.Transaction {
	return C.tx
}

// GetResultError return result error
func (C *Controller) GetResultError() error {
	return C.resultError
}

// ExecuteTransaction is the single entrypoint to execute a plain transaction.
// Each step in transaction creation, execution probably includes a mutation
// Each becomes a no-op if executionError occurred in any previous step
func (C *Controller) ExecuteTransaction() error {
	C.applyPermissionID()
	C.signTx()
	C.sendSignedTx()
	C.txConfirmation()
	return C.executionError
}

// GetRawData Byes from Transaction
func (C *Controller) GetRawData() ([]byte, error) {
	if C.tx == nil {
		return nil, errors.New("transaction is nil")
	}
	rawTransaction := C.tx.GetRawData()
	if rawTransaction == nil {
		return nil, errors.New("transaction raw data is nil")
	}
	rawData, err := proto.Marshal(rawTransaction)
	if err != nil {
		return nil, err
	}
	// Empty wire encoding is not a valid TRON payload to hash or sign.
	if len(rawData) == 0 {
		return nil, errors.New("transaction raw data is empty")
	}
	return rawData, nil
}

func (C *Controller) sendSignedTx() {
	if C.executionError != nil || C.Behavior.DryRun {
		return
	}
	result, err := C.client.Broadcast(C.tx)
	if err != nil {
		C.executionError = err
		return
	}
	if result.Code != 0 {
		C.executionError = fmt.Errorf("bad transaction: %v", string(result.GetMessage()))
	}
	C.Result = result
}
