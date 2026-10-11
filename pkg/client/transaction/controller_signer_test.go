package transaction

import (
	"context"
	"crypto/sha256"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/txcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// recordingSigner is a Signer that counts Sign calls and appends a fixed
// dummy signature, or returns a nil transaction when returnNil is set.
type recordingSigner struct {
	calls     int
	returnNil bool
}

func (s *recordingSigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	s.calls++
	if s.returnNil {
		return nil, nil
	}
	tx.Signature = append(tx.Signature, []byte{0x01})
	return tx, nil
}

func (s *recordingSigner) Address() address.Address { return nil }

// recordingPQSigner is a recordingSigner that also reports a post-quantum
// scheme, so the Controller treats it as a PQ signer.
type recordingPQSigner struct {
	recordingSigner
}

func (s *recordingPQSigner) PQScheme() core.PQScheme { return core.PQScheme_ML_DSA_44 }

func chainParamsWith(key string, value int64) *core.ChainParameters {
	return &core.ChainParameters{
		ChainParameter: []*core.ChainParameters_ChainParameter{{Key: key, Value: value}},
	}
}

func broadcastOK(_ context.Context, _ *core.Transaction) (*api.Return, error) {
	return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
}

func chainParamsDown(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
	return nil, status.Error(codes.Unavailable, "node down")
}

func newMLDSASigner(t *testing.T) (signer.Signer, pqc.PrivateKey) {
	t.Helper()
	key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, nil)
	require.NoError(t, err)
	s, err := signer.NewPQSigner(key)
	require.NoError(t, err)
	return s, key
}

// TestExecuteTransaction_ECDSAGolden pins the keystore signature produced by
// the Controller to a direct ks.SignTx on an identical transaction. ECDSA is
// RFC 6979 deterministic, so the bytes must match exactly. The node's chain
// parameters endpoint fails, proving no post-quantum probe is made for ECDSA.
func TestExecuteTransaction_ECDSAGolden(t *testing.T) {
	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")

	pid := int32(2)
	tests := []struct {
		name       string
		withSigner bool
		pid        *int32
	}{
		{"fallback software", false, nil},
		{"fallback software with permission id", false, &pid},
		{"WithSigner keystore", true, nil},
		{"WithSigner keystore with permission id", true, &pid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWalletServer{GetChainParametersFunc: chainParamsDown}
			c := newMockClient(t, mock)

			tx := newTestTransaction()
			want := proto.Clone(tx).(*core.Transaction)
			if tt.pid != nil {
				setPermissionID(want, *tt.pid)
			}
			want, err := ks.SignTx(acct, want)
			require.NoError(t, err, "reference SignTx")
			require.Len(t, want.GetSignature(), 1)

			var opts []func(*Controller)
			if tt.withSigner {
				opts = append(opts, WithSigner(signer.NewKeystoreSigner(ks, acct)))
			}
			if tt.pid != nil {
				opts = append(opts, WithPermissionID(*tt.pid))
			}
			ctrl := NewController(c, ks, &acct, tx, opts...)
			ctrl.Behavior.DryRun = true

			require.NoError(t, ctrl.ExecuteTransaction())
			require.Len(t, ctrl.tx.GetSignature(), 1)
			assert.Equal(t, want.GetSignature()[0], ctrl.tx.GetSignature()[0], "signature bytes changed")
			assert.True(t, proto.Equal(want.GetRawData(), ctrl.tx.GetRawData()), "raw data changed")
			assert.Empty(t, ctrl.tx.GetPqAuthSig())
			assert.Zero(t, mock.chainParamsCalls.Load(), "ECDSA signing must not probe chain parameters")
		})
	}
}

// TestExecuteTransaction_ECDSANoProbeOnBroadcast covers the non-DryRun path:
// an ECDSA transaction must still broadcast when the chain parameters
// endpoint is down, because no probe is made.
func TestExecuteTransaction_ECDSANoProbeOnBroadcast(t *testing.T) {
	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")

	tests := []struct {
		name string
		opts []func(*Controller)
	}{
		{"fallback software", nil},
		{"WithSigner keystore", []func(*Controller){WithSigner(signer.NewKeystoreSigner(ks, acct))}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWalletServer{
				BroadcastTransactionFunc: broadcastOK,
				GetChainParametersFunc:   chainParamsDown,
			}
			c := newMockClient(t, mock)

			ctrl := NewController(c, ks, &acct, newTestTransaction(), tt.opts...)
			require.NoError(t, ctrl.ExecuteTransaction())
			assert.Len(t, ctrl.tx.GetSignature(), 1)
			assert.Equal(t, int32(1), mock.broadcastCalls.Load())
			assert.Zero(t, mock.chainParamsCalls.Load())
		})
	}
}

