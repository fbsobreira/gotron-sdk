package client_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const (
	testAddrA    = "TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b"
	testAddrB    = "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"
	testContract = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
)

// emptyConstantResultMock reports success but returns no constant_result entries,
// which is what a hostile or buggy node can legitimately send back.
func emptyConstantResultMock() *mockWalletServer {
	return &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true, Code: api.Return_SUCCESS},
				ConstantResult: nil,
			}, nil
		},
	}
}

// A success code with an empty constant_result previously reached
// GetConstantResult()[0] and panicked with index out of range.
func TestTRC20Queries_EmptyConstantResult(t *testing.T) {
	c := newMockClient(t, emptyConstantResultMock())

	calls := map[string]func() error{
		"TRC20GetName":     func() error { _, err := c.TRC20GetName(testContract); return err },
		"TRC20GetSymbol":   func() error { _, err := c.TRC20GetSymbol(testContract); return err },
		"TRC20GetDecimals": func() error { _, err := c.TRC20GetDecimals(testContract); return err },
		"TRC20ContractBalance": func() error {
			_, err := c.TRC20ContractBalance(testAddrA, testContract)
			return err
		},
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				require.Error(t, call())
			})
		})
	}
}

// big.Int.Bytes returns the absolute value, so a negative amount was silently
// encoded as a real positive transfer. A nil amount panicked, and a value wider
// than 256 bits was appended whole because LeftPadBytes does not truncate.
func TestTRC20Writes_RejectInvalidAmount(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	amounts := map[string]*big.Int{
		"nil":                 nil,
		"negative":            big.NewInt(-1),
		"wider than 256 bits": new(big.Int).Lsh(big.NewInt(1), 256),
	}

	for name, amount := range amounts {
		t.Run(name, func(t *testing.T) {
			t.Run("TRC20Send", func(t *testing.T) {
				require.NotPanics(t, func() {
					_, err := c.TRC20Send(testAddrA, testAddrB, testContract, amount, 100)
					require.Error(t, err)
				})
			})
			t.Run("TRC20Approve", func(t *testing.T) {
				require.NotPanics(t, func() {
					_, err := c.TRC20Approve(testAddrA, testAddrB, testContract, amount, 100)
					require.Error(t, err)
				})
			})
			t.Run("TRC20TransferFrom", func(t *testing.T) {
				require.NotPanics(t, func() {
					_, err := c.TRC20TransferFrom(testAddrA, testAddrB, testAddrB, testContract, amount, 100)
					require.Error(t, err)
				})
			})
		})
	}
}

// The permission API takes map[string]interface{}, so a missing key or a plain
// int where int64 was meant previously panicked part-way through a multisig
// permission update rather than returning an error.
func TestUpdateAccountPermission_MalformedMaps(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	validOwner := map[string]interface{}{
		"threshold": int64(1),
		"keys":      map[string]int64{testAddrA: 1},
	}

	tests := map[string]struct {
		owner   map[string]interface{}
		witness map[string]interface{}
		actives []map[string]interface{}
	}{
		"owner missing threshold": {
			owner: map[string]interface{}{"keys": map[string]int64{testAddrA: 1}},
		},
		"owner missing keys": {
			owner: map[string]interface{}{"threshold": int64(1)},
		},
		"owner threshold is int not int64": {
			owner: map[string]interface{}{"threshold": 1, "keys": map[string]int64{testAddrA: 1}},
		},
		"active missing name": {
			owner:   validOwner,
			actives: []map[string]interface{}{{"threshold": int64(1)}},
		},
		"active missing operations": {
			owner: validOwner,
			actives: []map[string]interface{}{{
				"name": "active", "threshold": int64(1), "keys": map[string]int64{testAddrA: 1},
			}},
		},
		"witness missing keys": {
			owner:   validOwner,
			witness: map[string]interface{}{"threshold": int64(1)},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := c.UpdateAccountPermission(testAddrA, tt.owner, tt.witness, tt.actives)
				require.Error(t, err)
			})
		})
	}
}

// rejectedTx is what a node sends when it refuses a request: a nil gRPC error, a
// non-zero result code, a reason, and no Transaction.
func rejectedTx() *api.TransactionExtention {
	return &api.TransactionExtention{
		Result: &api.Return{
			Result:  false,
			Code:    api.Return_CONTRACT_VALIDATE_ERROR,
			Message: []byte("node refused the request"),
		},
	}
}

