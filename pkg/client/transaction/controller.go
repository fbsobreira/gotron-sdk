package transaction

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/ledger"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "github.com/golang/protobuf/proto"
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

// Controller drives the transaction signing process
type Controller struct {
	executionError error
	resultError    error
	client         *client.GrpcClient
	tx             *core.Transaction
	sender         sender
	Behavior       behavior
	Result         *api.Return
	Receipt        *core.TransactionInfo
}

type behavior struct {
	DryRun               bool
	SigningImpl          SignerImpl
	ConfirmationWaitTime uint32
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
		tx:       tx,
		Behavior: behavior{false, Software, 0},
	}
	for _, option := range options {
		option(ctrlr)
	}
	return ctrlr
}

func (C *Controller) signTxForSending() {
	if C.executionError != nil {
		return
	}
	signedTransaction, err :=
		C.sender.ks.SignTx(*C.sender.account, C.tx)
	if err != nil {
		C.executionError = err
		return
	}
	C.tx = signedTransaction
}

func (C *Controller) hardwareSignTxForSending() {
	if C.executionError != nil {
		return
	}
	data, _ := C.GetRawData()
	signature, err := ledger.SignTx(data)
	if err != nil {
		C.executionError = err
		return
	}

	/* TODO: validate signature
	if strings.Compare(signerAddr, address.ToBech32(C.sender.account.Address)) != 0 {
		C.executionError = ErrBadTransactionParam
		errorMsg := "signature verification failed : sender address doesn't match with ledger hardware address"
		C.transactionErrors = append(C.transactionErrors, &Error{
			ErrMessage:           &errorMsg,
			TimestampOfRejection: time.Now().Unix(),
		})
		return
	}
	*/
	// add signature
	C.tx.Signature = append(C.tx.Signature, signature)
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
	return common.ToHex(hash), nil
}

func (C *Controller) txConfirmation() {
	if C.executionError != nil || C.Behavior.DryRun {
		return
	}
	if C.Behavior.ConfirmationWaitTime > 0 {
		txHash, err := C.TransactionHash()
		if err != nil {
			C.executionError = fmt.Errorf("could not get tx hash")
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

// GetResultError return result error
func (C *Controller) GetResultError() error {
	return C.resultError
}

// ExecuteTransaction is the single entrypoint to execute a plain transaction.
// Each step in transaction creation, execution probably includes a mutation
// Each becomes a no-op if executionError occurred in any previous step
func (C *Controller) ExecuteTransaction() error {
	switch C.Behavior.SigningImpl {
	case Software:
		C.signTxForSending()
	case Ledger:
		C.hardwareSignTxForSending()
	}
	C.sendSignedTx()
	C.txConfirmation()
	return C.executionError
}

// GetRawData Byes from Transaction
func (C *Controller) GetRawData() ([]byte, error) {
	return proto.Marshal(C.tx.GetRawData())
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
