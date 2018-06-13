package crypto

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/crypto"
)

func Sign(hash []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	return crypto.Sign(hash, privateKey)
}

func VerifySignature(publicKey, hash, signature []byte) bool {
	return crypto.VerifySignature(publicKey, hash, signature)
}
