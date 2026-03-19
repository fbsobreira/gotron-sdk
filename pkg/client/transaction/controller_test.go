package transaction

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func newTestTransaction() *core.Transaction {
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type:         core.Transaction_Contract_TransferContract,
					Parameter:    &anypb.Any{},
					PermissionId: 0,
				},
			},
		},
	}
}

func TestWithPermissionID(t *testing.T) {
	tx := newTestTransaction()

	ctrl := NewController(nil, nil, nil, tx, WithPermissionID(2))

	require.NotNil(t, ctrl.Behavior.PermissionID, "expected PermissionId to be set")
	assert.Equal(t, int32(2), *ctrl.Behavior.PermissionID)
}

func TestWithPermissionIDZero(t *testing.T) {
	tx := newTestTransaction()
	// Pre-set a non-zero permission on the contract
	tx.GetRawData().GetContract()[0].PermissionId = 2

	ctrl := NewController(nil, nil, nil, tx, WithPermissionID(0))

	// WithPermissionID(0) should be explicitly set (not nil)
	require.NotNil(t, ctrl.Behavior.PermissionID, "expected PermissionId to be set, got nil")
	assert.Equal(t, int32(0), *ctrl.Behavior.PermissionID)

	// Apply should overwrite the contract's PermissionId back to 0
	ctrl.applyPermissionID()

	assert.Equal(t, int32(0), tx.GetRawData().GetContract()[0].PermissionId,
		"expected contract PermissionId=0 after apply")
}

func TestWithPermissionIDDefault(t *testing.T) {
	tx := newTestTransaction()

	ctrl := NewController(nil, nil, nil, tx)

	assert.Nil(t, ctrl.Behavior.PermissionID, "expected default PermissionId=nil")
}

func TestSetPermissionID(t *testing.T) {
	tx := newTestTransaction()

	setPermissionID(tx, 2)

	contracts := tx.GetRawData().GetContract()
	require.Len(t, contracts, 1)
	assert.Equal(t, int32(2), contracts[0].PermissionId)
}

func TestSetPermissionIDMultipleContracts(t *testing.T) {
	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{Type: core.Transaction_Contract_TransferContract, Parameter: &anypb.Any{}},
				{Type: core.Transaction_Contract_TransferAssetContract, Parameter: &anypb.Any{}},
			},
		},
	}

	setPermissionID(tx, 3)

	for i, contract := range tx.GetRawData().GetContract() {
		assert.Equal(t, int32(3), contract.PermissionId, "contract[%d]: expected PermissionId=3", i)
	}
}

func TestApplyPermissionID(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx, WithPermissionID(2))

	ctrl.applyPermissionID()

	contracts := ctrl.tx.GetRawData().GetContract()
	assert.Equal(t, int32(2), contracts[0].PermissionId, "expected PermissionId=2 after apply")
}

func TestApplyPermissionIDSkipsWhenNotSet(t *testing.T) {
	tx := newTestTransaction()
	// Manually set a non-zero value on the contract
	tx.GetRawData().GetContract()[0].PermissionId = 5

	ctrl := NewController(nil, nil, nil, tx) // no WithPermissionID option

	ctrl.applyPermissionID()

	// Should NOT overwrite because Behavior.PermissionID is nil
	assert.Equal(t, int32(5), tx.GetRawData().GetContract()[0].PermissionId,
		"expected PermissionId=5 to be preserved")
}

func TestSetPermissionIDNilSafe(t *testing.T) {
	// Should not panic on nil transaction or nil raw data
	setPermissionID(&core.Transaction{}, 2)
	setPermissionID(&core.Transaction{RawData: &core.TransactionRaw{}}, 2)
}

func TestSignTransactionWithPermissionID(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	require.NoError(t, err, "failed to generate key")

	makeTx := func() *core.Transaction {
		return &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{
					{
						Type:      core.Transaction_Contract_TransferContract,
						Parameter: &anypb.Any{},
					},
				},
			},
		}
	}

	// Sign without PermissionId
	tx1 := makeTx()
	signed1, err := SignTransaction(tx1, privKey)
	require.NoError(t, err, "sign tx1")

	// Sign with PermissionId = 2
	tx2 := makeTx()
	setPermissionID(tx2, 2)
	signed2, err := SignTransaction(tx2, privKey)
	require.NoError(t, err, "sign tx2")

	// Signatures must differ because the hash includes PermissionId
	assert.False(t, bytes.Equal(signed1.Signature[0], signed2.Signature[0]),
		"expected different signatures for different PermissionId values")

	// Verify PermissionId is preserved in the signed transaction
	assert.Equal(t, int32(2), signed2.GetRawData().GetContract()[0].PermissionId,
		"expected PermissionId=2 after signing")
}

