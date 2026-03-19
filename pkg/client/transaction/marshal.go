package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/encoding/protojson"
	proto "google.golang.org/protobuf/proto"
)

var (
	// ErrEmptyRawData is returned when a raw data hex string is empty.
	ErrEmptyRawData = errors.New("raw data hex string is empty")
	// ErrInvalidHex is returned when a hex string cannot be decoded.
	ErrInvalidHex = errors.New("invalid hex string")
	// ErrInvalidTxJSON is returned when transaction JSON cannot be parsed.
	ErrInvalidTxJSON = errors.New("invalid transaction JSON")
	// ErrTxIDMismatch is returned when the txID does not match the hash of raw_data.
	ErrTxIDMismatch = errors.New("txID does not match hash of raw_data")
	// ErrNilRawData is returned when a transaction has nil raw_data.
	ErrNilRawData = errors.New("transaction raw_data is nil")
)

// jsonTransaction represents the JSON envelope used by TRON HTTP APIs.
//
// The canonical serialization field is raw_data_hex, which contains the
// protobuf-encoded TransactionRaw as a hex string. The raw_data object
// is included for readability but uses protojson encoding (base64 bytes,
// @type for Any fields), which differs from the TRON HTTP API format
// (hex bytes, type_url/value for Any). Consumers should always prefer
// raw_data_hex for deserialization.
type jsonTransaction struct {
	Visible    bool            `json:"visible"`
	TxID       string          `json:"txID"`
	RawData    json.RawMessage `json:"raw_data,omitempty"`
	RawDataHex string          `json:"raw_data_hex"`
	Signature  []string        `json:"signature,omitempty"`
}

// FromRawDataHex reconstructs a Transaction from a hex-encoded raw_data
// protobuf. This is the format commonly returned by TRON HTTP APIs in the
// raw_data_hex field.
//
// The hex string may optionally have a "0x" prefix, which is stripped
// before decoding. Any provided signatures are attached to the resulting
// transaction, preserving their order.
//
// Example:
//
//	tx, err := transaction.FromRawDataHex(apiResponse.RawDataHex)
//	if err != nil { ... }
//	tx, err = transaction.SignTransaction(tx, privateKey)
func FromRawDataHex(rawDataHex string, signatures ...[]byte) (*core.Transaction, error) {
	rawDataHex = strings.TrimPrefix(rawDataHex, "0x")
	if rawDataHex == "" {
		return nil, ErrEmptyRawData
	}

	rawBytes, err := hex.DecodeString(rawDataHex)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidHex, err)
	}

	rawData := &core.TransactionRaw{}
	if err := proto.Unmarshal(rawBytes, rawData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal raw_data: %w", err)
	}

	tx := &core.Transaction{
		RawData:   rawData,
		Signature: signatures,
	}

	return tx, nil
}

// FromJSON reconstructs a Transaction from a JSON representation as
// returned by TRON HTTP API endpoints (e.g., /wallet/createtransaction).
//
// Deserialization uses the raw_data_hex field exclusively for reliable
// protobuf reconstruction. The raw_data JSON object is ignored because
// it has encoding differences (hex vs base64 bytes) that make direct
// JSON-to-proto mapping unreliable.
//
// If a txID is present in the JSON, it is validated against the SHA256
// hash of the deserialized raw_data. Signatures are decoded from hex
// strings and attached to the transaction.
//
// Example:
//
//	resp, _ := http.Post("https://api.trongrid.io/wallet/createtransaction", ...)
//	body, _ := io.ReadAll(resp.Body)
//	tx, err := transaction.FromJSON(body)
func FromJSON(jsonData []byte) (*core.Transaction, error) {
	var jtx jsonTransaction
	if err := json.Unmarshal(jsonData, &jtx); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidTxJSON, err)
	}

	if jtx.RawDataHex == "" {
		return nil, fmt.Errorf("%w: missing raw_data_hex field", ErrInvalidTxJSON)
	}

	// Decode signatures from hex strings
	var sigs [][]byte
	for i, sigHex := range jtx.Signature {
		sig, err := hex.DecodeString(sigHex)
		if err != nil {
			return nil, fmt.Errorf("failed to decode signature[%d]: %w", i, err)
		}
		sigs = append(sigs, sig)
	}

	tx, err := FromRawDataHex(jtx.RawDataHex, sigs...)
	if err != nil {
		return nil, err
	}

	// Validate txID if present
	if jtx.TxID != "" {
		computedID, err := computeTxID(tx)
		if err != nil {
			return nil, fmt.Errorf("failed to compute txID: %w", err)
		}
		if !strings.EqualFold(jtx.TxID, computedID) {
			return nil, fmt.Errorf("%w: expected %s, got %s", ErrTxIDMismatch, jtx.TxID, computedID)
		}
	}

	return tx, nil
}

// ToRawDataHex returns the hex-encoded protobuf serialization of the
// transaction's raw_data. The output is lowercase hex without a "0x"
// prefix, matching the format used by TRON HTTP APIs.
func ToRawDataHex(tx *core.Transaction) (string, error) {
	if tx == nil || tx.GetRawData() == nil {
		return "", ErrNilRawData
	}
	rawBytes, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw_data: %w", err)
	}
	return hex.EncodeToString(rawBytes), nil
}

// ToJSON serializes a Transaction to a JSON envelope containing txID,
// raw_data_hex, raw_data, and signature fields.
//
// The txID is computed as the SHA256 hash of the protobuf-encoded raw_data.
// The raw_data_hex field contains the canonical protobuf serialization and
// is suitable for use with FromJSON or FromRawDataHex.
// Signatures are encoded as lowercase hex strings.
//
// Note: the raw_data JSON object uses protojson encoding, which differs
// from the TRON HTTP API format (protojson uses base64 for bytes and @type
// for Any fields, while TRON uses hex strings and type_url/value). The
// raw_data field is included for human readability; use raw_data_hex for
// programmatic deserialization.
func ToJSON(tx *core.Transaction) ([]byte, error) {
	if tx == nil || tx.GetRawData() == nil {
		return nil, ErrNilRawData
	}

	rawBytes, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw_data: %w", err)
	}

	rawDataHex := hex.EncodeToString(rawBytes)

	h := sha256.Sum256(rawBytes)
	txID := hex.EncodeToString(h[:])

	// protojson for human-readable raw_data (note: not TRON-identical, see doc)
	rawDataJSON, err := protojson.Marshal(tx.GetRawData())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw_data to JSON: %w", err)
	}

	var sigs []string
	for _, sig := range tx.GetSignature() {
		sigs = append(sigs, hex.EncodeToString(sig))
	}

	jtx := jsonTransaction{
		TxID:       txID,
		RawData:    rawDataJSON,
		RawDataHex: rawDataHex,
		Signature:  sigs,
	}

	return json.Marshal(jtx)
}

// computeTxID returns the hex-encoded SHA256 hash of the transaction's
// protobuf-encoded raw_data. This matches the txID computation used by
// the TRON network.
func computeTxID(tx *core.Transaction) (string, error) {
	if tx == nil || tx.GetRawData() == nil {
		return "", ErrNilRawData
	}
	rawBytes, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(rawBytes)
	return hex.EncodeToString(h[:]), nil
}