func TestExecuteTransaction_WithPrivateKeySigner(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	s, err := signer.NewPrivateKeySigner(key)
	require.NoError(t, err)

	mock := &mockWalletServer{BroadcastTransactionFunc: broadcastOK}
	c := newMockClient(t, mock)

	tx := newTestTransaction()
	want, err := SignTransactionECDSA(proto.Clone(tx).(*core.Transaction), key)
	require.NoError(t, err)

	// No keystore or account: WithSigner alone must be enough.
	ctrl := NewController(c, nil, nil, tx, WithSigner(s))
	require.NoError(t, ctrl.ExecuteTransaction())

	require.Len(t, ctrl.tx.GetSignature(), 1)
	assert.Equal(t, want.GetSignature()[0], ctrl.tx.GetSignature()[0])

	raw, err := proto.Marshal(ctrl.tx.GetRawData())
	require.NoError(t, err)
	h := sha256.Sum256(raw)
	pub, err := crypto.SigToPub(h[:], ctrl.tx.GetSignature()[0])
	require.NoError(t, err)
	assert.Equal(t, crypto.PubkeyToAddress(key.PublicKey), crypto.PubkeyToAddress(*pub))
	assert.Equal(t, int32(1), mock.broadcastCalls.Load())
	assert.Zero(t, mock.chainParamsCalls.Load())
}

// TestExecuteTransaction_WithSignerWinsOverSigningImpl sets SigningImpl to
// Ledger after construction, as the CLI does. WithSigner must still be used;
// falling back to the Ledger would fail because no device is attached.
func TestExecuteTransaction_WithSignerWinsOverSigningImpl(t *testing.T) {
	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")

	ctrl := NewController(nil, nil, nil, newTestTransaction(), WithSigner(signer.NewKeystoreSigner(ks, acct)))
	ctrl.Behavior.SigningImpl = Ledger
	ctrl.Behavior.DryRun = true

	require.NoError(t, ctrl.ExecuteTransaction())
	assert.Len(t, ctrl.tx.GetSignature(), 1)
}