func TestSignTransactionMultiSig(t *testing.T) {
	key1, err := btcec.NewPrivateKey()
	require.NoError(t, err, "failed to generate key1")
	key2, err := btcec.NewPrivateKey()
	require.NoError(t, err, "failed to generate key2")

	tx := newTestTransaction()
	setPermissionID(tx, 2)

	// Capture raw data before signing
	rawBefore, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err, "marshal before signing")

	// First signature
	tx, err = SignTransaction(tx, key1)
	require.NoError(t, err, "first sign")
	require.Len(t, tx.Signature, 1, "expected 1 signature")

	// Raw data should be unchanged after first signature
	rawAfterFirst, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err, "marshal after first sign")
	assert.True(t, bytes.Equal(rawBefore, rawAfterFirst), "raw data changed after first signature")

	// Second signature
	tx, err = SignTransaction(tx, key2)
	require.NoError(t, err, "second sign")
	require.Len(t, tx.Signature, 2, "expected 2 signatures")

	// Raw data should be unchanged after second signature
	rawAfterSecond, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err, "marshal after second sign")
	assert.True(t, bytes.Equal(rawBefore, rawAfterSecond), "raw data changed after second signature")

	// Both signatures should be different (different keys)
	assert.False(t, bytes.Equal(tx.Signature[0], tx.Signature[1]),
		"expected different signatures from different keys")

	// PermissionId should still be set
	assert.Equal(t, int32(2), tx.GetRawData().GetContract()[0].PermissionId,
		"PermissionId should be preserved after multi-sig signing")
}

// ---------------------------------------------------------------------------
// Mock gRPC server for controller tests
// ---------------------------------------------------------------------------

type mockWalletServer struct {
	api.UnimplementedWalletServer
	BroadcastTransactionFunc   func(context.Context, *core.Transaction) (*api.Return, error)
	GetTransactionInfoByIdFunc func(context.Context, *api.BytesMessage) (*core.TransactionInfo, error) //nolint:staticcheck,revive // matches proto-generated method name
}

func (m *mockWalletServer) BroadcastTransaction(ctx context.Context, in *core.Transaction) (*api.Return, error) {
	if m.BroadcastTransactionFunc != nil {
		return m.BroadcastTransactionFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.BroadcastTransaction(ctx, in)
}

func (m *mockWalletServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) { //nolint:staticcheck,revive // matches proto-generated method name
	if m.GetTransactionInfoByIdFunc != nil {
		return m.GetTransactionInfoByIdFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetTransactionInfoById(ctx, in)
}

const bufSize = 1024 * 1024

func newMockClient(t *testing.T, mock *mockWalletServer) *client.GrpcClient {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, mock)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() {
		srv.GracefulStop()
		_ = lis.Close()
	})

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "grpc.NewClient")
	t.Cleanup(func() { _ = conn.Close() })

	c := client.NewGrpcClient("bufconn")
	c.Conn = conn
	c.Client = api.NewWalletClient(conn)
	return c
}

// newTestKeystore creates a real keystore in a temp dir, creates an account,
// and returns the keystore and account. The account is left locked.
func newTestKeystore(t *testing.T) (*keystore.KeyStore, keystore.Account) {
	t.Helper()
	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount("test-pass")
	require.NoError(t, err, "NewAccount")
	// Note: we intentionally do not call ks.Close() in a t.Cleanup here.
	// The keystore's internal watcher goroutine has a known race between
	// accountCache.scanAccounts and accountCache.close, which surfaces
	// under -race when Close() runs while the watcher is still starting.
	// The GC finalizer on KeyStore handles resource cleanup, and the temp
	// dir is removed by t.TempDir().
	return ks, acct
}

// ---------------------------------------------------------------------------
// 1. TransactionHash
// ---------------------------------------------------------------------------

func TestTransactionHash(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx)

	hash, err := ctrl.TransactionHash()
	require.NoError(t, err, "TransactionHash")

	// Manually compute expected hash.
	rawData, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err, "proto.Marshal")
	h := sha256.Sum256(rawData)
	want := common.BytesToHexString(h[:])

	assert.Equal(t, want, hash)
}

