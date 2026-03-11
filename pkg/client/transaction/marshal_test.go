package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
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
	if err != nil {
		t.Fatalf("failed to marshal raw_data: %v", err)
	}
	return hex.EncodeToString(rawBytes)
}

func TestFromRawDataHex(t *testing.T) {
	tx := makeTestTx()
	rawHex := marshalRawDataHex(t, tx)

	got, err := FromRawDataHex(rawHex)
	if err != nil {
		t.Fatalf("FromRawDataHex: %v", err)
	}

	if !proto.Equal(tx.GetRawData(), got.GetRawData()) {
		t.Error("raw_data mismatch after FromRawDataHex")
	}
	if len(got.GetSignature()) != 0 {
		t.Errorf("expected no signatures, got %d", len(got.GetSignature()))
	}
}

func TestFromRawDataHex_With0xPrefix(t *testing.T) {
	tx := makeTestTx()
	rawHex := "0x" + marshalRawDataHex(t, tx)

	got, err := FromRawDataHex(rawHex)
	if err != nil {
		t.Fatalf("FromRawDataHex with 0x prefix: %v", err)
	}

	if !proto.Equal(tx.GetRawData(), got.GetRawData()) {
		t.Error("raw_data mismatch with 0x prefix")
	}
}

func TestFromRawDataHex_WithSignatures(t *testing.T) {
	tx := makeTestTx()
	rawHex := marshalRawDataHex(t, tx)

	sig1 := []byte("signature-one-fake-data-65bytes-padded-to-be-realistic-enough!!")
	sig2 := []byte("signature-two-fake-data-65bytes-padded-to-be-realistic-enough!!")

	got, err := FromRawDataHex(rawHex, sig1, sig2)
	if err != nil {
		t.Fatalf("FromRawDataHex with sigs: %v", err)
	}

	if len(got.GetSignature()) != 2 {
		t.Fatalf("expected 2 signatures, got %d", len(got.GetSignature()))
	}
	if string(got.GetSignature()[0]) != string(sig1) {
		t.Error("signature[0] mismatch")
	}
	if string(got.GetSignature()[1]) != string(sig2) {
		t.Error("signature[1] mismatch")
	}
}

