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

func convertToInt(ty eABI.Type, v interface{}) interface{} {
	if ty.T == eABI.IntTy && ty.Size <= 64 {
		tmp, _ := strconv.ParseInt(v.(string), 10, ty.Size)
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
	} else if ty.T == eABI.UintTy && ty.Size <= 64 {
		tmp, _ := strconv.ParseUint(v.(string), 10, ty.Size)
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
		s := v.(string)
		// check for hex char
		if strings.HasPrefix(s, "0x") {
			v, _ = new(big.Int).SetString(s[2:], 16)
		} else {
			v, _ = new(big.Int).SetString(s, 10)
		}
	}
	return v
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
							if strings.HasPrefix(s, "0x") {
								tmp[i], _ = new(big.Int).SetString(s[2:], 16)
							} else {
								tmp[i], _ = new(big.Int).SetString(s, 10)
							}
						}
						v = tmp
					} else {
						v = convertSmallIntSlice(*ty.Elem, strs)
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
			if (ty.T == eABI.IntTy || ty.T == eABI.UintTy) && reflect.TypeOf(v).Kind() == reflect.String {
				v = convertToInt(ty, v)
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
func convertSmallIntSlice(elemTy eABI.Type, strs []string) interface{} {
	switch {
	case elemTy.T == eABI.UintTy && elemTy.Size == 8:
		out := make([]uint8, len(strs))
		for i, s := range strs {
			tmp, _ := strconv.ParseUint(s, 10, 8)
			out[i] = uint8(tmp)
		}
		return out
	case elemTy.T == eABI.UintTy && elemTy.Size == 16:
		out := make([]uint16, len(strs))
		for i, s := range strs {
			tmp, _ := strconv.ParseUint(s, 10, 16)
			out[i] = uint16(tmp)
		}
		return out
	case elemTy.T == eABI.UintTy && elemTy.Size == 32:
		out := make([]uint32, len(strs))
		for i, s := range strs {
			tmp, _ := strconv.ParseUint(s, 10, 32)
			out[i] = uint32(tmp)
		}
		return out
	case elemTy.T == eABI.UintTy && elemTy.Size == 64:
		out := make([]uint64, len(strs))
		for i, s := range strs {
			tmp, _ := strconv.ParseUint(s, 10, 64)
			out[i] = tmp
		}
		return out
	case elemTy.T == eABI.IntTy && elemTy.Size == 8:
		out := make([]int8, len(strs))
		for i, s := range strs {
			tmp, _ := strconv.ParseInt(s, 10, 8)
			out[i] = int8(tmp)
		}
		return out
	case elemTy.T == eABI.IntTy && elemTy.Size == 16:
		out := make([]int16, len(strs))
		for i, s := range strs {
			tmp, _ := strconv.ParseInt(s, 10, 16)
			out[i] = int16(tmp)
		}
		return out
	case elemTy.T == eABI.IntTy && elemTy.Size == 32:
		out := make([]int32, len(strs))
		for i, s := range strs {
			tmp, _ := strconv.ParseInt(s, 10, 32)
			out[i] = int32(tmp)
		}
		return out
	case elemTy.T == eABI.IntTy && elemTy.Size == 64:
		out := make([]int64, len(strs))
		for i, s := range strs {
			tmp, _ := strconv.ParseInt(s, 10, 64)
			out[i] = tmp
		}
		return out
	default:
		// Fallback to big.Int for unexpected sizes
		out := make([]*big.Int, len(strs))
		for i, s := range strs {
			out[i], _ = new(big.Int).SetString(s, 10)
		}
		return out
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
			addrBytes := make([]byte, 1+len(addr.Bytes()))
			addrBytes[0] = address.TronBytePrefix
			copy(addrBytes[1:], addr.Bytes())
			out[k] = address.Address(addrBytes)
		}
	}

	return nil
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