func TestExecuteTransaction_PQSigner(t *testing.T) {
	pid2 := int32(2)
	tests := []struct {
		name          string
		chainParams   func(context.Context, *api.EmptyMessage) (*core.ChainParameters, error)
		dryRun        bool
		wantErrIs     error
		wantErrSubstr []string
		permissionID  *int32
		wantBroadcast int32
	}{
		{
			name: "enabled",
			chainParams: func(context.Context, *api.EmptyMessage) (*core.ChainParameters, error) {
				return chainParamsWith("getAllowMlDsa44", 1), nil
			},
			wantBroadcast: 1,
		},
		{
			name: "enabled dry run",
			chainParams: func(context.Context, *api.EmptyMessage) (*core.ChainParameters, error) {
				return chainParamsWith("getAllowMlDsa44", 1), nil
			},
			dryRun: true,
		},
		{
			name: "disabled",
			chainParams: func(context.Context, *api.EmptyMessage) (*core.ChainParameters, error) {
				return chainParamsWith("getAllowMlDsa44", 0), nil
			},
			wantErrIs:     txcore.ErrPQSchemeDisabled,
			wantErrSubstr: []string{"ML_DSA_44"},
		},
		{
			name: "disabled dry run",
			chainParams: func(context.Context, *api.EmptyMessage) (*core.ChainParameters, error) {
				return chainParamsWith("getAllowMlDsa44", 0), nil
			},
			dryRun:    true,
			wantErrIs: txcore.ErrPQSchemeDisabled,
		},
		{
			name: "disabled with permission id",
			chainParams: func(context.Context, *api.EmptyMessage) (*core.ChainParameters, error) {
				return chainParamsWith("getAllowMlDsa44", 0), nil
			},
			permissionID: &pid2,
			wantErrIs:    txcore.ErrPQSchemeDisabled,
		},
		{
			name:          "probe error",
			chainParams:   chainParamsDown,
			wantErrSubstr: []string{"checking post-quantum capability", "node down"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWalletServer{
				BroadcastTransactionFunc: broadcastOK,
				GetChainParametersFunc:   tt.chainParams,
			}
			c := newMockClient(t, mock)
			s, key := newMLDSASigner(t)

			tx := newTestTransaction()
			before := proto.Clone(tx).(*core.Transaction)
			var opts []func(*Controller)
			opts = append(opts, WithSigner(s))
			if tt.permissionID != nil {
				opts = append(opts, WithPermissionID(*tt.permissionID))
			}
			ctrl := NewController(c, nil, nil, tx, opts...)
			ctrl.Behavior.DryRun = tt.dryRun

			err := ctrl.ExecuteTransaction()
			assert.Equal(t, int32(1), mock.chainParamsCalls.Load(), "expected exactly one capability probe")
			assert.Equal(t, tt.wantBroadcast, mock.broadcastCalls.Load())

			if tt.wantErrIs != nil || len(tt.wantErrSubstr) > 0 {
				require.Error(t, err)
				if tt.wantErrIs != nil {
					assert.ErrorIs(t, err, tt.wantErrIs)
				} else {
					assert.NotErrorIs(t, err, txcore.ErrPQSchemeDisabled)
				}
				for _, sub := range tt.wantErrSubstr {
					assert.ErrorContains(t, err, sub)
				}
				assert.Empty(t, ctrl.tx.GetSignature())
				assert.Empty(t, ctrl.tx.GetPqAuthSig())
				if tt.permissionID == nil {
					// A permission id is applied before signing, so only compare without one.
					assert.True(t, proto.Equal(before, ctrl.tx), "transaction must be unchanged when signing is refused")
				}
				return
			}

			require.NoError(t, err)
			assert.Empty(t, ctrl.tx.GetSignature(), "PQ signer must not add an ECDSA signature")
			require.Len(t, ctrl.tx.GetPqAuthSig(), 1)
			entry := ctrl.tx.GetPqAuthSig()[0]
			assert.Equal(t, core.PQScheme_ML_DSA_44, entry.GetScheme())
			assert.Equal(t, key.Public().Bytes(), entry.GetPublicKey())

			raw, err := proto.Marshal(ctrl.tx.GetRawData())
			require.NoError(t, err)
			digest := sha256.Sum256(raw)
			ok, err := key.Public().Verify(digest[:], entry.GetSignature())
			require.NoError(t, err)
			assert.True(t, ok, "pq_auth_sig does not verify over SHA-256(raw_data)")
		})
	}
}

func TestExecuteTransaction_PQSignerNilClient(t *testing.T) {
	s, _ := newMLDSASigner(t)
	tx := newTestTransaction()
	before := proto.Clone(tx).(*core.Transaction)

	ctrl := NewController(nil, nil, nil, tx, WithSigner(s))
	ctrl.Behavior.DryRun = true

	err := ctrl.ExecuteTransaction()
	require.Error(t, err)
	assert.ErrorContains(t, err, "checking post-quantum capability")
	assert.True(t, proto.Equal(before, ctrl.tx))
}

