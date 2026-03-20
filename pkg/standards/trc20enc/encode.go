// Package trc20enc provides low-level ABI encoding and decoding helpers for
// TRC20 token interactions. It has no dependency on the gRPC client or
// contract builder, making it safe to import from any layer.
package trc20enc

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode/utf8"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

// Well-known EVM function selectors for TRC20 (hex-encoded, no 0x prefix).
const (
	SelectorName         = "06fdde03"
	SelectorSymbol       = "95d89b41"
	SelectorDecimals     = "313ce567"
	SelectorTotalSupply  = "18160ddd"
	SelectorBalanceOf    = "70a08231"
	SelectorTransfer     = "a9059cbb"
	SelectorApprove      = "095ea7b3"
	SelectorTransferFrom = "23b872dd"
	SelectorAllowance    = "dd62ed3e"
)

// Pre-decoded selector bytes, cached at init to avoid repeated hex decoding.
var (
	selectorNameBytes         []byte
	selectorSymbolBytes       []byte
	selectorDecimalsBytes     []byte
	selectorTotalSupplyBytes  []byte
	selectorBalanceOfBytes    []byte
	selectorTransferBytes     []byte
	selectorApproveBytes      []byte
	selectorTransferFromBytes []byte
	selectorAllowanceBytes    []byte
)

func init() {
	selectorNameBytes, _ = hex.DecodeString(SelectorName)
	selectorSymbolBytes, _ = hex.DecodeString(SelectorSymbol)
	selectorDecimalsBytes, _ = hex.DecodeString(SelectorDecimals)
	selectorTotalSupplyBytes, _ = hex.DecodeString(SelectorTotalSupply)
	selectorBalanceOfBytes, _ = hex.DecodeString(SelectorBalanceOf)
	selectorTransferBytes, _ = hex.DecodeString(SelectorTransfer)
	selectorApproveBytes, _ = hex.DecodeString(SelectorApprove)
	selectorTransferFromBytes, _ = hex.DecodeString(SelectorTransferFrom)
	selectorAllowanceBytes, _ = hex.DecodeString(SelectorAllowance)
}

