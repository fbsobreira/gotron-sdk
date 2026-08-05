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
		"active threshold wrong type": {
			owner: validOwner,
			actives: []map[string]interface{}{{
				"name": "active", "threshold": 1, "operations": map[string]bool{}, "keys": map[string]int64{testAddrA: 1},
			}},
		},
		"active keys wrong type": {
			owner: validOwner,
			actives: []map[string]interface{}{{
				"name": "active", "threshold": int64(1), "operations": map[string]bool{}, "keys": "nope",
			}},
		},
		"witness missing keys": {
			owner:   validOwner,
			witness: map[string]interface{}{"threshold": int64(1)},
		},
		"witness threshold wrong type": {
			owner:   validOwner,
			witness: map[string]interface{}{"threshold": 1, "keys": map[string]int64{testAddrA: 1}},
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

// nilAccountWalletClient returns a literal nil Account with a nil error.
type nilAccountWalletClient struct{ api.WalletClient }

func (nilAccountWalletClient) GetAccount(
	context.Context, *core.Account, ...grpc.CallOption,
) (*core.Account, error) {
	return nil, nil
}

// A nil Account previously reached acc.Address and panicked.
func TestGetAccount_NilResponse(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	c.Client = nilAccountWalletClient{}

	require.NotPanics(t, func() {
		_, err := c.GetAccount(testAddrA)
		require.ErrorContains(t, err, "account not found")
	})
}

// nilEstimateWalletClient returns a literal nil EstimateEnergyMessage.
type nilEstimateWalletClient struct{ api.WalletClient }

func (nilEstimateWalletClient) EstimateEnergy(
	context.Context, *core.TriggerSmartContract, ...grpc.CallOption,
) (*api.EstimateEnergyMessage, error) {
	return nil, nil
}

func TestEstimateEnergy_NilResponse(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	c.Client = nilEstimateWalletClient{}

	require.NotPanics(t, func() {
		_, err := c.EstimateEnergy(testAddrA, testContract, "transfer(address,uint256)",
			`[{"address": "`+testAddrB+`"},{"uint256": "1"}]`, 0, "", 0)
		require.ErrorContains(t, err, "empty response")
	})
}

func TestEstimateEnergy_NodeRejection(t *testing.T) {
	params := `[{"address": "` + testAddrB + `"},{"uint256": "1"}]`

	t.Run("non-zero code with message", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			EstimateEnergyFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
				return &api.EstimateEnergyMessage{
					Result: &api.Return{
						Result:  false,
						Code:    api.Return_CONTRACT_VALIDATE_ERROR,
						Message: []byte("out of energy estimate"),
					},
				}, nil
			},
		})
		_, err := c.EstimateEnergy(testAddrA, testContract, "transfer(address,uint256)", params, 0, "", 0)
		require.ErrorContains(t, err, "out of energy estimate")
	})

	t.Run("non-zero code empty message", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			EstimateEnergyFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
				return &api.EstimateEnergyMessage{
					Result: &api.Return{
						Result: false,
						Code:   api.Return_CONTRACT_VALIDATE_ERROR,
					},
				}, nil
			},
		})
		_, err := c.EstimateEnergy(testAddrA, testContract, "transfer(address,uint256)", params, 0, "", 0)
		require.ErrorContains(t, err, "node rejected request")
	})
}

// More write builders must surface SUCCESS-without-transaction through requireTxExtension.
func TestBuilders_SuccessNoTransaction(t *testing.T) {
	t.Run("Transfer", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			CreateTransaction2Func: func(_ context.Context, _ *core.TransferContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.Transfer(testAddrA, testAddrB, 1)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("UpdateAccount", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			UpdateAccount2Func: func(_ context.Context, _ *core.AccountUpdateContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.UpdateAccount(testAddrA, "name")
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("FreezeBalanceV2", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			FreezeBalanceV2Func: func(_ context.Context, _ *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.FreezeBalanceV2(testAddrA, core.ResourceCode_BANDWIDTH, 1_000_000)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("UpdateEnergyLimitContract", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			UpdateEnergyLimitFunc: func(_ context.Context, _ *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.UpdateEnergyLimitContract(testAddrA, testContract, 1_000_000)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("UpdateSettingContract", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			UpdateSettingFunc: func(_ context.Context, _ *core.UpdateSettingContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.UpdateSettingContract(testAddrA, testContract, 50)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("CreateWitness", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			CreateWitness2Func: func(_ context.Context, _ *core.WitnessCreateContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.CreateWitness(testAddrA, "https://example.com")
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("ProposalCreate", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			ProposalCreateFunc: func(_ context.Context, _ *core.ProposalCreateContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.ProposalCreate(testAddrA, map[int64]int64{1: 2})
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("ExchangeCreate", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			ExchangeCreateFunc: func(_ context.Context, _ *core.ExchangeCreateContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.ExchangeCreate(testAddrA, "_", 1_000_000, "_", 1_000_000)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("UnfreezeBalance", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			UnfreezeBalance2Func: func(_ context.Context, _ *core.UnfreezeBalanceContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.UnfreezeBalance(testAddrA, "", core.ResourceCode_BANDWIDTH)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("UnfreezeBalanceV2", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			UnfreezeBalanceV2Func: func(_ context.Context, _ *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.UnfreezeBalanceV2(testAddrA, core.ResourceCode_BANDWIDTH, 1_000_000)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("ProposalApprove", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			ProposalApproveFunc: func(_ context.Context, _ *core.ProposalApproveContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.ProposalApprove(testAddrA, 1, true)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("UpdateWitness", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			UpdateWitness2Func: func(_ context.Context, _ *core.WitnessUpdateContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.UpdateWitness(testAddrA, "https://example.com")
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("UpdateAccountPermission", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			AccountPermissionUpdateFunc: func(_ context.Context, _ *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		owner := map[string]interface{}{
			"threshold": int64(1),
			"keys":      map[string]int64{testAddrA: 1},
		}
		require.NotPanics(t, func() {
			_, err := c.UpdateAccountPermission(testAddrA, owner, nil, nil)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("ExchangeInject", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			ExchangeInjectFunc: func(_ context.Context, _ *core.ExchangeInjectContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.ExchangeInject(testAddrA, 1, "_", 1_000_000)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("ExchangeWithdraw", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			ExchangeWithdrawFunc: func(_ context.Context, _ *core.ExchangeWithdrawContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.ExchangeWithdraw(testAddrA, 1, "_", 1_000_000)
			require.ErrorContains(t, err, "node returned no transaction")
		})
	})

	t.Run("ExchangeTrade", func(t *testing.T) {
		c := newMockClient(t, &mockWalletServer{
			ExchangeTransactionFunc: func(_ context.Context, _ *core.ExchangeTransactionContract) (*api.TransactionExtention, error) {
				return successNoTx(), nil
			},
		})
		require.NotPanics(t, func() {
			_, err := c.ExchangeTrade(testAddrA, 1, "_", 100, 1)
			require.ErrorContains(t, err, "node returned no transaction")
		})
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
