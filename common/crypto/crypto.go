package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron/common/hexutil"
)

// Lengths of hashes and addresses in bytes.
const (
	// HashLength is the expected length of the hash
	HashLength = 32
	// AddressLength is the expected length of the address
	AddressLength = 21
	// AmountDecimalPoint TRX decimal point
	AmountDecimalPoint = 6
)

// AddressPrefix is the byte prefix of the address used in TRON addresses.
// It's supposed to be '0xa0' for testnet, and '0x41' for mainnet.
// But the Shasta mainteiners don't use the testnet params. So the default value is 41.
// You may change it directly, or use the SetAddressPrefix/UseMainnet/UseTestnet methods.
var AddressPrefix = byte(0x41)

// AddressPrefixHex address in hex string.
var AddressPrefixHex = hexutil.ToHex([]byte{AddressPrefix})

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

// BytesToHash sets b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// BigToHash sets byte representation of b to hash.
// If b is larger than len(h), b will be cropped from the left.
func BigToHash(b *big.Int) Hash { return BytesToHash(b.Bytes()) }

// HexToHash sets byte representation of s to hash.
// If b is larger than len(h), b will be cropped from the left.
func HexToHash(s string) Hash { return BytesToHash(hexutil.FromHex(s)) }

// Bytes gets the byte representation of the underlying hash.
func (h Hash) Bytes() []byte { return h[:] }

// Big converts a hash to a big integer.
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex converts a hash to a hex string.
func (h Hash) Hex() string { return hexutil.Encode(h[:]) }

// TerminalString implements log.TerminalStringer, formatting a string for console
// output during logging.
func (h Hash) TerminalString() string {
	return fmt.Sprintf("%x…%x", h[:3], h[29:])
}

// String implements the stringer interface and is used also by the logger when
// doing full logging into a file.
func (h Hash) String() string {
	return h.Hex()
}

// SetBytes sets the hash to the value of b.
// If b is larger than len(h), b will be cropped from the left.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// SetAddressPrefix sets the prefix to the provided byte.
func SetAddressPrefix(p byte) {
	AddressPrefix = p
	AddressPrefixHex = hexutil.ToHex([]byte{p})
}

// UseMainnet sets the address prefix used for the main net.
func UseMainnet() {
	SetAddressPrefix(0x41)
}

// UseTestnet sets the address prefix used for the test net.
func UseTestnet() {
	SetAddressPrefix(0xa0)
}

/////////// Address

type Address [AddressLength]byte

func (a Address) Bytes() []byte {
	return a[:]
}

func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// BigToAddress returns Address with byte values of b.
// If b is larger than len(h), b will be cropped from the left.
func BigToAddress(b *big.Int) Address { return BytesToAddress(b.Bytes()) }

// HexToAddress returns Address with byte values of s.
// If s is larger than len(h), s will be cropped from the left.
func HexToAddress(s string) Address { return BytesToAddress(hexutil.FromHex(s)) }

// String implements fmt.Stringer.
func (a Address) String() string {
	return hexutil.ToHex(a.Bytes())
}

func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(crypto.S256(), rand.Reader)
}

func GetPrivateKeyByHexString(privateKeyHexString string) (*ecdsa.PrivateKey,
	error) {
	return crypto.HexToECDSA(privateKeyHexString)
}

func PubkeyToAddress(p ecdsa.PublicKey) Address {
	address := crypto.PubkeyToAddress(p)

	addressTron := make([]byte, AddressLength)

	addressTron = append(addressTron, AddressPrefix)
	addressTron = append(addressTron, address.Bytes()...)

	return BytesToAddress(addressTron)
}