func TestTransactionHashDeterministic(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx)

	h1, err := ctrl.TransactionHash()
	require.NoError(t, err, "first hash")
	h2, err := ctrl.TransactionHash()
	require.NoError(t, err, "second hash")
	assert.Equal(t, h1, h2, "non-deterministic hash")
}

func TestTransactionHashNilRawData(t *testing.T) {
	tx := &core.Transaction{}
	ctrl := NewController(nil, nil, nil, tx)

	hash, err := ctrl.TransactionHash()
	require.NoError(t, err, "unexpected error for nil raw data")
	// SHA256 of empty bytes (proto.Marshal of nil RawData returns empty slice)
	emptyHash := sha256.Sum256(nil)
	want := common.BytesToHexString(emptyHash[:])
	assert.Equal(t, want, hash, "nil raw data hash mismatch")
}

func TestTransactionHashHexFormat(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx)

	hash, err := ctrl.TransactionHash()
	require.NoError(t, err, "TransactionHash")

	// Must start with 0x prefix (BytesToHexString adds it).
	assert.True(t, strings.HasPrefix(hash, "0x"), "expected 0x prefix, got %s", hash)
	// Must be valid hex after prefix (64 hex chars for SHA256).
	hexPart := hash[2:]
	assert.Len(t, hexPart, 64, "expected 64 hex chars")
	_, err = hex.DecodeString(hexPart)
	assert.NoError(t, err, "invalid hex")
}

// ---------------------------------------------------------------------------
// 2. signTxForSending (uses real keystore)
// ---------------------------------------------------------------------------

func TestSignTxForSending_SkipsOnExecutionError(t *testing.T) {
	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")

	tx := newTestTransaction()
	ctrl := NewController(nil, ks, &acct, tx)
	ctrl.executionError = errors.New("prior error")

	ctrl.signTxForSending()

	// Should not have added a signature.
	assert.Empty(t, ctrl.tx.GetSignature(), "expected no signature when executionError is set")
	// executionError should still be the original error.
	assert.Equal(t, "prior error", ctrl.executionError.Error(), "executionError changed")
}

func TestSignTxForSending_LockedAccount(t *testing.T) {
	ks, acct := newTestKeystore(t)
	// Account is locked by default after creation.

	tx := newTestTransaction()
	ctrl := NewController(nil, ks, &acct, tx)

	ctrl.signTxForSending()

	require.Error(t, ctrl.executionError, "expected executionError for locked account")
	var authErr *keystore.AuthNeededError
	assert.ErrorAs(t, ctrl.executionError, &authErr, "expected AuthNeededError")
}

func TestSignTxForSending_UnlockedSuccess(t *testing.T) {
	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")

	tx := newTestTransaction()
	ctrl := NewController(nil, ks, &acct, tx)

	ctrl.signTxForSending()

	require.NoError(t, ctrl.executionError, "unexpected executionError")
	assert.Len(t, ctrl.tx.GetSignature(), 1, "expected 1 signature")
}

// ---------------------------------------------------------------------------
// 3. sendSignedTx
// ---------------------------------------------------------------------------

func TestSendSignedTx_SkipsOnExecutionError(t *testing.T) {
	mock := &mockWalletServer{}
	c := newMockClient(t, mock)

	tx := newTestTransaction()
	ctrl := NewController(c, nil, nil, tx)
	ctrl.executionError = errors.New("prior error")

	ctrl.sendSignedTx()

	assert.Nil(t, ctrl.Result, "expected nil Result when executionError is set")
}

func TestSendSignedTx_SkipsOnDryRun(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return nil, errors.New("should not be called")
		},
	}
	c := newMockClient(t, mock)

	tx := newTestTransaction()
	ctrl := NewController(c, nil, nil, tx)
	ctrl.Behavior.DryRun = true

	ctrl.sendSignedTx()

	assert.NoError(t, ctrl.executionError, "unexpected executionError")
	assert.Nil(t, ctrl.Result, "expected nil Result in DryRun mode")
}

func TestSendSignedTx_BroadcastError(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return nil, errors.New("network error")
		},
	}
	c := newMockClient(t, mock)

	tx := newTestTransaction()
	ctrl := NewController(c, nil, nil, tx)

	ctrl.sendSignedTx()

	require.Error(t, ctrl.executionError, "expected executionError on broadcast failure")
	assert.Contains(t, ctrl.executionError.Error(), "network error")
}