// TestExecuteTransaction_RawDataGuardWithSigner checks that the nil/empty raw
// data guard runs before any signer, and before the PQ probe.
func TestExecuteTransaction_RawDataGuardWithSigner(t *testing.T) {
	tests := []struct {
		name    string
		tx      *core.Transaction
		wantSub string
	}{
		{"nil transaction", nil, "transaction is nil"},
		{"nil raw data", &core.Transaction{}, "transaction raw data is nil"},
		{"empty raw data", &core.Transaction{RawData: &core.TransactionRaw{}}, "transaction raw data is empty"},
	}
	for _, tt := range tests {
		for _, pq := range []bool{false, true} {
			name := tt.name
			if pq {
				name += " pq"
			}
			t.Run(name, func(t *testing.T) {
				mock := &mockWalletServer{BroadcastTransactionFunc: broadcastOK}
				c := newMockClient(t, mock)

				var s signer.Signer
				var calls func() int
				if pq {
					ps := &recordingPQSigner{}
					s, calls = ps, func() int { return ps.calls }
				} else {
					rs := &recordingSigner{}
					s, calls = rs, func() int { return rs.calls }
				}

				ctrl := NewController(c, nil, nil, tt.tx, WithSigner(s))
				err := ctrl.ExecuteTransaction()

				require.Error(t, err)
				assert.ErrorContains(t, err, "get raw data for signing")
				assert.ErrorContains(t, err, tt.wantSub)
				assert.Zero(t, calls(), "Sign must not be called")
				assert.Zero(t, mock.chainParamsCalls.Load(), "must not probe for an unsignable transaction")
				assert.Zero(t, mock.broadcastCalls.Load())
			})
		}
	}
}

func TestExecuteTransaction_SignerReturnsNilTransaction(t *testing.T) {
	mock := &mockWalletServer{BroadcastTransactionFunc: broadcastOK}
	c := newMockClient(t, mock)

	s := &recordingSigner{returnNil: true}
	ctrl := NewController(c, nil, nil, newTestTransaction(), WithSigner(s))

	err := ctrl.ExecuteTransaction()
	require.Error(t, err)
	assert.Equal(t, 1, s.calls)
	assert.NotNil(t, ctrl.tx, "must keep the original transaction")
	assert.Zero(t, mock.broadcastCalls.Load())
}

// TestExecuteTransaction_SoftwareFallbackMissingKeystore used to panic on a
// nil dereference; it must now report ErrBadTransactionParam.
func TestExecuteTransaction_SoftwareFallbackMissingKeystore(t *testing.T) {
	ks, acct := newTestKeystore(t)

	tests := []struct {
		name string
		ks   bool
		acct bool
	}{
		{"nil keystore", false, true},
		{"nil account", true, false},
		{"both nil", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockWalletServer{BroadcastTransactionFunc: broadcastOK}
			c := newMockClient(t, mock)

			ctrl := NewController(c, nil, nil, newTestTransaction())
			if tt.ks {
				ctrl.sender.ks = ks
			}
			if tt.acct {
				a := acct
				ctrl.sender.account = &a
			}

			err := ctrl.ExecuteTransaction()
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrBadTransactionParam)
			assert.Empty(t, ctrl.tx.GetSignature())
			assert.Zero(t, mock.broadcastCalls.Load())

			// signTx reports the same error.
			ctrl2 := NewController(nil, ctrl.sender.ks, ctrl.sender.account, newTestTransaction())
			ctrl2.Behavior.SigningImpl = Software
			ctrl2.signTx()
			assert.ErrorIs(t, ctrl2.executionError, ErrBadTransactionParam)
		})
	}
}

// TestResolveSigner_LedgerFallbackIsLazy checks that SigningImpl is read at
// execution time (the CLI sets it after NewController) and that the Ledger
// fallback uses the known sender address without touching a device.
func TestResolveSigner_LedgerFallbackIsLazy(t *testing.T) {
	addr, err := address.Base58ToAddress("TPL66VK2gCXNCD7EJg9pgJRfqcRazjhUZY")
	require.NoError(t, err)

	acct := keystore.Account{Address: addr}
	ctrl := NewController(nil, nil, &acct, newTestTransaction())
	ctrl.Behavior.SigningImpl = Ledger

	s, err := ctrl.resolveSigner()
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, addr, s.Address())

	// No sender account: still resolves, with no address, as the old Ledger
	// path never needed one.
	ctrl.sender.account = nil
	s, err = ctrl.resolveSigner()
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Empty(t, s.Address())
}

// TestExecuteTransaction_UnknownSigningImplSkipsSigning pins the pre-existing
// behaviour: an unrecognised SigningImpl without WithSigner signs nothing and
// the transaction is broadcast as given.
func TestExecuteTransaction_UnknownSigningImplSkipsSigning(t *testing.T) {
	mock := &mockWalletServer{BroadcastTransactionFunc: broadcastOK}
	c := newMockClient(t, mock)

	ctrl := NewController(c, nil, nil, newTestTransaction())
	ctrl.Behavior.SigningImpl = SignerImpl(99)

	require.NoError(t, ctrl.ExecuteTransaction())
	assert.Empty(t, ctrl.tx.GetSignature())
	assert.Equal(t, int32(1), mock.broadcastCalls.Load())
}

