package transaction

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// TIP-899 wire checks for the vendored Nile protos (proto/patches/tip899-pq.patch).

func TestPQ_ECDSATransactionBytesUnchanged(t *testing.T) {
	tx, err := FromJSON([]byte(realSignedTxJSON))
	require.NoError(t, err, "FromJSON real signed tx")

	// Anchor to the node-produced raw_data_hex and txID, not to SDK re-marshaling.
	var fixture struct {
		TxID       string `json:"txID"`
		RawDataHex string `json:"raw_data_hex"`
	}
	require.NoError(t, json.Unmarshal([]byte(realSignedTxJSON), &fixture), "decode fixture")
	rawHex, err := ToRawDataHex(tx)
	require.NoError(t, err, "ToRawDataHex")
	require.Equal(t, fixture.RawDataHex, rawHex, "raw_data differs from node bytes")
	txID, err := computeTxID(tx)
	require.NoError(t, err, "computeTxID")
	require.Equal(t, fixture.TxID, txID, "txID differs from node")

	// Build the expected wire bytes by hand: raw_data (field 1) then signature (field 2).
	raw, err := hex.DecodeString(rawHex)
	require.NoError(t, err, "decode raw_data_hex")
	var want []byte
	want = protowire.AppendTag(want, 1, protowire.BytesType)
	want = protowire.AppendBytes(want, raw)
	want = protowire.AppendTag(want, 2, protowire.BytesType)
	want = protowire.AppendBytes(want, tx.GetSignature()[0])

	got, err := proto.Marshal(tx)
	require.NoError(t, err, "marshal")
	assert.Equal(t, hex.EncodeToString(want), hex.EncodeToString(got), "ECDSA tx bytes changed")

	// An empty (non-nil) pq_auth_sig is proto3-omitted too.
	tx.PqAuthSig = []*core.PQAuthSig{}
	got, err = proto.Marshal(tx)
	require.NoError(t, err, "marshal with empty pq_auth_sig")
	assert.Equal(t, hex.EncodeToString(want), hex.EncodeToString(got), "empty pq_auth_sig changed bytes")

	// Round trip stays byte-identical.
	var decoded core.Transaction
	require.NoError(t, proto.Unmarshal(want, &decoded), "unmarshal")
	again, err := proto.Marshal(&decoded)
	require.NoError(t, err, "re-marshal")
	assert.Equal(t, want, again, "round trip changed bytes")
}

func TestPQ_AuthSigWireEncoding(t *testing.T) {
	tx := makeTestTx()
	idBefore, err := computeTxID(tx)
	require.NoError(t, err, "computeTxID before")

	sig := &core.PQAuthSig{
		Scheme:    core.PQScheme_FN_DSA_512,
		PublicKey: []byte{0x01, 0x02},
		Signature: []byte{0x39, 0xaa},
	}
	tx.PqAuthSig = []*core.PQAuthSig{sig}

	raw, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err, "marshal raw_data")
	sigBytes, err := proto.Marshal(sig)
	require.NoError(t, err, "marshal PQAuthSig")
	var want []byte
	want = protowire.AppendTag(want, 1, protowire.BytesType)
	want = protowire.AppendBytes(want, raw)
	want = protowire.AppendTag(want, 6, protowire.BytesType) // 0x32
	want = protowire.AppendBytes(want, sigBytes)

	got, err := proto.Marshal(tx)
	require.NoError(t, err, "marshal tx")
	assert.Equal(t, hex.EncodeToString(want), hex.EncodeToString(got), "pq_auth_sig wire bytes")
	// PQAuthSig itself: scheme=1 (varint), public_key=2, signature=3.
	assert.Equal(t, "0801120201021a0239aa", hex.EncodeToString(sigBytes), "PQAuthSig wire bytes")

	// The txid commits raw_data only, never pq_auth_sig.
	idAfter, err := computeTxID(tx)
	require.NoError(t, err, "computeTxID after")
	assert.Equal(t, idBefore, idAfter, "pq_auth_sig changed the txid")

	var decoded core.Transaction
	require.NoError(t, proto.Unmarshal(got, &decoded), "unmarshal")
	require.Len(t, decoded.GetPqAuthSig(), 1, "pq_auth_sig entries")
	assert.True(t, proto.Equal(sig, decoded.GetPqAuthSig()[0]), "PQAuthSig round trip")
}

func TestPQ_SchemaMatchesTIP899(t *testing.T) {
	tests := []struct {
		name    string
		msg     protoreflect.MessageDescriptor
		field   protoreflect.Name
		number  protoreflect.FieldNumber
		message protoreflect.FullName // element type for message fields
		enum    protoreflect.FullName // type for enum fields
		list    bool
	}{
		{"Transaction.pq_auth_sig", (&core.Transaction{}).ProtoReflect().Descriptor(), "pq_auth_sig", 6, "protocol.PQAuthSig", "", true},
		{"BlockHeader.pq_auth_sig", (&core.BlockHeader{}).ProtoReflect().Descriptor(), "pq_auth_sig", 3, "protocol.PQAuthSig", "", false},
		{"HelloMessage.pq_auth_sig", (&core.HelloMessage{}).ProtoReflect().Descriptor(), "pq_auth_sig", 12, "protocol.PQAuthSig", "", false},
		{"PQAuthSig.scheme", (&core.PQAuthSig{}).ProtoReflect().Descriptor(), "scheme", 1, "", "protocol.PQScheme", false},
		{"PQAuthSig.public_key", (&core.PQAuthSig{}).ProtoReflect().Descriptor(), "public_key", 2, "", "", false},
		{"PQAuthSig.signature", (&core.PQAuthSig{}).ProtoReflect().Descriptor(), "signature", 3, "", "", false},
		{"CanDelegatedMaxSizeRequestMessage.pq_scheme", (&api.CanDelegatedMaxSizeRequestMessage{}).ProtoReflect().Descriptor(), "pq_scheme", 3, "", "protocol.PQScheme", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fd := tt.msg.Fields().ByName(tt.field)
			require.NotNil(t, fd, "field missing")
			assert.Equal(t, tt.number, fd.Number(), "field number")
			assert.Equal(t, tt.list, fd.IsList(), "repeated")
			if tt.message != "" {
				require.NotNil(t, fd.Message(), "not a message field")
				assert.Equal(t, tt.message, fd.Message().FullName(), "message type")
			}
			if tt.enum != "" {
				require.NotNil(t, fd.Enum(), "not an enum field")
				assert.Equal(t, tt.enum, fd.Enum().FullName(), "enum type")
			}
		})
	}

	assert.Equal(t, int32(0), int32(core.PQScheme_UNKNOWN_PQ_SCHEME), "UNKNOWN_PQ_SCHEME")
	assert.Equal(t, int32(1), int32(core.PQScheme_FN_DSA_512), "FN_DSA_512")
	assert.Equal(t, int32(2), int32(core.PQScheme_ML_DSA_44), "ML_DSA_44")
}
