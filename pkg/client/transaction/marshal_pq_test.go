package transaction

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// Golden ToJSON output for makeTestTx captured before pq_auth_sig support was
// added. Transactions without PQ signatures must keep serializing to exactly
// these bytes.
const (
	goldenToJSONNoSig = `{"visible":false,"txID":"f96ad67a985f1e60916aa365c537d884478762dc0a015d0826681c44dd4d56d0","raw_data":{"refBlockBytes":"Gis=","refBlockHash":"qrvM3e7/ABE=","expiration":"1700000000000","contract":[{"type":"TransferContract","parameter":{"@type":"type.googleapis.com/protocol.TransferContract"}}],"timestamp":"1699999990000"},"raw_data_hex":"0a021a2b2208aabbccddeeff00114080d095ffbc315a330801122f0a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637470f08195ffbc31"}`
	goldenToJSONSig   = `{"visible":false,"txID":"f96ad67a985f1e60916aa365c537d884478762dc0a015d0826681c44dd4d56d0","raw_data":{"refBlockBytes":"Gis=","refBlockHash":"qrvM3e7/ABE=","expiration":"1700000000000","contract":[{"type":"TransferContract","parameter":{"@type":"type.googleapis.com/protocol.TransferContract"}}],"timestamp":"1699999990000"},"raw_data_hex":"0a021a2b2208aabbccddeeff00114080d095ffbc315a330801122f0a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637470f08195ffbc31","signature":["aabbcc","0102"]}`
)

// pqBytes returns n deterministic non-trivial bytes derived from seed.
func pqBytes(seed byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = seed + byte(i*7)
	}
	return b
}

func fnEntry() *core.PQAuthSig {
	return &core.PQAuthSig{
		Scheme:    core.PQScheme_FN_DSA_512,
		PublicKey: pqBytes(0x10, 896),
		Signature: pqBytes(0x20, 650),
	}
}

func mlEntry() *core.PQAuthSig {
	return &core.PQAuthSig{
		Scheme:    core.PQScheme_ML_DSA_44,
		PublicKey: pqBytes(0x30, 1312),
		Signature: pqBytes(0x40, 2420),
	}
}

// pqJSONFixture builds a node-shaped envelope around makeTestTx's raw_data
// with a caller-supplied raw pq_auth_sig JSON value.
func pqJSONFixture(t *testing.T, pqRaw string) []byte {
	t.Helper()
	rawHex := marshalRawDataHex(t, makeTestTx())
	return []byte(fmt.Sprintf(`{"visible":false,"raw_data_hex":%q,"pq_auth_sig":%s}`, rawHex, pqRaw))
}

func assertPQEqual(t *testing.T, want, got []*core.PQAuthSig) {
	t.Helper()
	require.Len(t, got, len(want), "pq_auth_sig count")
	for i := range want {
		assert.Equal(t, want[i].GetScheme(), got[i].GetScheme(), "scheme[%d]", i)
		assert.True(t, bytes.Equal(want[i].GetPublicKey(), got[i].GetPublicKey()), "public_key[%d]", i)
		assert.True(t, bytes.Equal(want[i].GetSignature(), got[i].GetSignature()), "signature[%d]", i)
	}
}

func TestToJSON_NoPQ_ByteIdenticalGolden(t *testing.T) {
	got, err := ToJSON(makeTestTx())
	require.NoError(t, err, "ToJSON")
	assert.Equal(t, goldenToJSONNoSig, string(got), "no-signature output changed")
	assert.NotContains(t, string(got), "pq_auth_sig")

	tx := makeTestTx()
	tx.Signature = [][]byte{{0xaa, 0xbb, 0xcc}, {0x01, 0x02}}
	got, err = ToJSON(tx)
	require.NoError(t, err, "ToJSON with ECDSA signatures")
	assert.Equal(t, goldenToJSONSig, string(got), "ECDSA-only output changed")

	// An explicitly empty (non-nil) slice must also stay absent.
	tx.PqAuthSig = []*core.PQAuthSig{}
	got, err = ToJSON(tx)
	require.NoError(t, err, "ToJSON with empty pq slice")
	assert.Equal(t, goldenToJSONSig, string(got), "empty pq_auth_sig must be omitted")
}

