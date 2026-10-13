package transaction

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPrivateKeySigner(t *testing.T) signer.Signer {
	t.Helper()
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	s, err := signer.NewPrivateKeySigner(key)
	require.NoError(t, err)
	return s
}

// TestExecuteTransaction_NotConnected checks that broadcasting without a
// connected client returns client.ErrNotConnected instead of panicking, and
// that the transaction was still signed.
func TestExecuteTransaction_NotConnected(t *testing.T) {
	tests := []struct {
		name   string
		client *client.GrpcClient
	}{
		{"nil client", nil},
		{"client never started", client.NewGrpcClient("127.0.0.1:1")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewController(tt.client, nil, nil, newTestTransaction(), WithSigner(newPrivateKeySigner(t)))
			ctrl.Behavior.ConfirmationWaitTime = 5

			err := ctrl.ExecuteTransaction()
			require.ErrorIs(t, err, client.ErrNotConnected)
			assert.Len(t, ctrl.Transaction().GetSignature(), 1, "signing runs before the broadcast")
			assert.Nil(t, ctrl.Result)
			assert.Nil(t, ctrl.Receipt)
		})
	}
}

// TestTxConfirmation_NotConnected checks that the confirmation loop stops at
// once when the client is not connected, rather than retrying until
// ConfirmationWaitTime runs out and reporting a misleading timeout.
func TestTxConfirmation_NotConnected(t *testing.T) {
	tests := []struct {
		name   string
		client *client.GrpcClient
	}{
		{"nil client", nil},
		{"client never started", client.NewGrpcClient("127.0.0.1:1")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := NewController(tt.client, nil, nil, newTestTransaction())
			ctrl.Behavior.ConfirmationWaitTime = 30

			start := time.Now()
			ctrl.txConfirmation()
			require.ErrorIs(t, ctrl.executionError, client.ErrNotConnected)
			assert.Less(t, time.Since(start), 2*time.Second, "a disconnected client must not be retried")
			assert.Nil(t, ctrl.Receipt)
		})
	}
}
