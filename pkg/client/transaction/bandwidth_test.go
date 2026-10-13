package transaction

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func bandwidthTestTx(t *testing.T, types ...core.Transaction_Contract_ContractType) *core.Transaction {
	t.Helper()
	contracts := make([]*core.Transaction_Contract, 0, len(types))
	for _, ct := range types {
		param, err := anypb.New(&core.TransferContract{
			OwnerAddress: make([]byte, 21),
			ToAddress:    make([]byte, 21),
			Amount:       1_000_000,
		})
		require.NoError(t, err)
		contracts = append(contracts, &core.Transaction_Contract{Type: ct, Parameter: param})
	}
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{1, 2},
			RefBlockHash:  make([]byte, 8),
			Expiration:    1_700_000_060_000,
			Timestamp:     1_700_000_000_000,
			Contract:      contracts,
		},
		Signature: [][]byte{make([]byte, 65)},
	}
}

func TestEstimateBandwidth(t *testing.T) {
	transfer := core.Transaction_Contract_TransferContract
	shielded := core.Transaction_Contract_ShieldedTransferContract

	base := bandwidthTestTx(t, transfer)
	baseSize := int64(proto.Size(base))
	retOnly := &core.Transaction{Ret: []*core.Transaction_Result{{Ret: core.Transaction_Result_SUCESS, Fee: 100}}}
	require.Positive(t, proto.Size(retOnly))

	withRet := bandwidthTestTx(t, transfer)
	withRet.Ret = retOnly.Ret

	fnMax, err := pqc.MaxAuthSigSize(core.PQScheme_FN_DSA_512)
	require.NoError(t, err)
	mlMax, err := pqc.MaxAuthSigSize(core.PQScheme_ML_DSA_44)
	require.NoError(t, err)

	tests := []struct {
		name    string
		tx      *core.Transaction
		pending []core.PQScheme
		want    int64
	}{
		{name: "transfer", tx: base, want: baseSize + 64},
		{name: "ret is not counted", tx: withRet, want: baseSize + 64},
		{
			name: "two contracts",
			tx:   bandwidthTestTx(t, transfer, transfer),
			want: int64(proto.Size(bandwidthTestTx(t, transfer, transfer))) + 128,
		},
		{
			name: "shielded contract has no result allowance",
			tx:   bandwidthTestTx(t, shielded),
			want: int64(proto.Size(bandwidthTestTx(t, shielded))),
		},
		{
			name: "shielded and transfer",
			tx:   bandwidthTestTx(t, shielded, transfer),
			want: int64(proto.Size(bandwidthTestTx(t, shielded, transfer))) + 64,
		},
		{name: "no contracts", tx: &core.Transaction{}, want: 0},
		{name: "pending ML-DSA-44", tx: base, pending: []core.PQScheme{core.PQScheme_ML_DSA_44}, want: baseSize + 64 + int64(mlMax)},
		{name: "pending FN-DSA-512", tx: base, pending: []core.PQScheme{core.PQScheme_FN_DSA_512}, want: baseSize + 64 + int64(fnMax)},
		{
			name:    "pending both",
			tx:      withRet,
			pending: []core.PQScheme{core.PQScheme_FN_DSA_512, core.PQScheme_ML_DSA_44},
			want:    baseSize + 64 + int64(fnMax) + int64(mlMax),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := proto.Clone(tt.tx)
			got, err := EstimateBandwidth(tt.tx, tt.pending...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.True(t, proto.Equal(before, tt.tx), "the input must not be modified")
		})
	}
}

func TestEstimateBandwidth_PendingMatchesRealSignature(t *testing.T) {
	// Adding a maximum-size entry for real grows the transaction by exactly
	// the pending allowance.
	base := bandwidthTestTx(t, core.Transaction_Contract_TransferContract)
	signed := proto.Clone(base).(*core.Transaction)
	signed.PqAuthSig = []*core.PQAuthSig{{
		Scheme:    core.PQScheme_FN_DSA_512,
		PublicKey: make([]byte, pqc.FNDSA512PublicKeySize),
		Signature: make([]byte, pqc.FNDSA512SignatureMaxSize),
	}}
	estimated, err := EstimateBandwidth(base, core.PQScheme_FN_DSA_512)
	require.NoError(t, err)
	actual, err := EstimateBandwidth(signed)
	require.NoError(t, err)
	assert.Equal(t, actual, estimated)
}

func TestEstimateBandwidth_Errors(t *testing.T) {
	tx := bandwidthTestTx(t, core.Transaction_Contract_TransferContract)

	_, err := EstimateBandwidth(nil)
	require.Error(t, err)

	got, err := EstimateBandwidth(tx, core.PQScheme_ML_DSA_44, core.PQScheme_UNKNOWN_PQ_SCHEME)
	require.ErrorIs(t, err, pqc.ErrUnsupportedScheme)
	assert.Contains(t, err.Error(), "pending[1]")
	assert.Zero(t, got)
}