func TestToJSON_PQShape(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{{0xaa, 0xbb, 0xcc}}
	tx.PqAuthSig = []*core.PQAuthSig{
		{Scheme: core.PQScheme_FN_DSA_512, PublicKey: []byte{0x01, 0xab}, Signature: []byte{0xcd, 0xef}},
		{Scheme: core.PQScheme_ML_DSA_44, PublicKey: []byte{0x02}, Signature: []byte{0x03}},
	}

	out, err := ToJSON(tx)
	require.NoError(t, err, "ToJSON")

	// pq_auth_sig follows signature; enums by name, bytes as lowercase hex.
	want := `"signature":["aabbcc"],"pq_auth_sig":[` +
		`{"scheme":"FN_DSA_512","public_key":"01ab","signature":"cdef"},` +
		`{"scheme":"ML_DSA_44","public_key":"02","signature":"03"}]}`
	assert.True(t, strings.HasSuffix(string(out), want), "unexpected tail: %s", out)
}

func TestToJSON_PQNilEntry(t *testing.T) {
	tx := makeTestTx()
	tx.PqAuthSig = []*core.PQAuthSig{fnEntry(), nil}

	_, err := ToJSON(tx)
	require.Error(t, err, "nil pq_auth_sig entry must be an error")
	assert.Contains(t, err.Error(), "pq_auth_sig[1]")
}

func TestRoundTrip_JSON_PQ(t *testing.T) {
	tests := []struct {
		name string
		pq   []*core.PQAuthSig
	}{
		{"single FN-DSA-512", []*core.PQAuthSig{fnEntry()}},
		{"single ML-DSA-44", []*core.PQAuthSig{mlEntry()}},
		{"two entries FN then ML", []*core.PQAuthSig{fnEntry(), mlEntry()}},
		{"two entries ML then FN", []*core.PQAuthSig{mlEntry(), fnEntry()}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := makeTestTx()
			tx.PqAuthSig = tt.pq

			out, err := ToJSON(tx)
			require.NoError(t, err, "ToJSON")
			got, err := FromJSON(out)
			require.NoError(t, err, "FromJSON")

			assert.True(t, proto.Equal(tx.GetRawData(), got.GetRawData()), "raw_data mismatch")
			assert.Empty(t, got.GetSignature(), "no ECDSA signature expected")
			assertPQEqual(t, tt.pq, got.GetPqAuthSig())
			assert.True(t, proto.Equal(tx, got), "transaction mismatch")
		})
	}
}

func TestRoundTrip_JSON_HybridECDSAAndPQ(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{{0xde, 0xad}, {0xbe, 0xef}}
	tx.PqAuthSig = []*core.PQAuthSig{fnEntry(), mlEntry()}

	out, err := ToJSON(tx)
	require.NoError(t, err, "ToJSON")
	got, err := FromJSON(out)
	require.NoError(t, err, "FromJSON")

	assert.Equal(t, tx.GetSignature(), got.GetSignature(), "ECDSA signatures")
	assertPQEqual(t, tx.GetPqAuthSig(), got.GetPqAuthSig())
	assert.True(t, proto.Equal(tx, got), "transaction mismatch")
}

func TestFromJSON_PQDoesNotAffectTxID(t *testing.T) {
	tx := makeTestTx()
	plain, err := ToJSON(tx)
	require.NoError(t, err)
	tx.PqAuthSig = []*core.PQAuthSig{fnEntry()}
	withPQ, err := ToJSON(tx)
	require.NoError(t, err)

	var a, b jsonTransaction
	require.NoError(t, json.Unmarshal(plain, &a))
	require.NoError(t, json.Unmarshal(withPQ, &b))
	assert.Equal(t, a.TxID, b.TxID, "pq_auth_sig must not change txID")
	assert.Equal(t, a.RawDataHex, b.RawDataHex, "pq_auth_sig must not change raw_data_hex")

	// txID is still validated when pq_auth_sig is present.
	var env map[string]any
	require.NoError(t, json.Unmarshal(withPQ, &env))
	env["txID"] = strings.Repeat("00", 32)
	bad, err := json.Marshal(env)
	require.NoError(t, err)
	_, err = FromJSON(bad)
	require.ErrorIs(t, err, ErrTxIDMismatch)
}

