package trc20

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode/utf8"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

// encodeWithAddress builds selector + ABI-encoded address (32-byte padded).
func encodeWithAddress(selector string, addr address.Address) []byte {
	sel, _ := hex.DecodeString(selector)
	return append(sel, padAddress(addr)...)
}

// encodeWithTwoAddresses builds selector + two ABI-encoded addresses.
func encodeWithTwoAddresses(selector string, addr1, addr2 address.Address) []byte {
	sel, _ := hex.DecodeString(selector)
	buf := make([]byte, 0, len(sel)+64)
	buf = append(buf, sel...)
	buf = append(buf, padAddress(addr1)...)
	buf = append(buf, padAddress(addr2)...)
	return buf
}

// encodeTransfer builds selector + address + uint256.
func encodeTransfer(selector string, addr address.Address, amount *big.Int) ([]byte, error) {
	amountBytes, err := padUint256(amount)
	if err != nil {
		return nil, err
	}
	sel, _ := hex.DecodeString(selector)
	buf := make([]byte, 0, len(sel)+64)
	buf = append(buf, sel...)
	buf = append(buf, padAddress(addr)...)
	buf = append(buf, amountBytes...)
	return buf, nil
}

// encodeTransferFrom builds transferFrom(address,address,uint256) call data.
func encodeTransferFrom(from, to address.Address, amount *big.Int) ([]byte, error) {
	amountBytes, err := padUint256(amount)
	if err != nil {
		return nil, err
	}
	sel, _ := hex.DecodeString(selectorTransferFrom)
	buf := make([]byte, 0, len(sel)+96)
	buf = append(buf, sel...)
	buf = append(buf, padAddress(from)...)
	buf = append(buf, padAddress(to)...)
	buf = append(buf, amountBytes...)
	return buf, nil
}

// padAddress converts a 21-byte TRON address to a 32-byte ABI-encoded value.
// The first byte (0x41 prefix) is dropped, leaving 20 bytes left-padded to 32.
func padAddress(addr address.Address) []byte {
	var buf [32]byte
	// addr is 21 bytes: [0x41, <20-byte EVM address>]
	if len(addr) == address.AddressLength {
		copy(buf[12:], addr[1:]) // skip TRON prefix
	}
	return buf[:]
}

// padUint256 left-pads a non-negative big.Int to 32 bytes.
// Returns an error for nil, negative, or >256-bit values.
func padUint256(n *big.Int) ([]byte, error) {
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

// decodeUint256 extracts a big.Int from ABI-encoded constant result.
func decodeUint256(results [][]byte) (*big.Int, error) {
	if len(results) == 0 || len(results[0]) < 32 {
		return nil, fmt.Errorf("invalid uint256 result: insufficient data")
	}
	return new(big.Int).SetBytes(results[0][:32]), nil
}

// decodeString extracts a string from ABI-encoded constant result.
// Handles both standard ABI encoding (offset+length+data) and fixed
// 32-byte UTF-8 values (as used by some older tokens like MKR).
func decodeString(results [][]byte) (string, error) {
	if len(results) == 0 || len(results[0]) == 0 {
		return "", fmt.Errorf("empty result")
	}
	data := results[0]

	// Standard ABI string encoding: offset (32) + length (32) + data
	if len(data) > 64 {
		lengthBytes := data[32:64]
		length := new(big.Int).SetBytes(lengthBytes).Uint64()
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
