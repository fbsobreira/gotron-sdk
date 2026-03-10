package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	ErrEmptyRawData  = errors.New("raw data hex string is empty")
	ErrInvalidHex    = errors.New("invalid hex string")
	ErrInvalidTxJSON = errors.New("invalid transaction JSON")
	ErrTxIDMismatch  = errors.New("txID does not match hash of raw_data")
)

// jsonTransaction represents the JSON format returned by TRON HTTP APIs.
type jsonTransaction struct {
	Visible    bool            `json:"visible"`
	TxID       string          `json:"txID"`
	RawData    json.RawMessage `json:"raw_data"`
	RawDataHex string          `json:"raw_data_hex"`
	Signature  []string        `json:"signature"`
}

// FromRawDataHex reconstructs a Transaction from a hex-encoded raw_data protobuf.
// This is the format commonly returned by TRON HTTP APIs (raw_data_hex field).
// Optional signatures can be attached to the resulting transaction.
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

// FromJSON reconstructs a Transaction from a JSON representation
// as returned by TRON HTTP API endpoints (e.g., /wallet/createtransaction).
// It extracts raw_data_hex for reliable protobuf deserialization.
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

// ToRawDataHex returns the hex-encoded protobuf of transaction raw data.
func ToRawDataHex(tx *core.Transaction) (string, error) {
	rawBytes, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw_data: %w", err)
	}
	return hex.EncodeToString(rawBytes), nil
}

// ToJSON serializes a Transaction to JSON format compatible with TRON HTTP APIs.
func ToJSON(tx *core.Transaction) ([]byte, error) {
	rawBytes, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw_data: %w", err)
	}

	rawDataHex := hex.EncodeToString(rawBytes)

	h := sha256.Sum256(rawBytes)
	txID := hex.EncodeToString(h[:])

	// Use protojson for TRON-compatible JSON representation of raw_data
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

// computeTxID returns the hex-encoded SHA256 hash of the transaction's raw_data.
func computeTxID(tx *core.Transaction) (string, error) {
	rawBytes, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(rawBytes)
	return hex.EncodeToString(h[:]), nil
}
