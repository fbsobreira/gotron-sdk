package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"

	"github.com/ethereum/go-ethereum/crypto"
)

const AddressLength = 21

// AddressPrefix is the byte prefix of the address used in TRON addresses.
// It's supposed to be '0xa0' for testnet, and '0x41' for mainnet.
// But the Shasta mainteiners don't use the testnet params. So the default value is 41.
// You may change it directly, or use the SetAddressPrefix/UseMainnet/UseTestnet methods.
var AddressPrefix = byte(0x41)

// SetAddressPrefix sets the prefix to the provided byte.
func SetAddressPrefix(p byte) {
	AddressPrefix = p
}

// UseMainnet sets the address prefix used for the main net.
func UseMainnet() {
	SetAddressPrefix(0x41)
}

// UseTestnet sets the address prefix used for the test net.
func UseTestnet() {
	SetAddressPrefix(0xa0)
}

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