func TestFromRawDataHex_EmptyString(t *testing.T) {
	_, err := FromRawDataHex("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
	if err != ErrEmptyRawData {
		t.Errorf("expected ErrEmptyRawData, got: %v", err)
	}
}

func TestFromRawDataHex_InvalidHex(t *testing.T) {
	_, err := FromRawDataHex("not-valid-hex!")
	if err == nil {
		t.Fatal("expected error for invalid hex")
	}
	if !strings.Contains(err.Error(), "invalid hex") {
		t.Errorf("expected invalid hex error, got: %v", err)
	}
}

func TestFromRawDataHex_InvalidProtobuf(t *testing.T) {
	// Valid hex but not valid protobuf for TransactionRaw —
	// proto.Unmarshal is lenient with unknown fields, so we just verify it doesn't panic.
	_, err := FromRawDataHex("deadbeef")
	// This may or may not error depending on protobuf parsing; just ensure no panic.
	_ = err
}

func TestToRawDataHex(t *testing.T) {
	tx := makeTestTx()

	gotHex, err := ToRawDataHex(tx)
	if err != nil {
		t.Fatalf("ToRawDataHex: %v", err)
	}

	if gotHex == "" {
		t.Fatal("expected non-empty hex string")
	}

	// Verify by decoding back
	rawBytes, err := hex.DecodeString(gotHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	rawData := &core.TransactionRaw{}
	if err := proto.Unmarshal(rawBytes, rawData); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !proto.Equal(tx.GetRawData(), rawData) {
		t.Error("round-trip via ToRawDataHex failed")
	}
}

func TestToRawDataHex_NilRawData(t *testing.T) {
	_, err := ToRawDataHex(&core.Transaction{})
	if err != ErrNilRawData {
		t.Errorf("expected ErrNilRawData, got: %v", err)
	}

	_, err = ToRawDataHex(nil)
	if err != ErrNilRawData {
		t.Errorf("expected ErrNilRawData for nil tx, got: %v", err)
	}
}

func TestToJSON_NilRawData(t *testing.T) {
	_, err := ToJSON(&core.Transaction{})
	if err != ErrNilRawData {
		t.Errorf("expected ErrNilRawData, got: %v", err)
	}

	_, err = ToJSON(nil)
	if err != ErrNilRawData {
		t.Errorf("expected ErrNilRawData for nil tx, got: %v", err)
	}
}

func TestComputeTxID_NilRawData(t *testing.T) {
	_, err := computeTxID(&core.Transaction{})
	if err != ErrNilRawData {
		t.Errorf("expected ErrNilRawData, got: %v", err)
	}

	_, err = computeTxID(nil)
	if err != ErrNilRawData {
		t.Errorf("expected ErrNilRawData for nil tx, got: %v", err)
	}
}

func TestRoundTrip_RawDataHex(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{
		{0x01, 0x02, 0x03},
		{0x04, 0x05, 0x06},
	}

	hexStr, err := ToRawDataHex(tx)
	if err != nil {
		t.Fatalf("ToRawDataHex: %v", err)
	}

	got, err := FromRawDataHex(hexStr, tx.GetSignature()...)
	if err != nil {
		t.Fatalf("FromRawDataHex: %v", err)
	}

	if !proto.Equal(tx.GetRawData(), got.GetRawData()) {
		t.Error("raw_data mismatch in round-trip")
	}
	if len(got.GetSignature()) != len(tx.GetSignature()) {
		t.Errorf("signature count mismatch: %d vs %d", len(got.GetSignature()), len(tx.GetSignature()))
	}
}

func TestToJSON(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{{0xaa, 0xbb, 0xcc}}

	jsonBytes, err := ToJSON(tx)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var jtx jsonTransaction
	if err := json.Unmarshal(jsonBytes, &jtx); err != nil {
		t.Fatalf("unmarshal JSON output: %v", err)
	}

	if jtx.TxID == "" {
		t.Error("expected txID to be set")
	}
	if jtx.RawDataHex == "" {
		t.Error("expected raw_data_hex to be set")
	}
	if len(jtx.Signature) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(jtx.Signature))
	}
	if jtx.Signature[0] != "aabbcc" {
		t.Errorf("expected signature hex 'aabbcc', got %q", jtx.Signature[0])
	}

	// Verify txID is correct
	rawBytes, _ := proto.Marshal(tx.GetRawData())
	h := sha256.Sum256(rawBytes)
	expectedID := hex.EncodeToString(h[:])
	if jtx.TxID != expectedID {
		t.Errorf("txID mismatch: %s vs %s", jtx.TxID, expectedID)
	}
}

func TestToJSON_NoSignatures(t *testing.T) {
	tx := makeTestTx()

	jsonBytes, err := ToJSON(tx)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var jtx jsonTransaction
	if err := json.Unmarshal(jsonBytes, &jtx); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if jtx.Signature != nil {
		t.Errorf("expected nil signatures, got %v", jtx.Signature)
	}
}

func TestFromJSON(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{{0xde, 0xad}, {0xbe, 0xef}}

	jsonBytes, err := ToJSON(tx)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	got, err := FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	if !proto.Equal(tx.GetRawData(), got.GetRawData()) {
		t.Error("raw_data mismatch after FromJSON")
	}
	if len(got.GetSignature()) != 2 {
		t.Fatalf("expected 2 signatures, got %d", len(got.GetSignature()))
	}
}

