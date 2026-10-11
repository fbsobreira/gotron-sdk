package transaction

import (
	"crypto/sha256"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/signer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func newPQTx() *core.Transaction {
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x0a, 0x0b},
			RefBlockHash:  []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
			Expiration:    1_700_000_060_000,
			Timestamp:     1_700_000_000_000,
		},
	}
}

// requirePQSigned checks that tx carries exactly one pq_auth_sig, made by
// key over SHA-256(raw_data), and no ECDSA signature.
func requirePQSigned(t *testing.T, tx *core.Transaction, key pqc.PrivateKey) {
	t.Helper()
	require.Empty(t, tx.GetSignature())
	require.Len(t, tx.GetPqAuthSig(), 1)
	e := tx.GetPqAuthSig()[0]
	require.Equal(t, key.Scheme(), e.GetScheme())
	require.Equal(t, key.Public().Bytes(), e.GetPublicKey())

	raw, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err)
	digest := sha256.Sum256(raw)
	pub, err := pqc.ParsePublicKey(e.GetScheme(), e.GetPublicKey())
	require.NoError(t, err)
	ok, err := pub.Verify(digest[:], e.GetSignature())
	require.NoError(t, err)
	require.True(t, ok, "PQ signature does not verify against SHA-256(raw_data)")
}

func TestSignTransactionPQ_MLDSA44(t *testing.T) {
	key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, nil)
	require.NoError(t, err)
	tx := newPQTx()

	signed, err := SignTransactionPQ(tx, key)
	require.NoError(t, err)
	assert.Same(t, tx, signed)
	requirePQSigned(t, signed, key)

	_, err = SignTransactionPQ(tx, key)
	require.ErrorIs(t, err, signer.ErrPQAlreadySigned)
	assert.Len(t, tx.GetPqAuthSig(), 1)
}

func TestSignTransactionPQ_Refusals(t *testing.T) {
	assert.Equal(t, signer.MaxTotalSignatures, MaxTotalSignatures)
	assert.Equal(t, 5, MaxTotalSignatures)

	key, err := pqc.GenerateKey(core.PQScheme_ML_DSA_44, nil)
	require.NoError(t, err)
	full := newPQTx()
	for i := 0; i < MaxTotalSignatures; i++ {
		full.Signature = append(full.Signature, make([]byte, 65))
	}

	tests := []struct {
		name string
		tx   *core.Transaction
		key  pqc.PrivateKey
		err  error
	}{
		{"nil tx", nil, key, nil},
		{"nil raw_data", &core.Transaction{}, key, nil},
		{"nil key", newPQTx(), nil, nil},
		{"signature cap", full, key, signer.ErrTooManySignatures},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			signed, err := SignTransactionPQ(tc.tx, tc.key)
			require.Error(t, err)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			}
			assert.Nil(t, signed)
			assert.Empty(t, tc.tx.GetPqAuthSig())
		})
	}
}
