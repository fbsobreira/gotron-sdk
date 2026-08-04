package client

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"google.golang.org/protobuf/proto"
)

// requireTxExtension validates a node-built TransactionExtention for write paths.
//
// A real full node typically returns either SUCCESS with a full Transaction body,
// or a non-zero result code with no Transaction. Builders also reject SUCCESS with
// a missing body so callers never receive a hollow extension to sign (custom
// WalletClient stubs and partial responses can produce that shape).
//
// op is a short name used in the missing-transaction error (e.g. "delegate resource").
func requireTxExtension(tx *api.TransactionExtention, op string) error {
	if tx == nil || proto.Size(tx) == 0 {
		return fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return fmt.Errorf("%s", string(tx.GetResult().GetMessage()))
	}
	if tx.GetTransaction().GetRawData() == nil {
		if op == "" {
			return fmt.Errorf("node returned no transaction")
		}
		return fmt.Errorf("%s: node returned no transaction", op)
	}
	return nil
}
