package pqc_test

import (
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/pqc"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestParseScheme(t *testing.T) {
	tests := []struct {
		in      string
		want    core.PQScheme
		wantErr bool
	}{
		{in: "fn-dsa-512", want: core.PQScheme_FN_DSA_512},
		{in: "FN_DSA_512", want: core.PQScheme_FN_DSA_512},
		{in: "Fn-Dsa_512", want: core.PQScheme_FN_DSA_512},
		{in: "  fn-dsa-512\t\n", want: core.PQScheme_FN_DSA_512},
		{in: "ml-dsa-44", want: core.PQScheme_ML_DSA_44},
		{in: "ML_DSA_44", want: core.PQScheme_ML_DSA_44},
		{in: " Ml_dsa-44 ", want: core.PQScheme_ML_DSA_44},
		{in: "", wantErr: true},
		{in: "   ", wantErr: true},
		{in: "UNKNOWN_PQ_SCHEME", wantErr: true},
		{in: "unknown-pq-scheme", wantErr: true},
		{in: "ecdsa", wantErr: true},
		{in: "fn-dsa-1024", wantErr: true},
		{in: "fn dsa 512", wantErr: true},
		{in: "fndsa512", wantErr: true},
		{in: "ml-dsa-44x", wantErr: true},
		{in: strings.Repeat("x", 500), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := pqc.ParseScheme(tt.in)
			if tt.wantErr {
				require.ErrorIs(t, err, pqc.ErrUnsupportedScheme)
				assert.Equal(t, core.PQScheme_UNKNOWN_PQ_SCHEME, got)
				assert.Less(t, len(err.Error()), 120, "a long input must be truncated in the message")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMaxAuthSigSize(t *testing.T) {
	tests := []struct {
		name   string
		scheme core.PQScheme
		pubLen int
		sigLen int
		// wire is the serialized size of one max-size pq_auth_sig entry in a
		// Transaction: field tag, length prefix, scheme, public key and the
		// largest signature, each with its own tag and length.
		wire int
	}{
		{"FN-DSA-512", core.PQScheme_FN_DSA_512, pqc.FNDSA512PublicKeySize, pqc.FNDSA512SignatureMaxSize, 1574},
		{"ML-DSA-44", core.PQScheme_ML_DSA_44, pqc.MLDSA44PublicKeySize, pqc.MLDSA44SignatureSize, 3743},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := pqc.MaxAuthSigSize(tt.scheme)
			require.NoError(t, err)
			assert.Equal(t, tt.wire, got, "absolute wire size")

			// A populated transaction, so a tag or length prefix that grows
			// with the existing content would show up.
			base := &core.Transaction{
				RawData:   &core.TransactionRaw{Data: []byte("memo")},
				Signature: [][]byte{make([]byte, 65)},
				PqAuthSig: []*core.PQAuthSig{{
					Scheme:    tt.scheme,
					PublicKey: make([]byte, tt.pubLen),
					Signature: make([]byte, tt.sigLen),
				}},
			}
			more := proto.Clone(base).(*core.Transaction)
			more.PqAuthSig = append(more.PqAuthSig, &core.PQAuthSig{
				Scheme:    tt.scheme,
				PublicKey: make([]byte, tt.pubLen),
				Signature: make([]byte, tt.sigLen),
			})
			assert.Equal(t, proto.Size(more)-proto.Size(base), got)

			none := &core.Transaction{}
			one := &core.Transaction{PqAuthSig: base.PqAuthSig}
			assert.Equal(t, proto.Size(one)-proto.Size(none), got)

			// Never less than the raw key and signature bytes.
			assert.Greater(t, got, tt.pubLen+tt.sigLen)
		})
	}

	for _, scheme := range []core.PQScheme{core.PQScheme_UNKNOWN_PQ_SCHEME, core.PQScheme(99)} {
		t.Run(scheme.String(), func(t *testing.T) {
			got, err := pqc.MaxAuthSigSize(scheme)
			require.ErrorIs(t, err, pqc.ErrUnsupportedScheme)
			assert.Zero(t, got)
		})
	}
}
