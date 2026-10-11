//go:build falcon

package transaction

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSignTransactionPQ_FNDSA512 signs a transaction with a real FN-DSA-512
// key through the liboqs backend, directly and through NewPQSigner.
func TestSignTransactionPQ_FNDSA512(t *testing.T) {
	require.NoError(t, pqc.SchemeAvailable(core.PQScheme_FN_DSA_512))
	key, err := pqc.GenerateKey(core.PQScheme_FN_DSA_512, nil)
	require.NoError(t, err)

	tx, err := SignTransactionPQ(newPQTx(), key)
	require.NoError(t, err)
	requirePQSigned(t, tx, key)
	sig := tx.GetPqAuthSig()[0].GetSignature()
	assert.GreaterOrEqual(t, len(sig), pqc.FNDSA512SignatureMinSize)
	assert.LessOrEqual(t, len(sig), pqc.FNDSA512SignatureMaxSize)
	assert.Equal(t, byte(pqc.FNDSA512SignatureHeader), sig[0])

	s, err := signer.NewPQSigner(key)
	require.NoError(t, err)
	assert.Equal(t, key.Public().Address(), s.Address())
	tx, err = s.Sign(newPQTx())
	require.NoError(t, err)
	requirePQSigned(t, tx, key)

	key.Destroy()
	_, err = s.Sign(newPQTx())
	require.ErrorIs(t, err, pqc.ErrKeyDestroyed)
}
