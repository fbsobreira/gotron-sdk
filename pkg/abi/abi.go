// Package abi provides ABI encoding and decoding for TRON smart contracts.
package abi

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"golang.org/x/crypto/sha3"
)

// Param list
type Param map[string]interface{}

// LoadFromJSON string into ABI data
func LoadFromJSON(jString string) ([]Param, error) {
	if len(jString) == 0 {
		return nil, nil
	}
	data := []Param{}
	err := json.Unmarshal([]byte(jString), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Signature of a method
func Signature(method string) []byte {
	// hash method
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write([]byte(method))
	b := hasher.Sum(nil)
	return b[:4]
}

func convetToAddress(v interface{}) (eCommon.Address, error) {
	switch v := v.(type) {
	case string:
		addr, err := address.Base58ToAddress(v)
		if err != nil {
			return eCommon.Address{}, fmt.Errorf("invalid address %s: %+v", v, err)
		}
		return eCommon.BytesToAddress(addr.Bytes()[len(addr.Bytes())-20:]), nil
	}
	return eCommon.Address{}, fmt.Errorf("invalid address %v", v)
}

func convertToInt(ty eABI.Type, v interface{}) (interface{}, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", v)
	}
	if ty.T == eABI.IntTy && (ty.Size == 8 || ty.Size == 16 || ty.Size == 32 || ty.Size == 64) {
		tmp, err := strconv.ParseInt(s, 10, ty.Size)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %q as int%d: %w", s, ty.Size, err)
		}
		switch ty.Size {
		case 8:
			v = int8(tmp)
		case 16:
			v = int16(tmp)
		case 32:
			v = int32(tmp)
		case 64:
			v = int64(tmp)
		}
	} else if ty.T == eABI.UintTy && (ty.Size == 8 || ty.Size == 16 || ty.Size == 32 || ty.Size == 64) {
		tmp, err := strconv.ParseUint(s, 10, ty.Size)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %q as uint%d: %w", s, ty.Size, err)
		}
		switch ty.Size {
		case 8:
			v = uint8(tmp)
		case 16:
			v = uint16(tmp)
		case 32:
			v = uint32(tmp)
		case 64:
			v = uint64(tmp)
		}
	} else {
		// check for hex char
		var ok bool
		if strings.HasPrefix(s, "0x") {
			v, ok = new(big.Int).SetString(s[2:], 16)
		} else {
			v, ok = new(big.Int).SetString(s, 10)
		}
		if !ok {
			return nil, fmt.Errorf("cannot parse %q as big.Int", s)
		}
	}
	return v, nil
}

// GetPaddedParam from struct
func GetPaddedParam(param []Param) ([]byte, error) {
	values := make([]interface{}, 0)
	arguments := eABI.Arguments{}

	for _, p := range param {
		if len(p) != 1 {
			return nil, fmt.Errorf("invalid param %+v", p)
		}
		for k, v := range p {
			ty, err := eABI.NewType(k, "", nil)
			if err != nil {
				return nil, fmt.Errorf("invalid param %+v: %+v", p, err)
			}
			arguments = append(arguments,
				eABI.Argument{
					Name:    "",
					Type:    ty,
					Indexed: false,
				},
			)

			if ty.T == eABI.SliceTy || ty.T == eABI.ArrayTy {
				if ty.Elem.T == eABI.AddressTy {
					tmp, ok := v.([]interface{})
					if !ok {
						return nil, fmt.Errorf("unable to convert array of addresses %+v", p)
					}
					v = make([]eCommon.Address, 0)
					for i := range tmp {
						addr, err := convetToAddress(tmp[i])
						if err != nil {
							return nil, err
						}
						v = append(v.([]eCommon.Address), addr)
					}
				}

				if ty.Elem.T == eABI.IntTy || ty.Elem.T == eABI.UintTy {
					strs, err := toStringSlice(v)
					if err != nil {
						return nil, fmt.Errorf("unable to convert array of ints %+v", p)
					}
					if ty.Elem.Size > 64 {
						tmp := make([]*big.Int, len(strs))
						for i, s := range strs {
							var ok bool
							if strings.HasPrefix(s, "0x") {
								tmp[i], ok = new(big.Int).SetString(s[2:], 16)
							} else {
								tmp[i], ok = new(big.Int).SetString(s, 10)
							}
							if !ok {
								return nil, fmt.Errorf("element %d: cannot parse %q as big.Int", i, s)
							}
						}
						v = tmp
					} else {
						v, err = convertSmallIntSlice(*ty.Elem, strs)
						if err != nil {
							return nil, err
						}
					}
				}

				if ty.Elem.T == eABI.BytesTy {
					tmp, ok := v.([]interface{})
					if !ok {
						return nil, fmt.Errorf("unable to convert array of bytes %+v", p)
					}
					result := make([][]byte, len(tmp))
					for i := range tmp {
						b, err := convertToBytes(*ty.Elem, tmp[i])
						if err != nil {
							return nil, err
						}
						result[i] = b.([]byte)
					}
					v = result
				}
			}
			if ty.T == eABI.AddressTy {
				if v, err = convetToAddress(v); err != nil {
					return nil, err
				}
			}
			if ty.T == eABI.IntTy || ty.T == eABI.UintTy {
				if _, ok := v.(string); ok {
					v, err = convertToInt(ty, v)
					if err != nil {
						return nil, err
					}
				}
			}

			if ty.T == eABI.BytesTy || ty.T == eABI.FixedBytesTy {
				var err error
				if v, err = convertToBytes(ty, v); err != nil {
					return nil, err
				}
			}

			values = append(values, v)
		}
	}
	// convert params to bytes
	return arguments.PackValues(values)
}

