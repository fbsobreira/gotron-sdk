package crypto

import (
	"crypto/ecdsa"
	"crypto/rand"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/sasaxie/go-client-api/common/hexutil"
)

const AddressLength = 21
const AddressPrefix = "a0"

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

	addressPrefix, err := hexutil.Decode(AddressPrefix)
	if err != nil {
		log.Error(err.Error())
	}

	addressTron = append(addressTron, addressPrefix...)
	addressTron = append(addressTron, address.Bytes()...)

	return BytesToAddress(addressTron)
}