// SelectorBytes returns a copy of the pre-decoded bytes for the given hex selector.
// Returns nil if the selector is not recognized.
func SelectorBytes(selector string) []byte {
	var src []byte
	switch selector {
	case SelectorName:
		src = selectorNameBytes
	case SelectorSymbol:
		src = selectorSymbolBytes
	case SelectorDecimals:
		src = selectorDecimalsBytes
	case SelectorTotalSupply:
		src = selectorTotalSupplyBytes
	case SelectorBalanceOf:
		src = selectorBalanceOfBytes
	case SelectorTransfer:
		src = selectorTransferBytes
	case SelectorApprove:
		src = selectorApproveBytes
	case SelectorTransferFrom:
		src = selectorTransferFromBytes
	case SelectorAllowance:
		src = selectorAllowanceBytes
	default:
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

// EncodeBalanceOf builds the ABI call data hex string for balanceOf(address).
func EncodeBalanceOf(addr address.Address) string {
	return hex.EncodeToString(EncodeWithAddress(selectorBalanceOfBytes, addr))
}

// EncodeTransferCall builds the ABI call data hex string for
// transfer(address,uint256).
func EncodeTransferCall(to address.Address, amount *big.Int) (string, error) {
	data, err := EncodeAddressAmount(selectorTransferBytes, to, amount)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// EncodeApproveCall builds the ABI call data hex string for
// approve(address,uint256).
func EncodeApproveCall(spender address.Address, amount *big.Int) (string, error) {
	data, err := EncodeAddressAmount(selectorApproveBytes, spender, amount)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// EncodeTransferFromCall builds the ABI call data hex string for
// transferFrom(address,address,uint256).
func EncodeTransferFromCall(from, to address.Address, amount *big.Int) (string, error) {
	data, err := EncodeTransferFromRaw(from, to, amount)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

// EncodeAddressAmount builds selector + address + uint256 as raw bytes.
func EncodeAddressAmount(sel []byte, addr address.Address, amount *big.Int) ([]byte, error) {
	amountBytes, err := PadUint256(amount)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 0, len(sel)+64)
	buf = append(buf, sel...)
	buf = append(buf, PadAddress(addr)...)
	buf = append(buf, amountBytes...)
	return buf, nil
}

// EncodeWithAddress builds selector + ABI-encoded address (32-byte padded)
// as raw bytes.
func EncodeWithAddress(sel []byte, addr address.Address) []byte {
	return append(append([]byte(nil), sel...), PadAddress(addr)...)
}

// EncodeWithTwoAddresses builds selector + two ABI-encoded addresses as raw
// bytes.
func EncodeWithTwoAddresses(sel []byte, addr1, addr2 address.Address) []byte {
	buf := make([]byte, 0, len(sel)+64)
	buf = append(buf, sel...)
	buf = append(buf, PadAddress(addr1)...)
	buf = append(buf, PadAddress(addr2)...)
	return buf
}

// EncodeTransferFromRaw builds transferFrom(address,address,uint256) call
// data as raw bytes.
func EncodeTransferFromRaw(from, to address.Address, amount *big.Int) ([]byte, error) {
	amountBytes, err := PadUint256(amount)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 0, len(selectorTransferFromBytes)+96)
	buf = append(buf, selectorTransferFromBytes...)
	buf = append(buf, PadAddress(from)...)
	buf = append(buf, PadAddress(to)...)
	buf = append(buf, amountBytes...)
	return buf, nil
}

// PadAddress converts a 21-byte TRON address to a 32-byte ABI-encoded value.
// The first byte (0x41 prefix) is dropped, leaving 20 bytes left-padded to 32.
func PadAddress(addr address.Address) []byte {
	var buf [32]byte
	if len(addr) == address.AddressLength {
		copy(buf[12:], addr[1:])
	}
	return buf[:]
}

// PadUint256 left-pads a non-negative big.Int to 32 bytes.
// Returns an error for nil, negative, or >256-bit values.
func PadUint256(n *big.Int) ([]byte, error) {
	if n == nil {
		return nil, fmt.Errorf("invalid uint256: nil value")
	}
	if n.Sign() < 0 {
		return nil, fmt.Errorf("invalid uint256: negative value %s", n.String())
	}
	if n.BitLen() > 256 {
		return nil, fmt.Errorf("invalid uint256: value exceeds 256 bits")
	}
	var buf [32]byte
	b := n.Bytes()
	copy(buf[32-len(b):], b)
	return buf[:], nil
}

// DecodeUint256 extracts a big.Int from ABI-encoded bytes.
func DecodeUint256(data []byte) (*big.Int, error) {
	if len(data) < 32 {
		return nil, fmt.Errorf("invalid uint256 result: insufficient data")
	}
	return new(big.Int).SetBytes(data[:32]), nil
}

// DecodeUint256Results extracts a big.Int from ABI-encoded constant result
// slices.
func DecodeUint256Results(results [][]byte) (*big.Int, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("invalid uint256 result: insufficient data")
	}
	return DecodeUint256(results[0])
}

// DecodeString extracts a string from ABI-encoded bytes.
// Handles both standard ABI encoding (offset+length+data) and fixed
// 32-byte UTF-8 values (as used by some older tokens like MKR).
func DecodeString(data []byte) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("empty result")
	}

	// Standard ABI string encoding: offset (32) + length (32) + data
	if len(data) > 64 {
		lengthBytes := data[32:64]
		lengthBI := new(big.Int).SetBytes(lengthBytes)
		if lengthBI.BitLen() > 63 {
			return "", fmt.Errorf("string length exceeds maximum")
		}
		length := lengthBI.Uint64()
		if 64+length <= uint64(len(data)) {
			return string(data[64 : 64+length]), nil
		}
	}

	// Fallback: 32-byte fixed UTF-8 value (null-terminated)
	if len(data) >= 32 {
		b := data[:32]
		if i := bytes.IndexByte(b, 0); i >= 0 {
			b = b[:i]
		}
		if utf8.Valid(b) && len(b) > 0 {
			return string(b), nil
		}
	}

	return "", fmt.Errorf("cannot decode string from result")
}

// DecodeStringResults extracts a string from ABI-encoded constant result
// slices.
func DecodeStringResults(results [][]byte) (string, error) {
	if len(results) == 0 || len(results[0]) == 0 {
		return "", fmt.Errorf("empty result")
	}
	return DecodeString(results[0])
}
