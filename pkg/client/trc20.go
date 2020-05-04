package client

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode/utf8"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

const (
	trc20TransferMtrxodSignature = "0xa9059cbb"
	trc20TransferEventSignature  = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	trc20NameSignature           = "0x06fdde03"
	trc20SymbolSignature         = "0x95d89b41"
	trc20DecimalsSignature       = "0x313ce567"
	trc20BalanceOf               = "0x70a08231"
)

// TRC20Call make cosntant calll
func (g *GrpcClient) TRC20Call(from, contractAddress, data string) (*api.TransactionExtention, error) {
	var err error
	fromDesc := address.HexToAddress("410000000000000000000000000000000000000000")
	if len(from) > 0 {
		fromDesc, err = address.Base58ToAddress(from)
		if err != nil {
			return nil, err
		}
	}
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}
	dataBytes, err := common.FromHex(data)
	if err != nil {
		return nil, err
	}
	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            dataBytes,
	}

	return g.TriggerConstantContract(ct)
}

// TRC20GetName get token name
func (g *GrpcClient) TRC20GetName(contractAddress string) (string, error) {
	result, err := g.TRC20Call("", contractAddress, trc20NameSignature)
	if err != nil {
		return "", err
	}
	data := common.ToHex(result.GetConstantResult()[0])
	return g.ParseTRC20StringProperty(data)
}

// TRC20GetSymbol get contract symbol
func (g *GrpcClient) TRC20GetSymbol(contractAddress string) (string, error) {
	result, err := g.TRC20Call("", contractAddress, trc20SymbolSignature)
	if err != nil {
		return "", err
	}
	data := common.ToHex(result.GetConstantResult()[0])
	return g.ParseTRC20StringProperty(data)
}

// TRC20GetDecimals get contract decimals
func (g *GrpcClient) TRC20GetDecimals(contractAddress string) (*big.Int, error) {
	result, err := g.TRC20Call("", contractAddress, trc20DecimalsSignature)
	if err != nil {
		return nil, err
	}
	data := common.ToHex(result.GetConstantResult()[0])
	return g.ParseTRC20NumericProperty(data)
}

// ParseTRC20NumericProperty get number from data
func (g *GrpcClient) ParseTRC20NumericProperty(data string) (*big.Int, error) {
	if common.Has0xPrefix(data) {
		data = data[2:]
	}
	if len(data) == 64 {
		var n big.Int
		_, ok := n.SetString(data, 16)
		if ok {
			return &n, nil
		}
	}
	return nil, fmt.Errorf("Cannot parse %s", data)
}

// ParseTRC20StringProperty get string from data
func (g *GrpcClient) ParseTRC20StringProperty(data string) (string, error) {
	if common.Has0xPrefix(data) {
		data = data[2:]
	}
	if len(data) > 128 {
		n, _ := g.ParseTRC20NumericProperty(data[64:128])
		if n != nil {
			l := n.Uint64()
			if 2*int(l) <= len(data)-128 {
				b, err := hex.DecodeString(data[128 : 128+2*l])
				if err == nil {
					return string(b), nil
				}
			}
		}
	} else if len(data) == 64 {
		// allow string properties as 32 bytes of UTF-8 data
		b, err := hex.DecodeString(data)
		if err == nil {
			i := bytes.Index(b, []byte{0})
			if i > 0 {
				b = b[:i]
			}
			if utf8.Valid(b) {
				return string(b), nil
			}
		}
	}
	return "", fmt.Errorf("Cannot parse %s,", data)
}
