package abi

import (
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"strconv"

	eABI "github.com/ethereum/go-ethereum/accounts/abi"
	eCommon "github.com/ethereum/go-ethereum/common"
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

func convetToAddress(v interface{}) (eCommon.Address, error) {
	switch v.(type) {
	case string:
		addr, err := address.Base58ToAddress(v.(string))
		if err != nil {
			return eCommon.Address{}, fmt.Errorf("invalid address %s: %+v", v.(string), err)
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
		v, _ = new(big.Int).SetString(v.(string), 10)
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
				return nil, fmt.Errorf("invalid parem %+v: %+v", p, err)
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
					tmp := v.([]string)
					v = make([]eCommon.Address, 0)
					for i := range tmp {
						addr, err := convetToAddress(tmp[i])
						if err != nil {
							return nil, err
						}
						v = append(v.([]eCommon.Address), addr)
					}
				}

				if (ty.Elem.T == eABI.IntTy || ty.Elem.T == eABI.UintTy) &&
					ty.Elem.Size > 64 &&
					reflect.TypeOf(v).Elem().Kind() == reflect.String {
					tmp := make([]*big.Int, 0)
					for _, s := range v.([]string) {
						value, _ := new(big.Int).SetString(s, 10)
						tmp = append(tmp, value)
					}
					v = tmp
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

			values = append(values, v)
		}
	}
	// convert params to bytes
	return arguments.PackValues(values)
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
