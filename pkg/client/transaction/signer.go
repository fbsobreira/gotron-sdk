package transaction

import (
	"crypto/ecdsa"
	"crypto/sha256"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	proto "google.golang.org/protobuf/proto"
)

// SignerImpl identifies the signing backend used for transactions.
type SignerImpl int

const (
	// Software signs transactions using an in-process private key.
	Software SignerImpl = iota
	// Ledger signs transactions using a connected Ledger hardware wallet.
	Ledger
)

// SignTransaction signs a transaction using a btcec private key.
func SignTransaction(tx *core.Transaction, signer *btcec.PrivateKey) (*core.Transaction, error) {
	return SignTransactionECDSA(tx, signer.ToECDSA())
}

// SignTransactionECDSA signs a transaction using an ECDSA private key.
func SignTransactionECDSA(tx *core.Transaction, signer *ecdsa.PrivateKey) (*core.Transaction, error) {
	// Ensure the private key uses go-ethereum's secp256k1 curve.
	// Keys from btcec.ToECDSA() use a different curve instance that fails
	// validation in go-ethereum's non-CGO Sign() (e.g., on Windows).
	signer, err := crypto.ToECDSA(crypto.FromECDSA(signer))
	if err != nil {
		return nil, err
	}

	rawData, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		return nil, err
	}

	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)

	signature, err := crypto.Sign(hash, signer)
	if err != nil {
		return nil, err
	}
	tx.Signature = append(tx.Signature, signature)

	return tx, nil
}