// successNoTx is a success-shaped TransactionExtention with no Transaction body.
// A node (or a custom WalletClient) can return this; callers that only check the
// result code would otherwise nil-dereference when signing or reading RawData.
func successNoTx() *api.TransactionExtention {
	return &api.TransactionExtention{
		Result: &api.Return{Result: true, Code: api.Return_SUCCESS},
	}
}

// These builders previously returned the rejection as an apparently valid
// TransactionExtention, so the failure surfaced only after signing and
// broadcast — or not at all. They must also reject success-with-no-tx.
func TestBuilders_SurfaceNodeRejection(t *testing.T) {
	t.Run("WithdrawExpireUnfreeze/non-zero code", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			WithdrawExpireUnfreezeFunc: func(_ context.Context, _ *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
				return rejectedTx(), nil
			},
		})
		_, err := c.WithdrawExpireUnfreeze(testAddrA, 1700000000000)
		require.ErrorContains(t, err, "node refused the request")
	})
	t.Run("WithdrawExpireUnfreeze/success no transaction", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			WithdrawExpireUnfreezeFunc: func(_ context.Context, _ *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.WithdrawExpireUnfreeze(testAddrA, 1700000000000)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("DelegateResource/non-zero code", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			DelegateResourceFunc: func(_ context.Context, _ *core.DelegateResourceContract) (*api.TransactionExtention, error) {
				return rejectedTx(), nil
			},
		})
		_, err := c.DelegateResource(testAddrA, testAddrB, core.ResourceCode_BANDWIDTH, 1_000_000, false, 0)
		require.ErrorContains(t, err, "node refused the request")
	})
	t.Run("DelegateResource/success no transaction", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			DelegateResourceFunc: func(_ context.Context, _ *core.DelegateResourceContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.DelegateResource(testAddrA, testAddrB, core.ResourceCode_BANDWIDTH, 1_000_000, false, 0)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("UnDelegateResource/non-zero code", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			UnDelegateResourceFunc: func(_ context.Context, _ *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
				return rejectedTx(), nil
			},
		})
		_, err := c.UnDelegateResource(testAddrA, testAddrB, core.ResourceCode_BANDWIDTH, 1_000_000)
		require.ErrorContains(t, err, "node refused the request")
	})
	t.Run("UnDelegateResource/success no transaction", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			UnDelegateResourceFunc: func(_ context.Context, _ *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.UnDelegateResource(testAddrA, testAddrB, core.ResourceCode_BANDWIDTH, 1_000_000)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})
}

// DeployContractCtx assigned to tx.Transaction.RawData without checking the
// result, so a rejection nil-dereferenced instead of reporting the reason.
func TestDeployContract_NodeRejection(t *testing.T) {
	t.Run("non-zero result code", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			DeployContractFunc: func(_ context.Context, _ *core.CreateSmartContract) (*api.TransactionExtention, error) {
				return rejectedTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.DeployContract(testAddrA, "Test", &core.SmartContract_ABI{}, "6080", 100, 50, 10000)
			require.ErrorContains(t, err, "node refused the request")
		})
	})

	t.Run("success code but no transaction", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			DeployContractFunc: func(_ context.Context, _ *core.CreateSmartContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.DeployContract(testAddrA, "Test", &core.SmartContract_ABI{}, "6080", 100, 50, 10000)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})
}

// triggerContract (and TRC20 writes that go through it) must surface node
// rejection and success-with-no-RawData as errors, never as a signable tx.
func TestTriggerContract_NodeRejection(t *testing.T) {
	t.Run("TRC20Send/non-zero result code", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			TriggerContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
				return rejectedTx(), nil
			},
		})
		require.NotPanics(t, func() {
			tx, err := c.TRC20Send(testAddrA, testAddrB, testContract, big.NewInt(1), 100)
			require.ErrorContains(t, err, "node refused the request")
			require.Nil(t, tx)
		})
	})

	t.Run("TRC20Send/success no transaction", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			TriggerContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			tx, err := c.TRC20Send(testAddrA, testAddrB, testContract, big.NewInt(1), 100)
			require.ErrorContains(t, err, "node returned no transaction")
			require.Nil(t, tx)
		})
	})

	t.Run("TriggerContract/non-zero result code", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			TriggerContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
				return rejectedTx(), nil
			},
		})
		require.NotPanics(t, func() {
			tx, err := c.TriggerContract(testAddrA, testContract, "transfer(address,uint256)",
				`[{"address": "`+testAddrB+`"},{"uint256": "1"}]`, 100, 0, "", 0)
			require.ErrorContains(t, err, "node refused the request")
			require.Nil(t, tx)
		})
	})

	t.Run("TriggerContract/success no transaction", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			TriggerContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			tx, err := c.TriggerContract(testAddrA, testContract, "transfer(address,uint256)",
				`[{"address": "`+testAddrB+`"},{"uint256": "1"}]`, 100, 0, "", 0)
			require.ErrorContains(t, err, "node returned no transaction")
			require.Nil(t, tx)
		})
	})
}

