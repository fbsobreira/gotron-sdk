// Package trc20 provides a typed, high-level wrapper for TRC20 token
// interactions. It is built on top of the contract call builder and
// requires no ABI management from the caller.
package trc20

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
)

// Well-known EVM function selectors for TRC20.
const (
	selectorName         = "06fdde03"
	selectorSymbol       = "95d89b41"
	selectorDecimals     = "313ce567"
	selectorTotalSupply  = "18160ddd"
	selectorBalanceOf    = "70a08231"
	selectorTransfer     = "a9059cbb"
	selectorApprove      = "095ea7b3"
	selectorTransferFrom = "23b872dd"
	selectorAllowance    = "dd62ed3e"
)

// TokenInfo holds metadata returned by Info.
type TokenInfo struct {
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
}

// Balance holds a token balance with both raw and display forms.
type Balance struct {
	Raw     *big.Int
	Display string
	Symbol  string
}

// Token is a typed TRC20 token handle.
type Token struct {
	client          contract.Client
	contractAddress string
}

// New creates a Token instance for the given TRC20 contract.
func New(client contract.Client, contractAddress string) *Token {
	return &Token{
		client:          client,
		contractAddress: contractAddress,
	}
}

// Info retrieves the token name, symbol, decimals, and total supply.
func (t *Token) Info(ctx context.Context) (*TokenInfo, error) {
	name, err := t.Name(ctx)
	if err != nil {
		return nil, fmt.Errorf("name: %w", err)
	}
	symbol, err := t.Symbol(ctx)
	if err != nil {
		return nil, fmt.Errorf("symbol: %w", err)
	}
	decimals, err := t.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("decimals: %w", err)
	}
	totalSupply, err := t.TotalSupply(ctx)
	if err != nil {
		return nil, fmt.Errorf("totalSupply: %w", err)
	}
	return &TokenInfo{
		Name:        name,
		Symbol:      symbol,
		Decimals:    decimals,
		TotalSupply: totalSupply,
	}, nil
}

// Name returns the token name.
func (t *Token) Name(ctx context.Context) (string, error) {
	data, err := hex.DecodeString(selectorName)
	if err != nil {
		return "", err
	}
	result, err := contract.New(t.client, t.contractAddress).
		WithData(data).
		Call(ctx)
	if err != nil {
		return "", err
	}
	return decodeString(result.RawResults)
}

// Symbol returns the token symbol.
func (t *Token) Symbol(ctx context.Context) (string, error) {
	data, err := hex.DecodeString(selectorSymbol)
	if err != nil {
		return "", err
	}
	result, err := contract.New(t.client, t.contractAddress).
		WithData(data).
		Call(ctx)
	if err != nil {
		return "", err
	}
	return decodeString(result.RawResults)
}

// Decimals returns the token decimals.
func (t *Token) Decimals(ctx context.Context) (uint8, error) {
	data, err := hex.DecodeString(selectorDecimals)
	if err != nil {
		return 0, err
	}
	result, err := contract.New(t.client, t.contractAddress).
		WithData(data).
		Call(ctx)
	if err != nil {
		return 0, err
	}
	n, err := decodeUint256(result.RawResults)
	if err != nil {
		return 0, err
	}
	if !n.IsUint64() || n.Uint64() > 255 {
		return 0, fmt.Errorf("decimals value %s out of uint8 range", n.String())
	}
	return uint8(n.Uint64()), nil
}

// TotalSupply returns the total token supply.
func (t *Token) TotalSupply(ctx context.Context) (*big.Int, error) {
	data, err := hex.DecodeString(selectorTotalSupply)
	if err != nil {
		return nil, err
	}
	result, err := contract.New(t.client, t.contractAddress).
		WithData(data).
		Call(ctx)
	if err != nil {
		return nil, err
	}
	return decodeUint256(result.RawResults)
}

// BalanceOf returns the balance of the given address with both raw and
// human-readable display format.
func (t *Token) BalanceOf(ctx context.Context, addr string) (*Balance, error) {
	addrBytes, err := address.Base58ToAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid address %s: %w", addr, err)
	}

	callData := encodeWithAddress(selectorBalanceOf, addrBytes)

	result, err := contract.New(t.client, t.contractAddress).
		WithData(callData).
		Call(ctx)
	if err != nil {
		return nil, err
	}
	raw, err := decodeUint256(result.RawResults)
	if err != nil {
		return nil, err
	}

	decimals, err := t.Decimals(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching decimals: %w", err)
	}

	symbol, err := t.Symbol(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching symbol: %w", err)
	}

	return &Balance{
		Raw:     raw,
		Display: formatBalance(raw, decimals),
		Symbol:  symbol,
	}, nil
}