func TestWithSigner_NilIsIgnored(t *testing.T) {
	ctrl := NewController(nil, nil, nil, newTestTransaction(), WithSigner(nil))
	assert.Nil(t, ctrl.signer)
}

// TestExecuteTransaction_MultiSignAppends checks that signing a transaction
// that already carries a signature appends to it and leaves it untouched.
func TestExecuteTransaction_MultiSignAppends(t *testing.T) {
	ks, acct := newTestKeystore(t)
	require.NoError(t, ks.Unlock(acct, "test-pass"), "Unlock")
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	pkSigner, err := signer.NewPrivateKeySigner(key)
	require.NoError(t, err)

	prev := make([]byte, 65)
	for i := range prev {
		prev[i] = byte(i + 1)
	}

	refKeystore := func(t *testing.T, tx *core.Transaction) []byte {
		signed, err := ks.SignTx(acct, proto.Clone(tx).(*core.Transaction))
		require.NoError(t, err, "reference SignTx")
		require.Len(t, signed.GetSignature(), 1)
		return signed.GetSignature()[0]
	}
	refPrivateKey := func(t *testing.T, tx *core.Transaction) []byte {
		signed, err := SignTransactionECDSA(proto.Clone(tx).(*core.Transaction), key)
		require.NoError(t, err, "reference SignTransactionECDSA")
		require.Len(t, signed.GetSignature(), 1)
		return signed.GetSignature()[0]
	}

	tests := []struct {
		name string
		opts []func(*Controller)
		ref  func(*testing.T, *core.Transaction) []byte
	}{
		{"fallback software", nil, refKeystore},
		{"WithSigner keystore", []func(*Controller){WithSigner(signer.NewKeystoreSigner(ks, acct))}, refKeystore},
		{"WithSigner private key", []func(*Controller){WithSigner(pkSigner)}, refPrivateKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newTestTransaction()
			want := tt.ref(t, tx)
			tx.Signature = [][]byte{append([]byte(nil), prev...)}

			c := newMockClient(t, &mockWalletServer{})
			ctrl := NewController(c, ks, &acct, tx, tt.opts...)
			ctrl.Behavior.DryRun = true

			require.NoError(t, ctrl.ExecuteTransaction())
			require.Len(t, ctrl.tx.GetSignature(), 2)
			assert.Equal(t, prev, ctrl.tx.GetSignature()[0], "existing signature must be preserved")
			assert.Equal(t, want, ctrl.tx.GetSignature()[1])
			assert.Empty(t, ctrl.tx.GetPqAuthSig())
		})
	}
}

// TestExecuteTransaction_PQSignerKeepsExistingECDSASignature covers a mixed
// transaction: an ECDSA signature already present stays as is and the PQ
// signer adds exactly one pq_auth_sig entry.
func TestExecuteTransaction_PQSignerKeepsExistingECDSASignature(t *testing.T) {
	mock := &mockWalletServer{
		GetChainParametersFunc: func(context.Context, *api.EmptyMessage) (*core.ChainParameters, error) {
			return chainParamsWith("getAllowMlDsa44", 1), nil
		},
	}
	c := newMockClient(t, mock)
	s, key := newMLDSASigner(t)

	prev := make([]byte, 65)
	for i := range prev {
		prev[i] = byte(i + 1)
	}
	tx := newTestTransaction()
	tx.Signature = [][]byte{append([]byte(nil), prev...)}

	ctrl := NewController(c, nil, nil, tx, WithSigner(s))
	ctrl.Behavior.DryRun = true

	require.NoError(t, ctrl.ExecuteTransaction())
	require.Len(t, ctrl.tx.GetSignature(), 1)
	assert.Equal(t, prev, ctrl.tx.GetSignature()[0])
	require.Len(t, ctrl.tx.GetPqAuthSig(), 1)
	assert.Equal(t, key.Public().Bytes(), ctrl.tx.GetPqAuthSig()[0].GetPublicKey())
}