// nilInfoWalletClient returns a literal nil TransactionInfo with a nil error.
//
// This cannot be produced through the mock gRPC server: grpc-go materialises a
// (nil, nil) server return into an empty message on the wire, so the client
// always receives a non-nil value. Stubbing the exported Client field is the only
// way to reach the guard — and it is the case that matters, because an SDK
// consumer may substitute its own api.WalletClient. The embedded interface is nil,
// which is fine: only the overridden method is called.
type nilInfoWalletClient struct{ api.WalletClient }

// The Id spelling is fixed by the generated api.WalletClient interface.
//
//nolint:staticcheck // ST1003: must match the generated method name to satisfy the interface
func (nilInfoWalletClient) GetTransactionInfoById(
	context.Context, *api.BytesMessage, ...grpc.CallOption,
) (*core.TransactionInfo, error) {
	return nil, nil
}

// A nil response alongside a nil error previously reached txi.Id and panicked.
func TestGetTransactionInfoByID_NilResponse(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	c.Client = nilInfoWalletClient{}

	require.NotPanics(t, func() {
		_, err := c.GetTransactionInfoByID("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
		require.ErrorContains(t, err, "empty response")
	})
}

// An empty (non-nil) response is the shape a real gRPC round trip produces.
func TestGetTransactionInfoByID_EmptyResponse(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{
		GetTransactionInfoByIdFunc: func(_ context.Context, _ *api.BytesMessage) (*core.TransactionInfo, error) {
			return nil, nil
		},
	})
	require.NotPanics(t, func() {
		_, err := c.GetTransactionInfoByID("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
		require.Error(t, err)
	})
}

// nilResultWalletClient returns a TransactionExtention whose Result is nil,
// which the raw tx.Result.Code dereferences panicked on. As with the nil
// TransactionInfo above, a real gRPC round trip always materialises Result, so
// stubbing the exported Client field is the only way to reach these branches —
// and an SDK consumer substituting its own api.WalletClient can produce exactly
// this shape.
type nilResultWalletClient struct{ api.WalletClient }

func (nilResultWalletClient) TriggerConstantContract(
	context.Context, *core.TriggerSmartContract, ...grpc.CallOption,
) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{ConstantResult: [][]byte{make([]byte, 32)}}, nil
}

func (nilResultWalletClient) TriggerContract(
	context.Context, *core.TriggerSmartContract, ...grpc.CallOption,
) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, nil
}

func (nilResultWalletClient) DeployContract(
	context.Context, *core.CreateSmartContract, ...grpc.CallOption,
) (*api.TransactionExtention, error) {
	return &api.TransactionExtention{}, nil
}

func TestNilResultIsNotDereferenced(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	c.Client = nilResultWalletClient{}

	t.Run("TRC20GetDecimals via TRC20CallCtx", func(t *testing.T) {
		require.NotPanics(t, func() {
			// Result is nil but a well-formed constant result is present, so this
			// must survive the nil Result and decode normally.
			_, err := c.TRC20GetDecimals(testContract)
			require.NoError(t, err)
		})
	})

	t.Run("triggerContract", func(t *testing.T) {
		require.NotPanics(t, func() {
			_, err := c.TRC20Send(testAddrA, testAddrB, testContract, big.NewInt(1), 100)
			require.Error(t, err)
		})
	})

	t.Run("DeployContract", func(t *testing.T) {
		require.NotPanics(t, func() {
			_, err := c.DeployContract(testAddrA, "Test", &core.SmartContract_ABI{}, "6080", 100, 50, 10000)
			require.Error(t, err)
		})
	})
}

// Constant results are 32-byte aligned, so a shorter entry is malformed.
func TestTRC20Queries_ShortConstantResult(t *testing.T) {
	short := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true, Code: api.Return_SUCCESS},
				ConstantResult: [][]byte{{0x01, 0x02}},
			}, nil
		},
	}
	c := newMockClient(t, short)
	require.NotPanics(t, func() {
		_, err := c.TRC20GetDecimals(testContract)
		require.ErrorContains(t, err, "expected at least 32")
	})
}