// toStringSlice converts []string or []interface{} to []string.
// JSON unmarshaling produces []interface{} so both forms must be accepted.
func toStringSlice(v interface{}) ([]string, error) {
	switch s := v.(type) {
	case []string:
		return s, nil
	case []interface{}:
		out := make([]string, len(s))
		for i, elem := range s {
			str, ok := elem.(string)
			if !ok {
				return nil, fmt.Errorf("element %d is not a string: %v", i, elem)
			}
			out[i] = str
		}
		return out, nil
	}
	return nil, fmt.Errorf("expected string slice, got %T", v)
}

// convertSmallIntSlice handles int/uint arrays with size <= 64.
func convertSmallIntSlice(elemTy eABI.Type, strs []string) (interface{}, error) {
	switch {
	case elemTy.T == eABI.UintTy && elemTy.Size == 8:
		out := make([]uint8, len(strs))
		for i, s := range strs {
			tmp, err := strconv.ParseUint(s, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("element %d: cannot parse %q as uint8: %w", i, s, err)
			}
			out[i] = uint8(tmp)
		}
		return out, nil
	case elemTy.T == eABI.UintTy && elemTy.Size == 16:
		out := make([]uint16, len(strs))
		for i, s := range strs {
			tmp, err := strconv.ParseUint(s, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("element %d: cannot parse %q as uint16: %w", i, s, err)
			}
			out[i] = uint16(tmp)
		}
		return out, nil
	case elemTy.T == eABI.UintTy && elemTy.Size == 32:
		out := make([]uint32, len(strs))
		for i, s := range strs {
			tmp, err := strconv.ParseUint(s, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("element %d: cannot parse %q as uint32: %w", i, s, err)
			}
			out[i] = uint32(tmp)
		}
		return out, nil
	case elemTy.T == eABI.UintTy && elemTy.Size == 64:
		out := make([]uint64, len(strs))
		for i, s := range strs {
			tmp, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("element %d: cannot parse %q as uint64: %w", i, s, err)
			}
			out[i] = tmp
		}
		return out, nil
	case elemTy.T == eABI.IntTy && elemTy.Size == 8:
		out := make([]int8, len(strs))
		for i, s := range strs {
			tmp, err := strconv.ParseInt(s, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("element %d: cannot parse %q as int8: %w", i, s, err)
			}
			out[i] = int8(tmp)
		}
		return out, nil
	case elemTy.T == eABI.IntTy && elemTy.Size == 16:
		out := make([]int16, len(strs))
		for i, s := range strs {
			tmp, err := strconv.ParseInt(s, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("element %d: cannot parse %q as int16: %w", i, s, err)
			}
			out[i] = int16(tmp)
		}
		return out, nil
	case elemTy.T == eABI.IntTy && elemTy.Size == 32:
		out := make([]int32, len(strs))
		for i, s := range strs {
			tmp, err := strconv.ParseInt(s, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("element %d: cannot parse %q as int32: %w", i, s, err)
			}
			out[i] = int32(tmp)
		}
		return out, nil
	case elemTy.T == eABI.IntTy && elemTy.Size == 64:
		out := make([]int64, len(strs))
		for i, s := range strs {
			tmp, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("element %d: cannot parse %q as int64: %w", i, s, err)
			}
			out[i] = tmp
		}
		return out, nil
	default:
		out := make([]*big.Int, len(strs))
		for i, s := range strs {
			val, ok := new(big.Int).SetString(s, 10)
			if !ok {
				return nil, fmt.Errorf("element %d: cannot parse %q as big.Int", i, s)
			}
			out[i] = val
		}
		return out, nil
	}
}