func TestSendSignedTx_NonSuccessCode(t *testing.T) {
	// BroadcastCtx checks result.GetCode() != SUCCESS and returns an error
	// before the controller's own Code check, so this verifies the client
	// layer's error propagation through sendSignedTx.
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{
				Result:  true,
				Code:    api.Return_SIGERROR,
				Message: []byte("signature mismatch"),
			}, nil
		},
	}
	c := newMockClient(t, mock)

	tx := newTestTransaction()
	ctrl := NewController(c, nil, nil, tx)

	ctrl.sendSignedTx()

	require.Error(t, ctrl.executionError, "expected executionError for non-success result code")
	assert.Contains(t, ctrl.executionError.Error(), "signature mismatch",
		"expected error to contain 'signature mismatch'")
}

func TestSendSignedTx_Success(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}
	c := newMockClient(t, mock)

	tx := newTestTransaction()
	ctrl := NewController(c, nil, nil, tx)

	ctrl.sendSignedTx()

	require.NoError(t, ctrl.executionError, "unexpected executionError")
	require.NotNil(t, ctrl.Result, "expected non-nil Result")
	assert.Equal(t, api.Return_SUCCESS, ctrl.Result.Code)
}

// ---------------------------------------------------------------------------
// 4. ExecuteTransaction (full flow)
// ---------------------------------------------------------------------------

func TestExecuteTransaction_DryRunSignsButDoesNotBroadcast(t *testing.T) {
	broadcastCalled := false
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			broadcastCalled = true
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}
	c := newMockClient(t, mock)

	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")

	tx := newTestTransaction()
	ctrl := NewController(c, ks, &acct, tx)
	ctrl.Behavior.DryRun = true

	err := ctrl.ExecuteTransaction()
	require.NoError(t, err, "ExecuteTransaction")

	// Should have signed.
	assert.Len(t, ctrl.tx.GetSignature(), 1, "expected 1 signature in DryRun")

	// Should NOT have called broadcast.
	assert.False(t, broadcastCalled, "broadcast should not be called in DryRun mode")
}

func TestExecuteTransaction_SoftwareSigningSuccess(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}
	c := newMockClient(t, mock)

	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")

	tx := newTestTransaction()
	ctrl := NewController(c, ks, &acct, tx)

	err := ctrl.ExecuteTransaction()
	require.NoError(t, err, "ExecuteTransaction")
	require.NotNil(t, ctrl.Result, "expected non-nil Result")
	assert.Len(t, ctrl.tx.GetSignature(), 1, "expected 1 signature")
}

func TestExecuteTransaction_SigningErrorStopsBroadcast(t *testing.T) {
	broadcastCalled := false
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			broadcastCalled = true
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}
	c := newMockClient(t, mock)

	ks, acct := newTestKeystore(t)
	// Account is locked, so signing will fail.

	tx := newTestTransaction()
	ctrl := NewController(c, ks, &acct, tx)

	err := ctrl.ExecuteTransaction()
	require.Error(t, err, "expected error from locked account")
	assert.False(t, broadcastCalled, "broadcast should not be called when signing fails")
}

func TestExecuteTransaction_BroadcastErrorPropagated(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return nil, errors.New("broadcast failure")
		},
	}
	c := newMockClient(t, mock)

	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")

	tx := newTestTransaction()
	ctrl := NewController(c, ks, &acct, tx)

	err := ctrl.ExecuteTransaction()
	require.Error(t, err, "expected broadcast error")
	assert.Contains(t, err.Error(), "broadcast failure")
}

func TestExecuteTransaction_WithPermissionIDApplied(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, in *core.Transaction) (*api.Return, error) {
			// Verify permission ID was applied before signing.
			pid := in.GetRawData().GetContract()[0].GetPermissionId()
			if pid != 2 {
				return nil, fmt.Errorf("expected PermissionId=2, got %d", pid)
			}

			// Verify the signature covers the final raw data (with PermissionId set).
			// Re-hash the raw data and recover the public key from the signature
			// to prove the tx was signed AFTER PermissionId was applied.
			rawData, err := proto.Marshal(in.GetRawData())
			if err != nil {
				return nil, fmt.Errorf("marshal raw data: %w", err)
			}
			h := sha256.Sum256(rawData)
			if len(in.GetSignature()) == 0 {
				return nil, fmt.Errorf("expected at least 1 signature")
			}
			_, err = crypto.Ecrecover(h[:], in.GetSignature()[0])
			if err != nil {
				return nil, fmt.Errorf("signature does not match raw data (PermissionId may have been set after signing): %w", err)
			}

			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}
	c := newMockClient(t, mock)

	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")

	tx := newTestTransaction()
	ctrl := NewController(c, ks, &acct, tx, WithPermissionID(2))

	err := ctrl.ExecuteTransaction()
	require.NoError(t, err, "ExecuteTransaction")

	assert.Equal(t, int32(2), ctrl.tx.GetRawData().GetContract()[0].GetPermissionId(),
		"PermissionId not applied to transaction")
}