func TestFromJSON_PQJavaTronFixture(t *testing.T) {
	pk := pqBytes(0x55, 896)
	sig := pqBytes(0x66, 650)
	rawHex := marshalRawDataHex(t, makeTestTx())
	// Shape printed by java-tron's HTTP JsonFormat: enum by name, bytes as hex,
	// snake_case names, fields in field-number order.
	fixture := fmt.Sprintf(`{
  "visible": false,
  "txID": "f96ad67a985f1e60916aa365c537d884478762dc0a015d0826681c44dd4d56d0",
  "raw_data": {},
  "raw_data_hex": %q,
  "pq_auth_sig": [
    {"scheme": "FN_DSA_512", "public_key": %q, "signature": %q}
  ]
}`, rawHex, hex.EncodeToString(pk), hex.EncodeToString(sig))

	tx, err := FromJSON([]byte(fixture))
	require.NoError(t, err, "FromJSON")
	assertPQEqual(t, []*core.PQAuthSig{{Scheme: core.PQScheme_FN_DSA_512, PublicKey: pk, Signature: sig}}, tx.GetPqAuthSig())
	assert.Empty(t, tx.GetSignature())
}

func TestFromJSON_PQ0xPrefixAndUppercase(t *testing.T) {
	data := pqJSONFixture(t, `[{"scheme":"ML_DSA_44","public_key":"0xABcd","signature":"0X0102"}]`)
	tx, err := FromJSON(data)
	require.NoError(t, err)
	assertPQEqual(t, []*core.PQAuthSig{{Scheme: core.PQScheme_ML_DSA_44, PublicKey: []byte{0xab, 0xcd}, Signature: []byte{0x01, 0x02}}}, tx.GetPqAuthSig())
}

func TestFromJSON_PQIntegerScheme(t *testing.T) {
	tests := []struct {
		raw  string
		want core.PQScheme
	}{
		{"1", core.PQScheme_FN_DSA_512},
		{"2", core.PQScheme_ML_DSA_44},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			data := pqJSONFixture(t, `[{"scheme":`+tt.raw+`,"public_key":"0102","signature":"0304"}]`)
			tx, err := FromJSON(data)
			require.NoError(t, err)
			require.Len(t, tx.GetPqAuthSig(), 1)
			assert.Equal(t, tt.want, tx.GetPqAuthSig()[0].GetScheme())
		})
	}
}