func TestFromJSON_InvalidJSON(t *testing.T) {
	_, err := FromJSON([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid transaction JSON") {
		t.Errorf("expected invalid transaction JSON error, got: %v", err)
	}
}

func TestFromJSON_MissingRawDataHex(t *testing.T) {
	data := `{"txID":"abc","signature":[]}`
	_, err := FromJSON([]byte(data))
	if err == nil {
		t.Fatal("expected error for missing raw_data_hex")
	}
	if !strings.Contains(err.Error(), "missing raw_data_hex") {
		t.Errorf("expected missing raw_data_hex error, got: %v", err)
	}
}

func TestFromJSON_TxIDMismatch(t *testing.T) {
	tx := makeTestTx()
	jsonBytes, err := ToJSON(tx)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	// Tamper with the txID
	var jtx map[string]any
	json.Unmarshal(jsonBytes, &jtx)
	jtx["txID"] = "0000000000000000000000000000000000000000000000000000000000000000"
	tampered, _ := json.Marshal(jtx)

	_, err = FromJSON(tampered)
	if err == nil {
		t.Fatal("expected error for txID mismatch")
	}
	if !strings.Contains(err.Error(), "txID does not match") {
		t.Errorf("expected txID mismatch error, got: %v", err)
	}
}

func TestFromJSON_NoTxID(t *testing.T) {
	// When txID is absent, validation should be skipped
	tx := makeTestTx()
	rawHex := marshalRawDataHex(t, tx)

	data, _ := json.Marshal(map[string]any{
		"raw_data_hex": rawHex,
	})

	got, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON without txID: %v", err)
	}
	if !proto.Equal(tx.GetRawData(), got.GetRawData()) {
		t.Error("raw_data mismatch")
	}
}

func TestFromJSON_InvalidSignatureHex(t *testing.T) {
	tx := makeTestTx()
	rawHex := marshalRawDataHex(t, tx)

	data, _ := json.Marshal(map[string]any{
		"raw_data_hex": rawHex,
		"signature":    []string{"not-valid-hex!"},
	})

	_, err := FromJSON(data)
	if err == nil {
		t.Fatal("expected error for invalid signature hex")
	}
	if !strings.Contains(err.Error(), "failed to decode signature") {
		t.Errorf("expected signature decode error, got: %v", err)
	}
}

func TestRoundTrip_JSON(t *testing.T) {
	tx := makeTestTx()
	tx.Signature = [][]byte{
		{0x01, 0x02, 0x03, 0x04, 0x05},
	}

	jsonBytes, err := ToJSON(tx)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	got, err := FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	if !proto.Equal(tx.GetRawData(), got.GetRawData()) {
		t.Error("raw_data mismatch in JSON round-trip")
	}

	if len(got.GetSignature()) != len(tx.GetSignature()) {
		t.Fatalf("signature count: %d vs %d", len(got.GetSignature()), len(tx.GetSignature()))
	}

	for i := range tx.GetSignature() {
		if hex.EncodeToString(tx.GetSignature()[i]) != hex.EncodeToString(got.GetSignature()[i]) {
			t.Errorf("signature[%d] mismatch", i)
		}
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
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	got, err := FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	if len(got.GetSignature()) != 3 {
		t.Fatalf("expected 3 signatures, got %d", len(got.GetSignature()))
	}
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
	if err != nil {
		t.Fatalf("FromJSON real signed tx: %v", err)
	}

	// txID validation passed (FromJSON checks it internally), but verify explicitly
	computedID, err := computeTxID(tx)
	if err != nil {
		t.Fatalf("computeTxID: %v", err)
	}
	expectedID := "c70b20da8cb8d96a99c1aceb283412acb9dea313a9e467178431a3c6ae8b9f08"
	if computedID != expectedID {
		t.Errorf("txID mismatch: got %s, want %s", computedID, expectedID)
	}

	// Verify signature was decoded
	if len(tx.GetSignature()) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(tx.GetSignature()))
	}
	sigHex := hex.EncodeToString(tx.GetSignature()[0])
	expectedSig := "1788bf0281f67c42f5a09322523edee9969e808a863776f21ab6543de9dd42ca0e4955577879df8074086c620eb96709ba9a95d60a166de190a081e8d69801da00"
	if sigHex != expectedSig {
		t.Errorf("signature mismatch:\n  got  %s\n  want %s", sigHex, expectedSig)
	}

	// Verify raw_data was deserialized correctly
	raw := tx.GetRawData()
	if raw.GetExpiration() != 1773182914544 {
		t.Errorf("expiration: got %d, want 1773182914544", raw.GetExpiration())
	}
	if raw.GetTimestamp() != 1773182854544 {
		t.Errorf("timestamp: got %d, want 1773182854544", raw.GetTimestamp())
	}
	if raw.GetFeeLimit() != 100000000 {
		t.Errorf("fee_limit: got %d, want 100000000", raw.GetFeeLimit())
	}
	if len(raw.GetContract()) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(raw.GetContract()))
	}
	if raw.GetContract()[0].GetType() != core.Transaction_Contract_TriggerSmartContract {
		t.Errorf("contract type: got %v, want TriggerSmartContract", raw.GetContract()[0].GetType())
	}

	// Verify ToRawDataHex produces the same hex as the API
	gotHex, err := ToRawDataHex(tx)
	if err != nil {
		t.Fatalf("ToRawDataHex: %v", err)
	}
	expectedHex := "0a028ad92208eaef69105495eb1540f09fc0cfcd335aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a1541495511a493d8c362be4267224e6d81013a6862ee121541eca9bc828a3005b9a3b909f2cc5c2a54794de05f2244a9059cbb00000000000000000000004150125bea243a640bf85348873b6c4f3c517aee7a00000000000000000000000000000000000000000000000000000000000000017090cbbccfcd33900180c2d72f"
	if gotHex != expectedHex {
		t.Errorf("ToRawDataHex mismatch:\n  got  %s\n  want %s", gotHex, expectedHex)
	}
}

