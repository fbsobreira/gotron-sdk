package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
//
// TIP-899 post-quantum signatures travel in pq_auth_sig, which is
// hand-encoded like signature (enum by name, bytes as lowercase hex) rather
// than through protojson, matching the java-tron HTTP format.
type jsonTransaction struct {
	Visible    bool            `json:"visible"`
	TxID       string          `json:"txID"`
	RawData    json.RawMessage `json:"raw_data,omitempty"`
	RawDataHex string          `json:"raw_data_hex"`
	Signature  []string        `json:"signature,omitempty"`
	PQAuthSig  []jsonPQAuthSig `json:"pq_auth_sig,omitempty"`
}

// jsonPQAuthSig is the JSON form of core.PQAuthSig. Scheme stays raw so that
// FromJSON can accept both the enum name and an integer.
type jsonPQAuthSig struct {
	Scheme    json.RawMessage `json:"scheme"`
	PublicKey string          `json:"public_key"`
	Signature string          `json:"signature"`
}

// encodePQAuthSig converts one entry for ToJSON. It refuses what FromJSON
// would reject (a nil entry, an unknown or zero scheme, an empty key or
// signature), so ToJSON output always round-trips.
func encodePQAuthSig(pq *core.PQAuthSig, index int) (jsonPQAuthSig, error) {
	if pq == nil {
		return jsonPQAuthSig{}, fmt.Errorf("pq_auth_sig[%d] is nil", index)
	}
	name, ok := core.PQScheme_name[int32(pq.GetScheme())]
	if !ok || pq.GetScheme() == core.PQScheme_UNKNOWN_PQ_SCHEME {
		return jsonPQAuthSig{}, fmt.Errorf("pq_auth_sig[%d]: unknown scheme %d", index, int32(pq.GetScheme()))
	}
	if len(pq.GetPublicKey()) == 0 {
		return jsonPQAuthSig{}, fmt.Errorf("pq_auth_sig[%d]: empty public_key", index)
	}
	if len(pq.GetSignature()) == 0 {
		return jsonPQAuthSig{}, fmt.Errorf("pq_auth_sig[%d]: empty signature", index)
	}
	return jsonPQAuthSig{
		Scheme:    json.RawMessage(strconv.Quote(name)),
		PublicKey: hex.EncodeToString(pq.GetPublicKey()),
		Signature: hex.EncodeToString(pq.GetSignature()),
	}, nil
}

// parsePQScheme accepts an exact enum name or a JSON integer, as java-tron
// does. Unknown values and UNKNOWN_PQ_SCHEME (0) are rejected. Rejected input
// is never echoed.
func parsePQScheme(raw json.RawMessage, index int) (core.PQScheme, error) {
	errUnknown := fmt.Errorf("%w: pq_auth_sig[%d]: unknown scheme", ErrInvalidTxJSON, index)

	var v int32
	if len(raw) > 0 && raw[0] == '"' {
		var name string
		if err := json.Unmarshal(raw, &name); err != nil {
			return 0, errUnknown
		}
		n, ok := core.PQScheme_value[name]
		if !ok {
			return 0, errUnknown
		}
		v = n
	} else {
		n, err := strconv.ParseInt(string(raw), 10, 32)
		if err != nil {
			return 0, errUnknown
		}
		v = int32(n)
	}

	if _, ok := core.PQScheme_name[v]; !ok || core.PQScheme(v) == core.PQScheme_UNKNOWN_PQ_SCHEME {
		return 0, errUnknown
	}
	return core.PQScheme(v), nil
}

// FromRawDataHex reconstructs a Transaction from a hex-encoded raw_data
// protobuf. This is the format commonly returned by TRON HTTP APIs in the
// raw_data_hex field.
//
// The hex string may optionally have a "0x" prefix, which is stripped
// before decoding. Any provided signatures are attached to the resulting
// transaction, preserving their order.
//
// The signatures are ECDSA signatures only. raw_data_hex does not cover
// TIP-899 pq_auth_sig entries; use FromJSON to carry those.
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
// TIP-899 post-quantum signatures are read from the pq_auth_sig array. Each
// entry carries a scheme (the enum name such as "FN_DSA_512" or "ML_DSA_44",
// or its integer value), and hex-encoded public_key and signature (an
// optional "0x" prefix is accepted). Unknown schemes, UNKNOWN_PQ_SCHEME, bad
// hex, and empty keys or signatures are rejected. Key and signature lengths
// are not checked here; the node is the authority. pq_auth_sig is not part of
// raw_data, so it does not affect txID validation.
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

	pqSigs, err := decodePQAuthSigs(jtx.PQAuthSig)
	if err != nil {
		return nil, err
	}

	tx, err := FromRawDataHex(jtx.RawDataHex, sigs...)
	if err != nil {
		return nil, err
	}
	tx.PqAuthSig = pqSigs

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

// decodePQAuthSigs converts the JSON pq_auth_sig entries to proto messages.
// Errors name the entry index and field but never include the offending value.
func decodePQAuthSigs(entries []jsonPQAuthSig) ([]*core.PQAuthSig, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	out := make([]*core.PQAuthSig, 0, len(entries))
	for i, e := range entries {
		scheme, err := parsePQScheme(e.Scheme, i)
		if err != nil {
			return nil, err
		}
		pk, err := decodePQHex(e.PublicKey, i, "public_key")
		if err != nil {
			return nil, err
		}
		sig, err := decodePQHex(e.Signature, i, "signature")
		if err != nil {
			return nil, err
		}
		out = append(out, &core.PQAuthSig{
			Scheme:    scheme,
			PublicKey: pk,
			Signature: sig,
		})
	}
	return out, nil
}

// decodePQHex decodes one non-empty hex field of a pq_auth_sig entry,
// accepting an optional "0x" or "0X" prefix.
func decodePQHex(s string, index int, field string) ([]byte, error) {
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	if s == "" {
		return nil, fmt.Errorf("%w: pq_auth_sig[%d]: empty %s", ErrInvalidTxJSON, index, field)
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		// The decoder error quotes the offending byte; report only the location.
		return nil, fmt.Errorf("%w: pq_auth_sig[%d]: invalid %s hex", ErrInvalidTxJSON, index, field)
	}
	return b, nil
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
// raw_data_hex, raw_data, and signature fields. Transactions carrying TIP-899
// post-quantum signatures also get a pq_auth_sig array of
// {"scheme", "public_key", "signature"} objects, with the scheme printed by
// enum name and the bytes as lowercase hex; the field is omitted when there
// are none. An entry FromJSON would reject (nil, an unknown or zero scheme,
// an empty public key or signature) is an error, so the output round-trips.
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

	var pqSigs []jsonPQAuthSig
	for i, pq := range tx.GetPqAuthSig() {
		e, err := encodePQAuthSig(pq, i)
		if err != nil {
			return nil, err
		}
		pqSigs = append(pqSigs, e)
	}

	jtx := jsonTransaction{
		TxID:       txID,
		RawData:    rawDataJSON,
		RawDataHex: rawDataHex,
		Signature:  sigs,
		PQAuthSig:  pqSigs,
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
