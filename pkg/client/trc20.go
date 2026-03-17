package client

import (
	"bytes"
	"context"
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
	trc20TransferMethodSignature     = "0xa9059cbb"
	trc20ApproveMethodSignature      = "0x095ea7b3"
	trc20TransferEventSignature      = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	trc20NameSignature               = "0x06fdde03"
	trc20SymbolSignature             = "0x95d89b41"
	trc20DecimalsSignature           = "0x313ce567"
	trc20BalanceOf                   = "0x70a08231"
	trc20TransferFromMethodSignature = "0x23b872dd"
)

// TRC20Call make cosntant calll
func (g *GrpcClient) TRC20Call(from, contractAddress, data string, constant bool, feeLimit int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TRC20CallCtx(ctx, from, contractAddress, data, constant, feeLimit)
}

// TRC20CallCtx is the context-aware version of TRC20Call.
func (g *GrpcClient) TRC20CallCtx(ctx context.Context, from, contractAddress, data string, constant bool, feeLimit int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

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
		result, err = g.triggerConstantContract(ctx, ct)
	} else {
		result, err = g.triggerContract(ctx, ct, feeLimit)
	}
	if err != nil {
		return nil, err
	}
	if result.Result.Code > 0 {
		return result, fmt.Errorf("%s", string(result.Result.Message))
	}
	return result, nil

}

// TRC20GetName get token name
func (g *GrpcClient) TRC20GetName(contractAddress string) (string, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TRC20GetNameCtx(ctx, contractAddress)
}

// TRC20GetNameCtx is the context-aware version of TRC20GetName.
func (g *GrpcClient) TRC20GetNameCtx(ctx context.Context, contractAddress string) (string, error) {
	ctx = g.withAPIKey(ctx)

	result, err := g.TRC20CallCtx(ctx, "", contractAddress, trc20NameSignature, true, 0)
	if err != nil {
		return "", err
	}
	data := common.BytesToHexString(result.GetConstantResult()[0])
	return g.ParseTRC20StringProperty(data)
}

// TRC20GetSymbol get contract symbol
func (g *GrpcClient) TRC20GetSymbol(contractAddress string) (string, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TRC20GetSymbolCtx(ctx, contractAddress)
}

// TRC20GetSymbolCtx is the context-aware version of TRC20GetSymbol.
func (g *GrpcClient) TRC20GetSymbolCtx(ctx context.Context, contractAddress string) (string, error) {
	ctx = g.withAPIKey(ctx)

	result, err := g.TRC20CallCtx(ctx, "", contractAddress, trc20SymbolSignature, true, 0)
	if err != nil {
		return "", err
	}
	data := common.BytesToHexString(result.GetConstantResult()[0])
	return g.ParseTRC20StringProperty(data)
}

// TRC20GetDecimals get contract decimals
func (g *GrpcClient) TRC20GetDecimals(contractAddress string) (*big.Int, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TRC20GetDecimalsCtx(ctx, contractAddress)
}

// TRC20GetDecimalsCtx is the context-aware version of TRC20GetDecimals.
func (g *GrpcClient) TRC20GetDecimalsCtx(ctx context.Context, contractAddress string) (*big.Int, error) {
	ctx = g.withAPIKey(ctx)

	result, err := g.TRC20CallCtx(ctx, "", contractAddress, trc20DecimalsSignature, true, 0)
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
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TRC20ContractBalanceCtx(ctx, addr, contractAddress)
}

// TRC20ContractBalanceCtx is the context-aware version of TRC20ContractBalance.
func (g *GrpcClient) TRC20ContractBalanceCtx(ctx context.Context, addr, contractAddress string) (*big.Int, error) {
	ctx = g.withAPIKey(ctx)

	addrB, err := address.Base58ToAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %s: %v", addr, addr)
	}
	req := trc20BalanceOf + "0000000000000000000000000000000000000000000000000000000000000000"[len(addrB.Hex())-2:] + addrB.Hex()[2:]
	result, err := g.TRC20CallCtx(ctx, "", contractAddress, req, true, 0)
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
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TRC20SendCtx(ctx, from, to, contract, amount, feeLimit)
}

// TRC20SendCtx is the context-aware version of TRC20Send.
func (g *GrpcClient) TRC20SendCtx(ctx context.Context, from, to, contract string, amount *big.Int, feeLimit int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	addrB, err := address.Base58ToAddress(to)
	if err != nil {
		return nil, err
	}
	ab := common.LeftPadBytes(amount.Bytes(), 32)
	req := trc20TransferMethodSignature + "0000000000000000000000000000000000000000000000000000000000000000"[len(addrB.Hex())-4:] + addrB.Hex()[4:]
	req += common.Bytes2Hex(ab)
	return g.TRC20CallCtx(ctx, from, contract, req, false, feeLimit)
}

// TRC20TransferFrom transfers tokens on behalf of another address (owner signs the tx).
func (g *GrpcClient) TRC20TransferFrom(owner, from, to, contract string, amount *big.Int, feeLimit int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TRC20TransferFromCtx(ctx, owner, from, to, contract, amount, feeLimit)
}

// TRC20TransferFromCtx is the context-aware version of TRC20TransferFrom.
func (g *GrpcClient) TRC20TransferFromCtx(ctx context.Context, owner, from, to, contract string, amount *big.Int, feeLimit int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	if _, err := address.Base58ToAddress(owner); err != nil {
		return nil, fmt.Errorf("invalid owner address: %w", err)
	}
	addrA, err := address.Base58ToAddress(from)
	if err != nil {
		return nil, err
	}
	addrB, err := address.Base58ToAddress(to)
	if err != nil {
		return nil, err
	}
	ab := common.LeftPadBytes(amount.Bytes(), 32)
	req := "0x23b872dd" +
		"0000000000000000000000000000000000000000000000000000000000000000"[len(addrA.Hex())-4:] + addrA.Hex()[4:] +
		"0000000000000000000000000000000000000000000000000000000000000000"[len(addrB.Hex())-4:] + addrB.Hex()[4:]
	req += common.Bytes2Hex(ab)
	return g.TRC20CallCtx(ctx, owner, contract, req, false, feeLimit)
}

// TRC20Approve approve token to address
func (g *GrpcClient) TRC20Approve(from, to, contract string, amount *big.Int, feeLimit int64) (*api.TransactionExtention, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TRC20ApproveCtx(ctx, from, to, contract, amount, feeLimit)
}

// TRC20ApproveCtx is the context-aware version of TRC20Approve.
func (g *GrpcClient) TRC20ApproveCtx(ctx context.Context, from, to, contract string, amount *big.Int, feeLimit int64) (*api.TransactionExtention, error) {
	ctx = g.withAPIKey(ctx)

	addrB, err := address.Base58ToAddress(to)
	if err != nil {
		return nil, err
	}
	ab := common.LeftPadBytes(amount.Bytes(), 32)
	req := trc20ApproveMethodSignature + "0000000000000000000000000000000000000000000000000000000000000000"[len(addrB.Hex())-4:] + addrB.Hex()[4:]
	req += common.Bytes2Hex(ab)
	return g.TRC20CallCtx(ctx, from, contract, req, false, feeLimit)
}
