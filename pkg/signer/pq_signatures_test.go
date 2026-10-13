package signer

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPQSignatures(t *testing.T) {
	ml := newMLDSAKey(t)
	defer ml.Destroy()
	fn := fnFixtureKey(t)
	defer fn.Destroy()

	entry := func(k pqc.PrivateKey) *core.PQAuthSig {
		return &core.PQAuthSig{Scheme: k.Scheme(), PublicKey: k.Public().Bytes(), Signature: []byte{0x01}}
	}

	tests := []struct {
		name    string
		sigs    []*core.PQAuthSig
		want    []pqc.PrivateKey
		wantErr string
	}{
		{name: "none", sigs: nil},
		{name: "one ML-DSA-44", sigs: []*core.PQAuthSig{entry(ml)}, want: []pqc.PrivateKey{ml}},
		{name: "one FN-DSA-512", sigs: []*core.PQAuthSig{entry(fn)}, want: []pqc.PrivateKey{fn}},
		{
			name: "two schemes keep order",
			sigs: []*core.PQAuthSig{entry(fn), entry(ml)},
			want: []pqc.PrivateKey{fn, ml},
		},
		{
			name: "FN-DSA-512 key with the 0x09 header",
			sigs: []*core.PQAuthSig{{
				Scheme:    core.PQScheme_FN_DSA_512,
				PublicKey: append([]byte{0x09}, fn.Public().Bytes()...),
			}},
			want: []pqc.PrivateKey{fn},
		},
		{
			name:    "malformed key",
			sigs:    []*core.PQAuthSig{{Scheme: core.PQScheme_ML_DSA_44, PublicKey: []byte{1, 2, 3}}},
			wantErr: "pq_auth_sig[0]",
		},
		{
			name:    "malformed entry after a good one is not skipped",
			sigs:    []*core.PQAuthSig{entry(ml), {Scheme: core.PQScheme_FN_DSA_512, PublicKey: nil}},
			wantErr: "pq_auth_sig[1]",
		},
		{
			name:    "unsupported scheme",
			sigs:    []*core.PQAuthSig{entry(ml), entry(fn), {Scheme: core.PQScheme_UNKNOWN_PQ_SCHEME, PublicKey: make([]byte, 896)}},
			wantErr: "pq_auth_sig[2]",
		},
		{
			name:    "nil entry",
			sigs:    []*core.PQAuthSig{nil},
			wantErr: "pq_auth_sig[0]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := newPQTestTx()
			tx.PqAuthSig = tt.sigs
			got, err := PQSignatures(tx)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, len(tt.want))
			for i, k := range tt.want {
				assert.Equal(t, k.Scheme(), got[i].Scheme)
				assert.Equal(t, k.Public().Address(), got[i].Address)
				assert.True(t, k.Public().Equal(got[i].PublicKey))
			}
		})
	}
}

func TestPQSignaturesMalformedKeyWrapsCause(t *testing.T) {
	tx := newPQTestTx()
	tx.PqAuthSig = []*core.PQAuthSig{{Scheme: core.PQScheme_ML_DSA_44, PublicKey: []byte{1}}}
	_, err := PQSignatures(tx)
	require.ErrorIs(t, err, pqc.ErrInvalidKey)

	tx.PqAuthSig = []*core.PQAuthSig{{Scheme: core.PQScheme(99), PublicKey: []byte{1}}}
	_, err = PQSignatures(tx)
	require.ErrorIs(t, err, pqc.ErrUnsupportedScheme)
}

func TestPQSignaturesNilTransaction(t *testing.T) {
	got, err := PQSignatures(nil)
	require.Error(t, err)
	assert.Nil(t, got)
}
