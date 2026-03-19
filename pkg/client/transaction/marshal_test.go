package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// makeTestTx creates a realistic Transaction for testing.
func makeTestTx() *core.Transaction {
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x1a, 0x2b},
			RefBlockHash:  []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00, 0x11},
			Expiration:    1700000000000,
			Timestamp:     1699999990000,
			Contract: []*core.Transaction_Contract{
				{
					Type:      core.Transaction_Contract_TransferContract,
					Parameter: &anypb.Any{TypeUrl: "type.googleapis.com/protocol.TransferContract"},
				},
			},
		},
	}
}

// marshalRawDataHex helper returns hex-encoded raw_data for a transaction.
func marshalRawDataHex(t *testing.T, tx *core.Transaction) string {
	t.Helper()
	rawBytes, err := proto.Marshal(tx.GetRawData())
	require.NoError(t, err, "failed to marshal raw_data")
	return hex.EncodeToString(rawBytes)
}

func TestFromRawDataHex(t *testing.T) {
	tx := makeTestTx()
	rawHex := marshalRawDataHex(t, tx)

	got, err := FromRawDataHex(rawHex)
	require.NoError(t, err, "FromRawDataHex")

	assert.True(t, proto.Equal(tx.GetRawData(), got.GetRawData()), "raw_data mismatch after FromRawDataHex")
	assert.Empty(t, got.GetSignature(), "expected no signatures")
}

func TestFromRawDataHex_With0xPrefix(t *testing.T) {
	tx := makeTestTx()
	rawHex := "0x" + marshalRawDataHex(t, tx)

	got, err := FromRawDataHex(rawHex)
	require.NoError(t, err, "FromRawDataHex with 0x prefix")

	assert.True(t, proto.Equal(tx.GetRawData(), got.GetRawData()), "raw_data mismatch with 0x prefix")
}

func TestFromRawDataHex_WithSignatures(t *testing.T) {
	tx := makeTestTx()
	rawHex := marshalRawDataHex(t, tx)

	sig1 := []byte("signature-one-fake-data-65bytes-padded-to-be-realistic-enough!!")
	sig2 := []byte("signature-two-fake-data-65bytes-padded-to-be-realistic-enough!!")

	got, err := FromRawDataHex(rawHex, sig1, sig2)
	require.NoError(t, err, "FromRawDataHex with sigs")

	require.Len(t, got.GetSignature(), 2, "expected 2 signatures")
	assert.Equal(t, string(sig1), string(got.GetSignature()[0]), "signature[0] mismatch")
	assert.Equal(t, string(sig2), string(got.GetSignature()[1]), "signature[1] mismatch")
}

func TestFromRawDataHex_EmptyString(t *testing.T) {
	_, err := FromRawDataHex("")
	require.Error(t, err, "expected error for empty string")
	assert.ErrorIs(t, err, ErrEmptyRawData)
}

func TestFromRawDataHex_InvalidHex(t *testing.T) {
	_, err := FromRawDataHex("not-valid-hex!")
	require.Error(t, err, "expected error for invalid hex")
	assert.Contains(t, err.Error(), "invalid hex")
}