// Allowance returns the remaining allowance that spender can spend on behalf of owner.
func (t *Token) Allowance(ctx context.Context, owner, spender string) (*big.Int, error) {
	ownerBytes, err := address.Base58ToAddress(owner)
	if err != nil {
		return nil, fmt.Errorf("invalid owner address %s: %w", owner, err)
	}
	spenderBytes, err := address.Base58ToAddress(spender)
	if err != nil {
		return nil, fmt.Errorf("invalid spender address %s: %w", spender, err)
	}

	callData := encodeWithTwoAddresses(selectorAllowance, ownerBytes, spenderBytes)

	result, err := contract.New(t.client, t.contractAddress).
		WithData(callData).
		Call(ctx)
	if err != nil {
		return nil, err
	}
	return decodeUint256(result.RawResults)
}

// Transfer returns a ContractCall for transferring tokens.
// Call Send(ctx, signer) on the result to execute. Address validation
// errors are deferred and surface when a terminal (Build/Send) is called.
func (t *Token) Transfer(from, to string, amount *big.Int, opts ...contract.Option) *contract.ContractCall {
	toBytes, err := address.Base58ToAddress(to)
	if err != nil {
		return contract.New(t.client, t.contractAddress).
			SetError(fmt.Errorf("invalid to address %s: %w", to, err))
	}
	callData, err := encodeTransfer(selectorTransfer, toBytes, amount)
	if err != nil {
		return contract.New(t.client, t.contractAddress).SetError(err)
	}

	return contract.New(t.client, t.contractAddress).
		From(from).
		WithData(callData).
		Apply(opts...)
}

// Approve returns a ContractCall for approving a spender.
func (t *Token) Approve(from, spender string, amount *big.Int, opts ...contract.Option) *contract.ContractCall {
	spenderBytes, err := address.Base58ToAddress(spender)
	if err != nil {
		return contract.New(t.client, t.contractAddress).
			SetError(fmt.Errorf("invalid spender address %s: %w", spender, err))
	}
	callData, err := encodeTransfer(selectorApprove, spenderBytes, amount)
	if err != nil {
		return contract.New(t.client, t.contractAddress).SetError(err)
	}

	return contract.New(t.client, t.contractAddress).
		From(from).
		WithData(callData).
		Apply(opts...)
}

// TransferFrom returns a ContractCall for transferring tokens on behalf of
// another address (requires prior approval). The caller parameter is the
// address that signs the transaction (the approved spender).
func (t *Token) TransferFrom(caller, from, to string, amount *big.Int, opts ...contract.Option) *contract.ContractCall {
	fromBytes, err := address.Base58ToAddress(from)
	if err != nil {
		return contract.New(t.client, t.contractAddress).
			SetError(fmt.Errorf("invalid from address %s: %w", from, err))
	}
	toBytes, err := address.Base58ToAddress(to)
	if err != nil {
		return contract.New(t.client, t.contractAddress).
			SetError(fmt.Errorf("invalid to address %s: %w", to, err))
	}
	callData, err := encodeTransferFrom(fromBytes, toBytes, amount)
	if err != nil {
		return contract.New(t.client, t.contractAddress).SetError(err)
	}

	return contract.New(t.client, t.contractAddress).
		From(caller).
		WithData(callData).
		Apply(opts...)
}

// formatBalance converts a raw big.Int token amount to a human-readable string
// with the given number of decimals.
func formatBalance(raw *big.Int, decimals uint8) string {
	if raw == nil || raw.Sign() == 0 {
		return "0"
	}

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Div(raw, divisor)
	remainder := new(big.Int).Mod(raw, divisor)

	if remainder.Sign() == 0 {
		return addThousandSeparators(whole.String())
	}

	fracStr := remainder.String()
	// Pad with leading zeros to match decimals count.
	for len(fracStr) < int(decimals) {
		fracStr = "0" + fracStr
	}
	// Trim trailing zeros.
	fracStr = strings.TrimRight(fracStr, "0")

	return addThousandSeparators(whole.String()) + "." + fracStr
}

// addThousandSeparators inserts commas as thousand separators.
func addThousandSeparators(s string) string {
	if len(s) <= 3 {
		return s
	}

	var b strings.Builder
	start := len(s) % 3
	if start > 0 {
		b.WriteString(s[:start])
	}
	for i := start; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