func TestFromJSON_PQInvalid(t *testing.T) {
	tests := []struct {
		name    string
		pq      string
		wantErr string
	}{
		{"unknown scheme name", `[{"scheme":"RSA_2048","public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"lowercase scheme name", `[{"scheme":"fn_dsa_512","public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"numeric string scheme", `[{"scheme":"1","public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"unknown scheme number", `[{"scheme":3,"public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"negative scheme number", `[{"scheme":-1,"public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"fractional scheme number", `[{"scheme":1.5,"public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"scheme zero", `[{"scheme":0,"public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"scheme UNKNOWN_PQ_SCHEME", `[{"scheme":"UNKNOWN_PQ_SCHEME","public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"scheme missing", `[{"public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"scheme null", `[{"scheme":null,"public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"scheme bool", `[{"scheme":true,"public_key":"01","signature":"02"}]`, "pq_auth_sig[0]: unknown scheme"},
		{"null entry", `[null]`, "pq_auth_sig[0]"},
		{"bad public_key hex", `[{"scheme":"FN_DSA_512","public_key":"zz","signature":"02"}]`, "pq_auth_sig[0]: invalid public_key hex"},
		{"odd-length public_key hex", `[{"scheme":"FN_DSA_512","public_key":"abc","signature":"02"}]`, "pq_auth_sig[0]: invalid public_key hex"},
		{"bad signature hex on second entry", `[{"scheme":"FN_DSA_512","public_key":"01","signature":"02"},{"scheme":"ML_DSA_44","public_key":"01","signature":"qq"}]`, "pq_auth_sig[1]: invalid signature hex"},
		{"empty public_key", `[{"scheme":"FN_DSA_512","public_key":"","signature":"02"}]`, "pq_auth_sig[0]: empty public_key"},
		{"missing public_key", `[{"scheme":"FN_DSA_512","signature":"02"}]`, "pq_auth_sig[0]: empty public_key"},
		{"0x-only public_key", `[{"scheme":"FN_DSA_512","public_key":"0x","signature":"02"}]`, "pq_auth_sig[0]: empty public_key"},
		{"empty signature", `[{"scheme":"FN_DSA_512","public_key":"01","signature":""}]`, "pq_auth_sig[0]: empty signature"},
		{"missing signature", `[{"scheme":"FN_DSA_512","public_key":"01"}]`, "pq_auth_sig[0]: empty signature"},
		{"not an array", `{"scheme":"FN_DSA_512"}`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx, err := FromJSON(pqJSONFixture(t, tt.pq))
			require.Error(t, err)
			assert.Nil(t, tx)
			assert.ErrorIs(t, err, ErrInvalidTxJSON)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestFromJSON_PQErrorsDoNotEchoContent(t *testing.T) {
	const secret = "SECRETCONTENT"
	for _, pq := range []string{
		`[{"scheme":"` + secret + `","public_key":"01","signature":"02"}]`,
		`[{"scheme":"FN_DSA_512","public_key":"` + secret + `","signature":"02"}]`,
		`[{"scheme":"FN_DSA_512","public_key":"01","signature":"` + secret + `"}]`,
	} {
		_, err := FromJSON(pqJSONFixture(t, pq))
		require.Error(t, err)
		assert.NotContains(t, err.Error(), secret)
		assert.NotContains(t, strings.ToLower(err.Error()), strings.ToLower(secret))
	}

	// The hex decoder's own error quotes the offending byte (a single
	// character, which the substring checks above cannot catch), so pin the
	// exact messages: any appended detail fails.
	for _, tt := range []struct{ pq, want string }{
		{`[{"scheme":"FN_DSA_512","public_key":"ab` + secret + `","signature":"02"}]`,
			"invalid transaction JSON: pq_auth_sig[0]: invalid public_key hex"},
		{`[{"scheme":"FN_DSA_512","public_key":"01","signature":"0x` + secret + `"}]`,
			"invalid transaction JSON: pq_auth_sig[0]: invalid signature hex"},
	} {
		_, err := FromJSON(pqJSONFixture(t, tt.pq))
		require.EqualError(t, err, tt.want)
	}
}

func TestFromJSON_NoPQField(t *testing.T) {
	tx, err := FromJSON([]byte(goldenToJSONSig))
	require.NoError(t, err)
	assert.Empty(t, tx.GetPqAuthSig())
	assert.Len(t, tx.GetSignature(), 2)
}

func TestFromRawDataHex_CarriesNoPQ(t *testing.T) {
	tx, err := FromRawDataHex(marshalRawDataHex(t, makeTestTx()), []byte{0x01})
	require.NoError(t, err)
	assert.Empty(t, tx.GetPqAuthSig())
}

// ToJSON refuses every entry FromJSON would reject, so its output always
// round-trips.
func TestToJSON_PQInvalidEntryRefused(t *testing.T) {
	tests := []struct {
		name    string
		entry   *core.PQAuthSig
		wantErr string
	}{
		{"unknown scheme 0", &core.PQAuthSig{Scheme: core.PQScheme_UNKNOWN_PQ_SCHEME, PublicKey: []byte{1}, Signature: []byte{2}}, "pq_auth_sig[0]: unknown scheme 0"},
		{"out-of-range scheme", &core.PQAuthSig{Scheme: core.PQScheme(99), PublicKey: []byte{1}, Signature: []byte{2}}, "pq_auth_sig[0]: unknown scheme 99"},
		{"empty public key", &core.PQAuthSig{Scheme: core.PQScheme_FN_DSA_512, Signature: []byte{2}}, "pq_auth_sig[0]: empty public_key"},
		{"empty signature", &core.PQAuthSig{Scheme: core.PQScheme_ML_DSA_44, PublicKey: []byte{1}}, "pq_auth_sig[0]: empty signature"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := makeTestTx()
			tx.PqAuthSig = []*core.PQAuthSig{tt.entry}
			out, err := ToJSON(tx)
			require.EqualError(t, err, tt.wantErr)
			assert.Nil(t, out)
		})
	}
}
