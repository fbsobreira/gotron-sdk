package client

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/require"
)

func TestRequireTxExtension(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		require.ErrorContains(t, requireTxExtension(nil, "op"), "bad transaction")
	})

	t.Run("empty", func(t *testing.T) {
		require.ErrorContains(t, requireTxExtension(&api.TransactionExtention{}, "op"), "bad transaction")
	})

	t.Run("non-zero code", func(t *testing.T) {
		tx := &api.TransactionExtention{
			Result: &api.Return{
				Result:  false,
				Code:    api.Return_CONTRACT_VALIDATE_ERROR,
				Message: []byte("node refused the request"),
			},
		}
		err := requireTxExtension(tx, "delegate resource")
		require.ErrorContains(t, err, "node refused the request")
	})

	t.Run("non-zero code empty message", func(t *testing.T) {
		tx := &api.TransactionExtention{
			Result: &api.Return{
				Result: false,
				Code:   api.Return_CONTRACT_VALIDATE_ERROR,
			},
		}
		err := requireTxExtension(tx, "transfer")
		require.ErrorContains(t, err, "node rejected request")
		require.ErrorContains(t, err, "CONTRACT_VALIDATE_ERROR")
	})

	t.Run("success without transaction", func(t *testing.T) {
		tx := &api.TransactionExtention{
			Result: &api.Return{Result: true, Code: api.Return_SUCCESS},
		}
		err := requireTxExtension(tx, "delegate resource")
		require.ErrorContains(t, err, "delegate resource: node returned no transaction")
	})

	t.Run("success without raw data", func(t *testing.T) {
		tx := &api.TransactionExtention{
			Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
			Transaction: &core.Transaction{},
		}
		err := requireTxExtension(tx, "transfer")
		require.ErrorContains(t, err, "transfer: node returned no transaction")
	})

	t.Run("success with raw data", func(t *testing.T) {
		tx := &api.TransactionExtention{
			Result: &api.Return{Result: true, Code: api.Return_SUCCESS},
			Transaction: &core.Transaction{
				RawData: &core.TransactionRaw{},
			},
		}
		require.NoError(t, requireTxExtension(tx, "transfer"))
	})
}