func TestFromRawDataHex_InvalidProtobuf(t *testing.T) {
	// Valid hex but invalid wire-format protobuf data.
	_, err := FromRawDataHex("deadbeef")
	require.Error(t, err, "expected error for invalid protobuf data")
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestToRawDataHex(t *testing.T) {
	tx := makeTestTx()

	gotHex, err := ToRawDataHex(tx)
	require.NoError(t, err, "ToRawDataHex")
	require.NotEmpty(t, gotHex, "expected non-empty hex string")

	// Verify by decoding back
	rawBytes, err := hex.DecodeString(gotHex)
	require.NoError(t, err, "hex decode")
	rawData := &core.TransactionRaw{}
	require.NoError(t, proto.Unmarshal(rawBytes, rawData), "unmarshal")
	assert.True(t, proto.Equal(tx.GetRawData(), rawData), "round-trip via ToRawDataHex failed")
}

func TestToRawDataHex_NilRawData(t *testing.T) {
	_, err := ToRawDataHex(&core.Transaction{})
	assert.ErrorIs(t, err, ErrNilRawData)

	_, err = ToRawDataHex(nil)
	assert.ErrorIs(t, err, ErrNilRawData)
}

func TestToJSON_NilRawData(t *testing.T) {
	_, err := ToJSON(&core.Transaction{})
	assert.ErrorIs(t, err, ErrNilRawData)

	_, err = ToJSON(nil)
	assert.ErrorIs(t, err, ErrNilRawData)
}

func TestComputeTxID_NilRawData(t *testing.T) {
	_, err := computeTxID(&core.Transaction{})
	assert.ErrorIs(t, err, ErrNilRawData)

	_, err = computeTxID(nil)
	assert.ErrorIs(t, err, ErrNilRawData)
}

func TestRoundTrip_RawDataHex(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{
		{0x01, 0x02, 0x03},
		{0x04, 0x05, 0x06},
	}

	hexStr, err := ToRawDataHex(tx)
	require.NoError(t, err, "ToRawDataHex")

	got, err := FromRawDataHex(hexStr, tx.GetSignature()...)
	require.NoError(t, err, "FromRawDataHex")

	assert.True(t, proto.Equal(tx.GetRawData(), got.GetRawData()), "raw_data mismatch in round-trip")
	assert.Equal(t, len(tx.GetSignature()), len(got.GetSignature()), "signature count mismatch")
}

func TestToJSON(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{{0xaa, 0xbb, 0xcc}}

	jsonBytes, err := ToJSON(tx)
	require.NoError(t, err, "ToJSON")

	var jtx jsonTransaction
	require.NoError(t, json.Unmarshal(jsonBytes, &jtx), "unmarshal JSON output")

	assert.NotEmpty(t, jtx.TxID, "expected txID to be set")
	assert.NotEmpty(t, jtx.RawDataHex, "expected raw_data_hex to be set")
	require.Len(t, jtx.Signature, 1, "expected 1 signature")
	assert.Equal(t, "aabbcc", jtx.Signature[0], "expected signature hex 'aabbcc'")

	// Verify txID is correct
	rawBytes, _ := proto.Marshal(tx.GetRawData())
	h := sha256.Sum256(rawBytes)
	expectedID := hex.EncodeToString(h[:])
	assert.Equal(t, expectedID, jtx.TxID, "txID mismatch")
}

func TestToJSON_NoSignatures(t *testing.T) {
	tx := makeTestTx()

	jsonBytes, err := ToJSON(tx)
	require.NoError(t, err, "ToJSON")

	var jtx jsonTransaction
	require.NoError(t, json.Unmarshal(jsonBytes, &jtx), "unmarshal")

	// With omitempty, the signature field should be absent from JSON output
	assert.False(t, strings.Contains(string(jsonBytes), `"signature"`),
		"expected signature field to be omitted from JSON output")
}

func TestFromJSON(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{{0xde, 0xad}, {0xbe, 0xef}}

	jsonBytes, err := ToJSON(tx)
	require.NoError(t, err, "ToJSON")

	got, err := FromJSON(jsonBytes)
	require.NoError(t, err, "FromJSON")

	assert.True(t, proto.Equal(tx.GetRawData(), got.GetRawData()), "raw_data mismatch after FromJSON")
	require.Len(t, got.GetSignature(), 2, "expected 2 signatures")
}

func TestFromJSON_InvalidJSON(t *testing.T) {
	_, err := FromJSON([]byte("not json"))
	require.Error(t, err, "expected error for invalid JSON")
	assert.Contains(t, err.Error(), "invalid transaction JSON")
}

func TestFromJSON_MissingRawDataHex(t *testing.T) {
	data := `{"txID":"abc","signature":[]}`
	_, err := FromJSON([]byte(data))
	require.Error(t, err, "expected error for missing raw_data_hex")
	assert.Contains(t, err.Error(), "missing raw_data_hex")
}

func TestFromJSON_TxIDMismatch(t *testing.T) {
	tx := makeTestTx()
	jsonBytes, err := ToJSON(tx)
	require.NoError(t, err, "ToJSON")

	// Tamper with the txID
	var jtx map[string]any
	require.NoError(t, json.Unmarshal(jsonBytes, &jtx), "unmarshal")
	jtx["txID"] = "0000000000000000000000000000000000000000000000000000000000000000"
	tampered, _ := json.Marshal(jtx)

	_, err = FromJSON(tampered)
	require.Error(t, err, "expected error for txID mismatch")
	assert.Contains(t, err.Error(), "txID does not match")
}

func TestFromJSON_NoTxID(t *testing.T) {
	// When txID is absent, validation should be skipped
	tx := makeTestTx()
	rawHex := marshalRawDataHex(t, tx)

	data, _ := json.Marshal(map[string]any{
		"raw_data_hex": rawHex,
	})

	got, err := FromJSON(data)
	require.NoError(t, err, "FromJSON without txID")
	assert.True(t, proto.Equal(tx.GetRawData(), got.GetRawData()), "raw_data mismatch")
}

func TestFromJSON_InvalidSignatureHex(t *testing.T) {
	tx := makeTestTx()
	rawHex := marshalRawDataHex(t, tx)

	data, _ := json.Marshal(map[string]any{
		"raw_data_hex": rawHex,
		"signature":    []string{"not-valid-hex!"},
	})

	_, err := FromJSON(data)
	require.Error(t, err, "expected error for invalid signature hex")
	assert.Contains(t, err.Error(), "failed to decode signature")
}

func TestRoundTrip_JSON(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{
		{0x01, 0x02, 0x03, 0x04, 0x05},
	}

	jsonBytes, err := ToJSON(tx)
	require.NoError(t, err, "ToJSON")

	got, err := FromJSON(jsonBytes)
	require.NoError(t, err, "FromJSON")

	assert.True(t, proto.Equal(tx.GetRawData(), got.GetRawData()), "raw_data mismatch in JSON round-trip")
	require.Len(t, got.GetSignature(), len(tx.GetSignature()), "signature count mismatch")

	for i := range tx.GetSignature() {
		assert.Equal(t, hex.EncodeToString(tx.GetSignature()[i]),
			hex.EncodeToString(got.GetSignature()[i]), "signature[%d] mismatch", i)
	}
}

func TestRoundTrip_JSON_MultipleSignatures(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{
		{0xaa, 0xbb, 0xcc},
		{0xdd, 0xee, 0xff},
		{0x11, 0x22, 0x33},
	}

	jsonBytes, err := ToJSON(tx)
	require.NoError(t, err, "ToJSON")

	got, err := FromJSON(jsonBytes)
	require.NoError(t, err, "FromJSON")

	require.Len(t, got.GetSignature(), 3, "expected 3 signatures")
}

// Real transaction fixtures captured from Nile testnet TRON HTTP API.
// These verify our marshal/unmarshal is compatible with the actual API format.

// Signed TriggerSmartContract transaction from gettransactionbyid endpoint.
const realSignedTxJSON = `{
  "ret": [{"contractRet": "SUCCESS"}],
  "signature": [
    "1788bf0281f67c42f5a09322523edee9969e808a863776f21ab6543de9dd42ca0e4955577879df8074086c620eb96709ba9a95d60a166de190a081e8d69801da00"
  ],
  "txID": "c70b20da8cb8d96a99c1aceb283412acb9dea313a9e467178431a3c6ae8b9f08",
  "raw_data": {
    "contract": [{"parameter": {"value": {"data": "a9059cbb00000000000000000000004150125bea243a640bf85348873b6c4f3c517aee7a0000000000000000000000000000000000000000000000000000000000000001", "owner_address": "41495511a493d8c362be4267224e6d81013a6862ee", "contract_address": "41eca9bc828a3005b9a3b909f2cc5c2a54794de05f"}, "type_url": "type.googleapis.com/protocol.TriggerSmartContract"}, "type": "TriggerSmartContract"}],
    "ref_block_bytes": "8ad9",
    "ref_block_hash": "eaef69105495eb15",
    "expiration": 1773182914544,
    "fee_limit": 100000000,
    "timestamp": 1773182854544
  },
  "raw_data_hex": "0a028ad92208eaef69105495eb1540f09fc0cfcd335aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a1541495511a493d8c362be4267224e6d81013a6862ee121541eca9bc828a3005b9a3b909f2cc5c2a54794de05f2244a9059cbb00000000000000000000004150125bea243a640bf85348873b6c4f3c517aee7a00000000000000000000000000000000000000000000000000000000000000017090cbbccfcd33900180c2d72f"
}`

// Unsigned TransferContract transaction from createtransaction endpoint.
const realUnsignedTxJSON = `{
  "visible": false,
  "txID": "ae80c7aa55e19c2da5e712d4be50e4c9422dd6bfc99039cc42d5deb76938c0e7",
  "raw_data": {
    "contract": [{"parameter": {"value": {"amount": 1000, "owner_address": "41eca9bc828a3005b9a3b909f2cc5c2a54794de05f", "to_address": "41495511a493d8c362be4267224e6d81013a6862ee"}, "type_url": "type.googleapis.com/protocol.TransferContract"}, "type": "TransferContract"}],
    "ref_block_bytes": "8acb",
    "ref_block_hash": "7c571ea3e7e9dbf3",
    "expiration": 1773182925000,
    "timestamp": 1773182865805
  },
  "raw_data_hex": "0a028acb22087c571ea3e7e9dbf340c8f1c0cfcd335a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541eca9bc828a3005b9a3b909f2cc5c2a54794de05f121541495511a493d8c362be4267224e6d81013a6862ee18e807708da3bdcfcd33"
}`

func TestFromJSON_RealSignedTransaction(t *testing.T) {
	tx, err := FromJSON([]byte(realSignedTxJSON))
	require.NoError(t, err, "FromJSON real signed tx")

	// txID validation passed (FromJSON checks it internally), but verify explicitly
	computedID, err := computeTxID(tx)
	require.NoError(t, err, "computeTxID")
	expectedID := "c70b20da8cb8d96a99c1aceb283412acb9dea313a9e467178431a3c6ae8b9f08"
	assert.Equal(t, expectedID, computedID, "txID mismatch")

	// Verify signature was decoded
	require.Len(t, tx.GetSignature(), 1, "expected 1 signature")
	sigHex := hex.EncodeToString(tx.GetSignature()[0])
	expectedSig := "1788bf0281f67c42f5a09322523edee9969e808a863776f21ab6543de9dd42ca0e4955577879df8074086c620eb96709ba9a95d60a166de190a081e8d69801da00"
	assert.Equal(t, expectedSig, sigHex, "signature mismatch")

	// Verify raw_data was deserialized correctly
	raw := tx.GetRawData()
	assert.Equal(t, int64(1773182914544), raw.GetExpiration(), "expiration mismatch")
	assert.Equal(t, int64(1773182854544), raw.GetTimestamp(), "timestamp mismatch")
	assert.Equal(t, int64(100000000), raw.GetFeeLimit(), "fee_limit mismatch")
	require.Len(t, raw.GetContract(), 1, "expected 1 contract")
	assert.Equal(t, core.Transaction_Contract_TriggerSmartContract, raw.GetContract()[0].GetType(), "contract type mismatch")

	// Verify ToRawDataHex produces the same hex as the API
	gotHex, err := ToRawDataHex(tx)
	require.NoError(t, err, "ToRawDataHex")
	expectedHex := "0a028ad92208eaef69105495eb1540f09fc0cfcd335aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a1541495511a493d8c362be4267224e6d81013a6862ee121541eca9bc828a3005b9a3b909f2cc5c2a54794de05f2244a9059cbb00000000000000000000004150125bea243a640bf85348873b6c4f3c517aee7a00000000000000000000000000000000000000000000000000000000000000017090cbbccfcd33900180c2d72f"
	assert.Equal(t, expectedHex, gotHex, "ToRawDataHex mismatch")
}

func TestFromJSON_RealUnsignedTransaction(t *testing.T) {
	tx, err := FromJSON([]byte(realUnsignedTxJSON))
	require.NoError(t, err, "FromJSON real unsigned tx")

	// txID validation passed internally, verify explicitly
	computedID, err := computeTxID(tx)
	require.NoError(t, err, "computeTxID")
	expectedID := "ae80c7aa55e19c2da5e712d4be50e4c9422dd6bfc99039cc42d5deb76938c0e7"
	assert.Equal(t, expectedID, computedID, "txID mismatch")

	// No signatures on unsigned tx
	assert.Empty(t, tx.GetSignature(), "expected 0 signatures")

	// Verify raw_data fields
	raw := tx.GetRawData()
	assert.Equal(t, int64(1773182925000), raw.GetExpiration(), "expiration mismatch")
	require.Len(t, raw.GetContract(), 1, "expected 1 contract")
	assert.Equal(t, core.Transaction_Contract_TransferContract, raw.GetContract()[0].GetType(), "contract type mismatch")
}

func TestFromRawDataHex_RealTransaction(t *testing.T) {
	rawHex := "0a028acb22087c571ea3e7e9dbf340c8f1c0cfcd335a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541eca9bc828a3005b9a3b909f2cc5c2a54794de05f121541495511a493d8c362be4267224e6d81013a6862ee18e807708da3bdcfcd33"

	tx, err := FromRawDataHex(rawHex)
	require.NoError(t, err, "FromRawDataHex real tx")

	// Verify txID matches what the API returned
	computedID, err := computeTxID(tx)
	require.NoError(t, err, "computeTxID")
	expectedID := "ae80c7aa55e19c2da5e712d4be50e4c9422dd6bfc99039cc42d5deb76938c0e7"
	assert.Equal(t, expectedID, computedID, "txID mismatch")

	// Re-encoding should produce identical hex
	gotHex, err := ToRawDataHex(tx)
	require.NoError(t, err, "ToRawDataHex")
	assert.Equal(t, rawHex, gotHex, "re-encoded hex mismatch")
}

func TestComputeTxID(t *testing.T) {
	tx := makeTestTx()

	id, err := computeTxID(tx)
	require.NoError(t, err, "computeTxID")

	// Verify manually
	rawBytes, _ := proto.Marshal(tx.GetRawData())
	h := sha256.Sum256(rawBytes)
	expected := hex.EncodeToString(h[:])

	assert.Equal(t, expected, id, "txID mismatch")
}