func convertToBytes(ty eABI.Type, v interface{}) (interface{}, error) {
	// if string
	if data, ok := v.(string); ok {
		// convert from hex string
		dataBytes, err := hex.DecodeString(data)
		if err != nil {
			// try with base64
			dataBytes, err = base64.StdEncoding.DecodeString(data)
			if err != nil {
				return nil, err
			}
		}
		// if array and size == 0
		if ty.T == eABI.BytesTy || ty.Size == 0 {
			return dataBytes, nil
		}
		if len(dataBytes) != ty.Size {
			return nil, fmt.Errorf("invalid size: %d/%d", ty.Size, len(dataBytes))
		}
		switch ty.Size {
		case 1:
			value := [1]byte{}
			copy(value[:], dataBytes[:1])
			return value, nil
		case 2:
			value := [2]byte{}
			copy(value[:], dataBytes[:2])
			return value, nil
		case 8:
			value := [8]byte{}
			copy(value[:], dataBytes[:8])
			return value, nil
		case 16:
			value := [16]byte{}
			copy(value[:], dataBytes[:16])
			return value, nil
		case 32:
			value := [32]byte{}
			copy(value[:], dataBytes[:32])
			return value, nil
		}
	}
	return v, nil
}

// Pack data into bytes
func Pack(method string, param []Param) ([]byte, error) {
	signature := Signature(method)
	pBytes, err := GetPaddedParam(param)
	if err != nil {
		return nil, err
	}
	signature = append(signature, pBytes...)
	return signature, nil
}

// entrySignature builds the canonical signature for an ABI entry,
// e.g. "rollDice(uint256,uint256,address)".
func entrySignature(entry *core.SmartContract_ABI_Entry) string {
	types := make([]string, len(entry.Inputs))
	for i, input := range entry.Inputs {
		types[i] = input.Type
	}
	return fmt.Sprintf("%s(%s)", entry.Name, strings.Join(types, ","))
}

// matchEntry checks whether an ABI entry matches the given method string.
// The method can be either a plain name (e.g. "transfer") or a full
// signature (e.g. "transfer(address,uint256)"). When a plain name is used
// and multiple entries share that name, the first match is returned — callers
// should use the full signature form for overloaded methods.
func matchEntry(entry *core.SmartContract_ABI_Entry, method string) bool {
	if strings.Contains(method, "(") {
		return entrySignature(entry) == method
	}
	return entry.Name == method
}

