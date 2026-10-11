//go:build integration

package client_test

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/fbsobreira/gotron-sdk/pkg/txcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const mainnetEndpoint = "grpc.trongrid.io:50051"

// TestIntegrationPQ_NileCapabilities checks that Nile advertises FN-DSA-512.
// ML-DSA-44 is only logged: Nile may enable it at any time.
func TestIntegrationPQ_NileCapabilities(t *testing.T) {
	c := newIntegrationClient(t)

	caps, err := c.GetPQCapabilities()
	require.NoError(t, err)
	assert.True(t, caps.FNDSA512, "FN-DSA-512 should be enabled on Nile")
	t.Logf("Nile PQ capabilities: FN-DSA-512=%v ML-DSA-44=%v", caps.FNDSA512, caps.MLDSA44)
}

// TestIntegrationPQ_MainnetCapabilities checks that mainnet has no PQ scheme enabled.
func TestIntegrationPQ_MainnetCapabilities(t *testing.T) {
	mc := client.NewGrpcClientWithTimeout(mainnetEndpoint, integrationTimeout)
	if err := mc.Start(grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		t.Skipf("cannot connect to mainnet at %s: %v", mainnetEndpoint, err)
	}
	t.Cleanup(mc.Stop)

	caps, err := mc.GetPQCapabilities()
	require.NoError(t, err)
	assert.False(t, caps.FNDSA512, "FN-DSA-512 should be disabled on mainnet")
	assert.False(t, caps.MLDSA44, "ML-DSA-44 should be disabled on mainnet")
}

// TestIntegrationPQ_MLDSARefusedBeforeBroadcast verifies txcore.Send refuses to
// sign or broadcast with an ML-DSA-44 key while Nile has the scheme disabled.
// Nothing is broadcast: the node only builds the transaction.
func TestIntegrationPQ_MLDSARefusedBeforeBroadcast(t *testing.T) {
	c := newIntegrationClient(t)

	caps, err := c.GetPQCapabilities()
	require.NoError(t, err)
	if caps.MLDSA44 {
		t.Skip("ML-DSA-44 is enabled on Nile; the refusal path cannot be exercised")
	}

	key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, rand.Reader)
	require.NoError(t, err)
	t.Cleanup(key.Destroy)
	s, err := signer.NewPQSigner(key)
	require.NoError(t, err)

	txe, err := c.Transfer(nileTestAddress, nileTestAddress2, 1)
	require.NoError(t, err)
	tx := txe.GetTransaction()
	require.NotNil(t, tx)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()
	_, err = txcore.Send(ctx, c, s, tx)
	require.Error(t, err)
	assert.True(t, errors.Is(err, txcore.ErrPQSchemeDisabled), "unexpected error: %v", err)
	assert.Empty(t, tx.GetSignature())
	assert.Empty(t, tx.GetPqAuthSig())
}
