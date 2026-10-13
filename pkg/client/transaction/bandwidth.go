package transaction

import (
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// maxResultSizeInTx is java-tron's MAX_RESULT_SIZE_IN_TX: the bytes it reserves
// per contract for the transaction result (ret) that the node adds itself.
const maxResultSizeInTx = 64

// EstimateBandwidth estimates the bandwidth, in bytes, a transaction will
// consume, using java-tron's rule: the serialized size of the transaction with
// its ret field cleared, plus 64 bytes (MAX_RESULT_SIZE_IN_TX) for every
// contract except a ShieldedTransferContract. The input is not modified.
//
// pending lists the post-quantum signatures that are not on tx yet. Each adds
// pqc.MaxAuthSigSize(scheme) bytes, the size of a pq_auth_sig entry with the
// scheme's public key and its largest signature. FN-DSA-512 signatures vary in
// length (617 to 667 bytes), so a pending FN-DSA-512 signature counts at its
// maximum and the estimate can exceed the real usage by up to 50 bytes per
// signature; ML-DSA-44 signatures have a fixed size. Pending schemes must be
// supported, otherwise the error wraps pqc.ErrUnsupportedScheme.
//
// This is an estimate, not a quote. Signatures already on tx (ECDSA in
// signature, post-quantum in pq_auth_sig) are counted because they are part of
// the serialized transaction. An ECDSA signature that is not on tx yet is NOT
// counted, since pending only covers post-quantum signatures: sign with the
// ECDSA keys first, or add 67 bytes (65 plus the field tag and length) for
// each one still to come. The account's free bandwidth, staked bandwidth, and
// the node's own accounting can still differ from this figure.
// A transaction that creates a new account is charged under the
// create-account rules, not by this byte count.
func EstimateBandwidth(tx *core.Transaction, pending ...core.PQScheme) (int64, error) {
	if tx == nil {
		return 0, errors.New("transaction: nil transaction")
	}
	// Clone so that clearing ret never touches the caller's transaction.
	c, ok := proto.Clone(tx).(*core.Transaction)
	if !ok {
		return 0, errors.New("transaction: cannot clone transaction")
	}
	c.Ret = nil
	size := int64(proto.Size(c))

	for _, contract := range c.GetRawData().GetContract() {
		if contract.GetType() == core.Transaction_Contract_ShieldedTransferContract {
			continue
		}
		size += maxResultSizeInTx
	}

	for i, scheme := range pending {
		n, err := pqc.MaxAuthSigSize(scheme)
		if err != nil {
			return 0, fmt.Errorf("transaction: pending[%d]: %w", i, err)
		}
		size += int64(n)
	}
	return size, nil
}
