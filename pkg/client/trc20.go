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
	trc20TransferMethodSignature = "0xa9059cbb"
	trc20ApproveMethodSignature  = "0x095ea7b3"
	trc20TransferEventSignature  = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	trc20NameSignature           = "0x06fdde03"
	trc20SymbolSignature         = "0x95d89b41"
	trc20DecimalsSignature       = "0x313ce567"
	trc20BalanceOf               = "0x70a08231"
)

// TRC20Call make cosntant calll
func (g *GrpcClient) TRC20Call(from, contractAddress, data string, constant bool, feeLimit int64) (*api.TransactionExtention, error) {
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
	var result *api.TransactionExtention
	if constant {
		result, err = g.triggerConstantContract(ct)
	} else {
		result, err = g.triggerContract(ct, feeLimit)
	}
	if err != nil {
		return nil, err
	}
	if result.Result.Code > 0 {
		return result, fmt.Errorf(string(result.Result.Message))
	}
	return result, nil

}

// TRC20GetName get token name
func (g *GrpcClient) TRC20GetName(contractAddress string) (string, error) {
	result, err := g.TRC20Call("", contractAddress, trc20NameSignature, true, 0)
	if err != nil {
		return "", err
	}
	data := common.BytesToHexString(result.GetConstantResult()[0])
	return g.ParseTRC20StringProperty(data)
}

// TRC20GetSymbol get contract symbol
func (g *GrpcClient) TRC20GetSymbol(contractAddress string) (string, error) {
	result, err := g.TRC20Call("", contractAddress, trc20SymbolSignature, true, 0)
	if err != nil {
		return "", err
	}
	data := common.BytesToHexString(result.GetConstantResult()[0])
	return g.ParseTRC20StringProperty(data)
}

// TRC20GetDecimals get contract decimals
func (g *GrpcClient) TRC20GetDecimals(contractAddress string) (*big.Int, error) {
	result, err := g.TRC20Call("", contractAddress, trc20DecimalsSignature, true, 0)
	if err != nil {
		return nil, err
	}
	data := common.BytesToHexString(result.GetConstantResult()[0])
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

	if len(data) == 0 {
		return big.NewInt(0), nil
	}

	return nil, fmt.Errorf("cannot parse %s", data)
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
	return "", fmt.Errorf("cannot parse %s,", data)
}

// TRC20ContractBalance get Address balance
func (g *GrpcClient) TRC20ContractBalance(addr, contractAddress string) (*big.Int, error) {
	addrB, err := address.Base58ToAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %s: %v", addr, addr)
	}
	req := trc20BalanceOf + "0000000000000000000000000000000000000000000000000000000000000000"[len(addrB.Hex())-2:] + addrB.Hex()[2:]
	result, err := g.TRC20Call("", contractAddress, req, true, 0)
	if err != nil {
		return nil, err
	}
	data := common.BytesToHexString(result.GetConstantResult()[0])
	r, err := g.ParseTRC20NumericProperty(data)
	if err != nil {
		return nil, fmt.Errorf("contract address %s: %v", contractAddress, err)
	}
	if r == nil {
		return nil, fmt.Errorf("contract address %s: invalid balance of %s", contractAddress, addr)
	}
	return r, nil
}

// TRC20Send send token to address
func (g *GrpcClient) TRC20Send(from, to, contract string, amount *big.Int, feeLimit int64) (*api.TransactionExtention, error) {
	addrB, err := address.Base58ToAddress(to)
	if err != nil {
		return nil, err
	}
	ab := common.LeftPadBytes(amount.Bytes(), 32)
	req := trc20TransferMethodSignature + "0000000000000000000000000000000000000000000000000000000000000000"[len(addrB.Hex())-4:] + addrB.Hex()[4:]
	req += common.Bytes2Hex(ab)
	return g.TRC20Call(from, contract, req, false, feeLimit)
}

// TRC20Approve approve token to address
func (g *GrpcClient) TRC20Approve(from, to, contract string, amount *big.Int, feeLimit int64) (*api.TransactionExtention, error) {
	addrB, err := address.Base58ToAddress(to)
	if err != nil {
		return nil, err
	}
	ab := common.LeftPadBytes(amount.Bytes(), 32)
	req := trc20ApproveMethodSignature + "0000000000000000000000000000000000000000000000000000000000000000"[len(addrB.Hex())-4:] + addrB.Hex()[4:]
	req += common.Bytes2Hex(ab)
	return g.TRC20Call(from, contract, req, false, feeLimit)
}