// GetParser return output method parser arguments from ABI
func GetParser(ABI *core.SmartContract_ABI, method string) (eABI.Arguments, error) {
	arguments := eABI.Arguments{}
	for _, entry := range ABI.Entrys {
		if entry.Type != core.SmartContract_ABI_Entry_Function {
			continue
		}
		if matchEntry(entry, method) {
			for _, out := range entry.Outputs {
				ty, err := eABI.NewType(out.Type, "", nil)
				if err != nil {
					return nil, fmt.Errorf("invalid param %s: %+v", out.Type, err)
				}
				arguments = append(arguments, eABI.Argument{
					Name:    out.Name,
					Type:    ty,
					Indexed: out.Indexed,
				})
			}
			return arguments, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

// GetEventParser returns indexed and non-indexed argument lists for an ABI event.
func GetEventParser(ABI *core.SmartContract_ABI, event string) (indexed eABI.Arguments, nonIndexed eABI.Arguments, err error) {
	for _, entry := range ABI.Entrys {
		if entry.Type != core.SmartContract_ABI_Entry_Event {
			continue
		}
		if matchEntry(entry, event) {
			for _, param := range entry.Inputs {
				ty, err := eABI.NewType(param.Type, "", nil)
				if err != nil {
					return nil, nil, fmt.Errorf("invalid param %s: %+v", param.Type, err)
				}
				arg := eABI.Argument{
					Name:    param.Name,
					Type:    ty,
					Indexed: param.Indexed,
				}
				if param.Indexed {
					indexed = append(indexed, arg)
				} else {
					nonIndexed = append(nonIndexed, arg)
				}
			}
			return indexed, nonIndexed, nil
		}
	}
	return nil, nil, fmt.Errorf("event %s not found", event)
}

// ParseTopicsIntoMap parses event log topics into a map with automatic
// Ethereum-to-TRON address conversion. The topics slice should not include
// the event signature hash (topics[0]); pass only the indexed parameter topics.
func ParseTopicsIntoMap(out map[string]interface{}, fields eABI.Arguments, topics [][]byte) error {
	if out == nil {
		return fmt.Errorf("out is nil")
	}

	ethTopics := make([]eCommon.Hash, len(topics))
	for i, v := range topics {
		ethTopics[i] = eCommon.BytesToHash(v)
	}

	if err := eABI.ParseTopicsIntoMap(out, fields, ethTopics); err != nil {
		return err
	}

	// Convert any Ethereum addresses to TRON addresses
	for k, v := range out {
		if addr, ok := v.(eCommon.Address); ok {
			out[k] = ethToTronAddress(addr)
		}
	}

	return nil
}

// revertSelector is the 4-byte function selector for Error(string),
// used by Solidity's revert/require statements.
var revertSelector = [4]byte{0x08, 0xc3, 0x79, 0xa0}

// panicSelector is the 4-byte function selector for Panic(uint256),
// emitted by Solidity for assertion failures, arithmetic overflow, etc.
var panicSelector = [4]byte{0x4e, 0x48, 0x7b, 0x71}

// panicReasons maps Solidity panic codes to human-readable descriptions.
var panicReasons = map[uint8]string{
	0x00: "generic compiler panic",
	0x01: "assertion failure",
	0x11: "arithmetic overflow/underflow",
	0x12: "division or modulo by zero",
	0x21: "invalid enum conversion",
	0x22: "invalid storage byte array encoding",
	0x31: "pop on empty array",
	0x32: "array index out of bounds",
	0x41: "out of memory",
	0x51: "zero-initialized function pointer call",
}

// ethToTronAddress converts an Ethereum common.Address to a TRON address
// by prepending the 0x41 prefix byte.
func ethToTronAddress(addr eCommon.Address) address.Address {
	tronAddr := make([]byte, 1+len(addr.Bytes()))
	tronAddr[0] = address.TronBytePrefix
	copy(tronAddr[1:], addr.Bytes())
	return address.Address(tronAddr)
}

// DecodeOutput decodes ABI-encoded output bytes from a constant contract call
// into a slice of typed values. Addresses are automatically converted from
// Ethereum format to TRON base58 format.
func DecodeOutput(contractABI *core.SmartContract_ABI, method string, data []byte) ([]interface{}, error) {
	args, err := GetParser(contractABI, method)
	if err != nil {
		return nil, fmt.Errorf("get output parser: %w", err)
	}

	// Functions with no declared outputs produce empty data — return early.
	if len(args) == 0 {
		return nil, nil
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("empty output data")
	}

	values, err := args.UnpackValues(data)
	if err != nil {
		return nil, fmt.Errorf("unpack output: %w", err)
	}

	// Convert Ethereum addresses to TRON addresses in-place.
	for i, v := range values {
		values[i] = convertOutputValue(v)
	}

	return values, nil
}

// convertOutputValue converts Ethereum addresses to TRON addresses in decoded
// ABI output values. Handles single addresses, address slices, and fixed-size
// address arrays (e.g. [3]common.Address). For unrecognized types, the value
// is returned unchanged.
func convertOutputValue(v interface{}) interface{} {
	switch val := v.(type) {
	case eCommon.Address:
		return ethToTronAddress(val)
	case []eCommon.Address:
		result := make([]address.Address, len(val))
		for i, addr := range val {
			result[i] = ethToTronAddress(addr)
		}
		return result
	default:
		// Handle fixed-size address arrays ([N]common.Address) which
		// go-ethereum returns for Solidity fixed-size array outputs.
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Array && rv.Type().Elem() == reflect.TypeOf(eCommon.Address{}) {
			result := make([]address.Address, rv.Len())
			for i := range result {
				result[i] = ethToTronAddress(rv.Index(i).Interface().(eCommon.Address))
			}
			return result
		}
		return v
	}
}

// DecodeRevertReason extracts the human-readable error string from
// ABI-encoded revert data. Supports both Error(string) (selector 0x08c379a0)
// from revert/require and Panic(uint256) (selector 0x4e487b71) from
// assertion failures and arithmetic errors.
func DecodeRevertReason(data []byte) (string, error) {
	if len(data) < 4 {
		return "", fmt.Errorf("data too short for revert selector: %d bytes", len(data))
	}

	selector := [4]byte(data[:4])

	switch selector {
	case revertSelector:
		return decodeErrorString(data[4:])
	case panicSelector:
		return decodePanicReason(data[4:])
	default:
		return "", fmt.Errorf("unknown error selector: 0x%x", data[:4])
	}
}

// decodeErrorString decodes the ABI-encoded string from Error(string) revert data.
func decodeErrorString(data []byte) (string, error) {
	strTy, err := eABI.NewType("string", "", nil)
	if err != nil {
		return "", fmt.Errorf("create string type: %w", err)
	}

	args := eABI.Arguments{{Type: strTy}}
	values, err := args.UnpackValues(data)
	if err != nil {
		return "", fmt.Errorf("unpack revert reason: %w", err)
	}

	reason, ok := values[0].(string)
	if !ok {
		return "", fmt.Errorf("unexpected revert reason type: %T", values[0])
	}

	return reason, nil
}

// decodePanicReason decodes the ABI-encoded uint256 from Panic(uint256) data
// and returns a human-readable description of the panic code.
func decodePanicReason(data []byte) (string, error) {
	uintTy, err := eABI.NewType("uint256", "", nil)
	if err != nil {
		return "", fmt.Errorf("create uint256 type: %w", err)
	}

	args := eABI.Arguments{{Type: uintTy}}
	values, err := args.UnpackValues(data)
	if err != nil {
		return "", fmt.Errorf("unpack panic code: %w", err)
	}

	code, ok := values[0].(*big.Int)
	if !ok {
		return "", fmt.Errorf("unexpected panic code type: %T", values[0])
	}

	if code.IsUint64() && code.Uint64() <= 0xFF {
		if desc, found := panicReasons[uint8(code.Uint64())]; found {
			return fmt.Sprintf("panic: %s (0x%02x)", desc, code.Uint64()), nil
		}
	}

	return fmt.Sprintf("panic: unknown code (0x%x)", code), nil
}

// entryTypeName maps proto EntryType enum values to lowercase Ethereum ABI
// JSON labels. Unknown or out-of-range values produce an empty string.
var entryTypeName = map[core.SmartContract_ABI_Entry_EntryType]string{
	core.SmartContract_ABI_Entry_Constructor: "constructor",
	core.SmartContract_ABI_Entry_Function:    "function",
	core.SmartContract_ABI_Entry_Event:       "event",
	core.SmartContract_ABI_Entry_Fallback:    "fallback",
	core.SmartContract_ABI_Entry_Receive:     "receive",
	core.SmartContract_ABI_Entry_Error:       "error",
}

// stateMutabilityName maps proto StateMutabilityType enum values to lowercase
// Ethereum ABI JSON labels.
var stateMutabilityName = map[core.SmartContract_ABI_Entry_StateMutabilityType]string{
	core.SmartContract_ABI_Entry_Pure:       "pure",
	core.SmartContract_ABI_Entry_View:       "view",
	core.SmartContract_ABI_Entry_Nonpayable: "nonpayable",
	core.SmartContract_ABI_Entry_Payable:    "payable",
}

// entryHasName reports whether the entry type carries a name field in
// canonical Ethereum ABI JSON. Fallback and receive entries are unnamed.
func entryHasName(t core.SmartContract_ABI_Entry_EntryType) bool {
	return t != core.SmartContract_ABI_Entry_Fallback &&
		t != core.SmartContract_ABI_Entry_Receive
}

// entryHasOutputs reports whether the entry type carries an outputs field
// in canonical Ethereum ABI JSON.
func entryHasOutputs(t core.SmartContract_ABI_Entry_EntryType) bool {
	return t == core.SmartContract_ABI_Entry_Function ||
		t == core.SmartContract_ABI_Entry_Error
}

// entryHasInputs reports whether the entry type carries an inputs field
// in canonical Ethereum ABI JSON.
func entryHasInputs(t core.SmartContract_ABI_Entry_EntryType) bool {
	return t != core.SmartContract_ABI_Entry_Fallback &&
		t != core.SmartContract_ABI_Entry_Receive
}

// entryHasMutability reports whether the entry type carries a
// stateMutability field in canonical Ethereum ABI JSON.
func entryHasMutability(t core.SmartContract_ABI_Entry_EntryType) bool {
	return t == core.SmartContract_ABI_Entry_Constructor ||
		t == core.SmartContract_ABI_Entry_Function ||
		t == core.SmartContract_ABI_Entry_Fallback ||
		t == core.SmartContract_ABI_Entry_Receive
}

// FormatABIEntry converts a single proto ABI entry into a map using
// human-readable string labels for type and stateMutability, matching the
// canonical Ethereum ABI JSON format. Fields that are not applicable to
// a given entry type are omitted (e.g., events have no outputs).
// A nil entry returns an empty map.
func FormatABIEntry(entry *core.SmartContract_ABI_Entry) map[string]any {
	if entry == nil {
		return map[string]any{}
	}

	eType := entry.GetType()
	m := map[string]any{}

	if t, ok := entryTypeName[eType]; ok {
		m["type"] = t
	}

	if entryHasName(eType) {
		m["name"] = entry.GetName()
	}

	if entryHasMutability(eType) {
		if sm, ok := stateMutabilityName[entry.GetStateMutability()]; ok {
			m["stateMutability"] = sm
		}
	}

	if entry.GetAnonymous() {
		m["anonymous"] = true
	}
	if entry.GetPayable() {
		m["payable"] = true
	}
	if entry.GetConstant() {
		m["constant"] = true
	}

	if entryHasInputs(eType) {
		if inputs := entry.GetInputs(); len(inputs) > 0 {
			m["inputs"] = formatParams(inputs)
		} else {
			m["inputs"] = []map[string]any{}
		}
	}

	if entryHasOutputs(eType) {
		if outputs := entry.GetOutputs(); len(outputs) > 0 {
			m["outputs"] = formatParams(outputs)
		} else {
			m["outputs"] = []map[string]any{}
		}
	}

	return m
}

// FormatABI converts an entire proto SmartContract_ABI into a slice of
// human-readable maps suitable for JSON serialization. A nil ABI returns
// an empty slice.
func FormatABI(abi *core.SmartContract_ABI) []map[string]any {
	entries := abi.GetEntrys()
	result := make([]map[string]any, len(entries))
	for i, entry := range entries {
		result[i] = FormatABIEntry(entry)
	}
	return result
}

// formatParams converts proto ABI entry params to a slice of maps.
func formatParams(params []*core.SmartContract_ABI_Entry_Param) []map[string]any {
	out := make([]map[string]any, len(params))
	for i, p := range params {
		pm := map[string]any{
			"name": p.GetName(),
			"type": p.GetType(),
		}
		if p.GetIndexed() {
			pm["indexed"] = true
		}
		out[i] = pm
	}
	return out
}

// GetInputsParser returns input method parser arguments from ABI
func GetInputsParser(ABI *core.SmartContract_ABI, method string) (eABI.Arguments, error) {
	arguments := eABI.Arguments{}
	for _, entry := range ABI.Entrys {
		if entry.Type != core.SmartContract_ABI_Entry_Function {
			continue
		}
		if matchEntry(entry, method) {
			for _, out := range entry.Inputs {
				ty, err := eABI.NewType(out.Type, "", nil)
				if err != nil {
					return nil, fmt.Errorf("invalid param %s: %+v", out.Type, err)
				}
				arguments = append(arguments, eABI.Argument{
					Name:    out.Name,
					Type:    ty,
					Indexed: out.Indexed,
				})
			}
			return arguments, nil
		}
	}
	return nil, fmt.Errorf("not found")
}