func TestFromJSON_RealUnsignedTransaction(t *testing.T) {
	tx, err := FromJSON([]byte(realUnsignedTxJSON))
	if err != nil {
		t.Fatalf("FromJSON real unsigned tx: %v", err)
	}

	// txID validation passed internally, verify explicitly
	computedID, err := computeTxID(tx)
	if err != nil {
		t.Fatalf("computeTxID: %v", err)
	}
	expectedID := "ae80c7aa55e19c2da5e712d4be50e4c9422dd6bfc99039cc42d5deb76938c0e7"
	if computedID != expectedID {
		t.Errorf("txID mismatch: got %s, want %s", computedID, expectedID)
	}

	// No signatures on unsigned tx
	if len(tx.GetSignature()) != 0 {
		t.Errorf("expected 0 signatures, got %d", len(tx.GetSignature()))
	}

	// Verify raw_data fields
	raw := tx.GetRawData()
	if raw.GetExpiration() != 1773182925000 {
		t.Errorf("expiration: got %d, want 1773182925000", raw.GetExpiration())
	}
	if len(raw.GetContract()) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(raw.GetContract()))
	}
	if raw.GetContract()[0].GetType() != core.Transaction_Contract_TransferContract {
		t.Errorf("contract type: got %v, want TransferContract", raw.GetContract()[0].GetType())
	}
}

func TestFromRawDataHex_RealTransaction(t *testing.T) {
	rawHex := "0a028acb22087c571ea3e7e9dbf340c8f1c0cfcd335a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541eca9bc828a3005b9a3b909f2cc5c2a54794de05f121541495511a493d8c362be4267224e6d81013a6862ee18e807708da3bdcfcd33"

	tx, err := FromRawDataHex(rawHex)
	if err != nil {
		t.Fatalf("FromRawDataHex real tx: %v", err)
	}

	// Verify txID matches what the API returned
	computedID, err := computeTxID(tx)
	if err != nil {
		t.Fatalf("computeTxID: %v", err)
	}
	expectedID := "ae80c7aa55e19c2da5e712d4be50e4c9422dd6bfc99039cc42d5deb76938c0e7"
	if computedID != expectedID {
		t.Errorf("txID mismatch: got %s, want %s", computedID, expectedID)
	}

	// Re-encoding should produce identical hex
	gotHex, err := ToRawDataHex(tx)
	if err != nil {
		t.Fatalf("ToRawDataHex: %v", err)
	}
	if gotHex != rawHex {
		t.Errorf("re-encoded hex mismatch:\n  got  %s\n  want %s", gotHex, rawHex)
	}
}

func TestComputeTxID(t *testing.T) {
	tx := makeTestTx()

	id, err := computeTxID(tx)
	if err != nil {
		t.Fatalf("computeTxID: %v", err)
	}

	// Verify manually
	rawBytes, _ := proto.Marshal(tx.GetRawData())
	h := sha256.Sum256(rawBytes)
	expected := hex.EncodeToString(h[:])

	if id != expected {
		t.Errorf("txID: got %s, want %s", id, expected)
	}
}