// ---------------------------------------------------------------------------
// 5. txConfirmation
// ---------------------------------------------------------------------------

func TestTxConfirmation_SkipsOnExecutionError(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx)
	ctrl.executionError = errors.New("prior error")
	ctrl.Behavior.ConfirmationWaitTime = 5

	ctrl.txConfirmation()

	assert.Nil(t, ctrl.Receipt, "expected nil Receipt when executionError is set")
}

func TestTxConfirmation_SkipsOnDryRun(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx)
	ctrl.Behavior.DryRun = true
	ctrl.Behavior.ConfirmationWaitTime = 5

	ctrl.txConfirmation()

	assert.Nil(t, ctrl.Receipt, "expected nil Receipt in DryRun mode")
}

func TestTxConfirmation_ZeroWaitTimeSetsEmptyReceipt(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx)
	ctrl.Behavior.ConfirmationWaitTime = 0

	ctrl.txConfirmation()

	require.NotNil(t, ctrl.Receipt, "expected non-nil Receipt for zero wait time")
	assert.NotNil(t, ctrl.Receipt.Receipt, "expected non-nil Receipt.Receipt (ResourceReceipt)")
}

func TestTxConfirmation_SuccessfulConfirmation(t *testing.T) {
	tx := newTestTransaction()

	// Compute the tx hash so we can match it in the mock.
	rawData, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err, "proto.Marshal")
	h := sha256.Sum256(rawData)
	txIDBytes := h[:]

	mock := &mockWalletServer{
		GetTransactionInfoByIdFunc: func(_ context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			if !bytes.Equal(in.GetValue(), txIDBytes) {
				return nil, fmt.Errorf("expected tx id %x, got %x", txIDBytes, in.GetValue())
			}
			return &core.TransactionInfo{
				Id:      txIDBytes,
				Result:  0,
				Receipt: &core.ResourceReceipt{},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	ctrl := NewController(c, nil, nil, tx)
	ctrl.Behavior.ConfirmationWaitTime = 1

	ctrl.txConfirmation()

	require.NoError(t, ctrl.executionError, "unexpected executionError")
	require.NotNil(t, ctrl.Receipt, "expected non-nil Receipt")
}

func TestTxConfirmation_FailedResultSetsResultError(t *testing.T) {
	tx := newTestTransaction()

	rawData, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err, "proto.Marshal")
	h := sha256.Sum256(rawData)
	txIDBytes := h[:]

	mock := &mockWalletServer{
		GetTransactionInfoByIdFunc: func(_ context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
			if !bytes.Equal(in.GetValue(), txIDBytes) {
				return nil, fmt.Errorf("expected tx id %x, got %x", txIDBytes, in.GetValue())
			}
			return &core.TransactionInfo{
				Id:         txIDBytes,
				Result:     core.TransactionInfo_FAILED,
				ResMessage: []byte("out of energy"),
				Receipt:    &core.ResourceReceipt{},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	ctrl := NewController(c, nil, nil, tx)
	ctrl.Behavior.ConfirmationWaitTime = 1

	ctrl.txConfirmation()

	require.NoError(t, ctrl.executionError, "unexpected executionError")
	require.Error(t, ctrl.resultError, "expected resultError for failed result")
	assert.Contains(t, ctrl.resultError.Error(), "out of energy")
}

// ---------------------------------------------------------------------------
// 6. GetResultError
// ---------------------------------------------------------------------------

func TestGetResultError_NilWhenNoError(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx)

	assert.NoError(t, ctrl.GetResultError())
}

func TestGetResultError_ReturnsError(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx)
	ctrl.resultError = errors.New("some result error")

	err := ctrl.GetResultError()
	require.Error(t, err, "expected error, got nil")
	assert.Equal(t, "some result error", err.Error())
}
