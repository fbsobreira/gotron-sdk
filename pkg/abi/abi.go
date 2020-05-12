package abi

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
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

// GetPaddedParam from struct
func GetPaddedParam(p Param) ([]byte, error) {
	for k := range p {
		switch k {
		case "uint16", "uint32", "uint64",
			"int16", "int32", "int64":
			n := big.NewInt(int64(p[k].(float64)))
			b := make([]byte, 32-len(n.Bytes()))
			return append(b, n.Bytes()...), nil
		case "uint", "uint128", "uint256",
			"int", "int128", "int256":
			n, ok := new(big.Int).SetString(p[k].(string), 10)
			if !ok {
				return nil, fmt.Errorf("Invalid value %s", p[k].(string))
			}
			b := make([]byte, 32-len(n.Bytes()))
			return append(b, n.Bytes()...), nil
		case "address":
			addr, err := address.Base58ToAddress(p[k].(string))
			if err != nil {
				return nil, fmt.Errorf("Invalid value %s", p[k].(string))
			}
			b := make([]byte, 12)
			return append(b, addr[1:]...), nil
		// TODO: decode another types
		default:
			return nil, fmt.Errorf("Invalid type %s", k)
		}
	}

	return nil, nil
}

// Pack data into bytes
func Pack(method string, param []Param) ([]byte, error) {
	signature := Signature(method)
	// convert params to bytes
	for _, p := range param {
		pBytes, err := GetPaddedParam(p)
		if err != nil {
			return nil, err
		}
		signature = append(signature, pBytes...)
	}
	return signature, nil
}
