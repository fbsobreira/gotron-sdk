// Package address provides TRON address encoding, decoding, and validation.
package address

import (
	"crypto/ecdsa"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

const (
	// HashLength is the expected length of the hash
	HashLength = 32
	// AddressLength is the expected length of the address
	AddressLength = 21
	// AddressLengthBase58 is the expected length of the address in base58format
	AddressLengthBase58 = 34
	// TronBytePrefix is the hex prefix to address
	TronBytePrefix = byte(0x41)
)

// Address represents the 21 byte address of an Tron account.
type Address []byte

// Bytes returns a copy of the raw byte representation of the address.
func (a Address) Bytes() []byte {
	result := make([]byte, len(a))
	copy(result, a)
	return result
}

// Hex returns the address encoded as a hex string (e.g. "41..." for mainnet).
func (a Address) Hex() string {
	return common.BytesToHexString(a[:])
}

// BigToAddress returns Address with byte values of b.
// Returns an error if b requires more than AddressLength bytes.
func BigToAddress(b *big.Int) (Address, error) {
	id := b.Bytes()
	if len(id) > AddressLength {
		return nil, fmt.Errorf("%d bytes, max %d: %w", len(id), AddressLength, ErrOversizeBigInt)
	}
	addr := make([]byte, AddressLength)
	copy(addr[AddressLength-len(id):], id)
	return addr, nil
}

// HexToAddress returns Address with byte values of s.
// Returns an error if s is not valid hex.
func HexToAddress(s string) (Address, error) {
	addr, err := common.FromHex(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", err.Error(), ErrInvalidHex)
	}
	return addr, nil
}

// Base58ToAddress returns Address with byte values of s.
func Base58ToAddress(s string) (Address, error) {
	addr, err := common.DecodeCheck(s)
	if err != nil {
		return nil, fmt.Errorf("base58 decode %q: %w", s, err)
	}
	return addr, nil
}

// Base64ToAddress returns Address with byte values of s.
func Base64ToAddress(s string) (Address, error) {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return Address(decoded), nil
}

// String implements fmt.Stringer.
func (a Address) String() string {
	if len(a) == 0 {
		return ""
	}

	if a[0] == 0 {
		return new(big.Int).SetBytes(a.Bytes()).String()
	}
	return common.EncodeCheck(a.Bytes())
}

// PubkeyToAddress derives a TRON address from an ECDSA public key.
func PubkeyToAddress(p ecdsa.PublicKey) Address {
	address := crypto.PubkeyToAddress(p)

	addressTron := make([]byte, 0)
	addressTron = append(addressTron, TronBytePrefix)
	addressTron = append(addressTron, address.Bytes()...)
	return addressTron
}

// BTCECPubkeyToAddress derives a TRON address from a btcec public key.
func BTCECPubkeyToAddress(p *btcec.PublicKey) Address {
	if p == nil {
		return nil
	}
	pubKey := p.ToECDSA()
	return PubkeyToAddress(*pubKey)
}

// BTCECPrivkeyToAddress derives a TRON address from a btcec private key.
func BTCECPrivkeyToAddress(p *btcec.PrivateKey) Address {
	if p == nil {
		return nil
	}
	pubKey := p.PubKey().ToECDSA()
	return PubkeyToAddress(*pubKey)
}

// BytesToAddress converts raw bytes to a TRON Address.
// For 20-byte input, the TRON mainnet prefix (0x41) is prepended.
// For 21-byte input, the bytes are copied as-is.
// For any other length, a copy of the input bytes is returned without validation.
func BytesToAddress(b []byte) Address {
	switch len(b) {
	case AddressLength - 1:
		addr := make([]byte, AddressLength)
		addr[0] = TronBytePrefix
		copy(addr[1:], b)
		return addr
	default:
		result := make([]byte, len(b))
		copy(result, b)
		return result
	}
}

// EthAddressToAddress converts a 20-byte Ethereum address to a 21-byte TRON address.
// It prepends the TRON mainnet prefix (0x41).
// Returns an error if the input is not exactly 20 bytes.
func EthAddressToAddress(ethAddr []byte) (Address, error) {
	if len(ethAddr) != AddressLength-1 {
		return nil, fmt.Errorf("got %d, want %d: %w", len(ethAddr), AddressLength-1, ErrInvalidAddressLength)
	}
	addr := make([]byte, AddressLength)
	addr[0] = TronBytePrefix
	copy(addr[1:], ethAddr)
	return addr, nil
}

// Scan implements the [database/sql.Scanner] interface for reading addresses from database columns.
func (a *Address) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("can't scan %T into Address", src)
	}
	if len(srcB) != AddressLength {
		return fmt.Errorf("can't scan []byte of len %d into Address, want %d", len(srcB), AddressLength)
	}
	buf := make([]byte, len(srcB))
	copy(buf, srcB)
	*a = Address(buf)
	return nil
}

// Value implements the [database/sql/driver.Valuer] interface for storing addresses in database columns.
func (a Address) Value() (driver.Value, error) {
	return []byte(a), nil
}

// IsValid checks if the address is a valid TRON address with checksum validation.
func (a Address) IsValid() bool {
	if len(a) != AddressLength {
		return false
	}
	if a[0] != TronBytePrefix {
		return false
	}
	// Validate Base58Check round-trip
	encoded := common.EncodeCheck(a)
	_, err := common.DecodeCheck(encoded)
	return err == nil
}
