package trc20

import (
	"math/big"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/standards/trc20enc"
)

// encodeWithAddress builds selector + ABI-encoded address (32-byte padded).
func encodeWithAddress(selector string, addr address.Address) []byte {
	return trc20enc.EncodeWithAddress(trc20enc.SelectorBytes(selector), addr)
}

// encodeWithTwoAddresses builds selector + two ABI-encoded addresses.
func encodeWithTwoAddresses(selector string, addr1, addr2 address.Address) []byte {
	return trc20enc.EncodeWithTwoAddresses(trc20enc.SelectorBytes(selector), addr1, addr2)
}

// encodeTransfer builds selector + address + uint256.
func encodeTransfer(selector string, addr address.Address, amount *big.Int) ([]byte, error) {
	return trc20enc.EncodeAddressAmount(trc20enc.SelectorBytes(selector), addr, amount)
}

// encodeTransferFrom builds transferFrom(address,address,uint256) call data.
func encodeTransferFrom(from, to address.Address, amount *big.Int) ([]byte, error) {
	return trc20enc.EncodeTransferFromRaw(from, to, amount)
}

// padAddress converts a 21-byte TRON address to a 32-byte ABI-encoded value.
// The first byte (0x41 prefix) is dropped, leaving 20 bytes left-padded to 32.
func padAddress(addr address.Address) []byte {
	return trc20enc.PadAddress(addr)
}

// padUint256 left-pads a non-negative big.Int to 32 bytes.
// Returns an error for nil, negative, or >256-bit values.
func padUint256(n *big.Int) ([]byte, error) {
	return trc20enc.PadUint256(n)
}

// decodeUint256 extracts a big.Int from ABI-encoded constant result slices.
func decodeUint256(results [][]byte) (*big.Int, error) {
	return trc20enc.DecodeUint256Results(results)
}

// decodeString extracts a string from ABI-encoded constant result slices.
func decodeString(results [][]byte) (string, error) {
	return trc20enc.DecodeStringResults(results)
}
