package util

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"github.com/fbsobreira/gotron/common/crypto"
	"github.com/fbsobreira/gotron/core"
	"github.com/golang/protobuf/proto"
	"log"
	"time"
)

func SignTransaction(transaction *core.Transaction, key *ecdsa.PrivateKey) {
	transaction.GetRawData().Timestamp = time.Now().UnixNano() / 1000000

	rawData, err := proto.Marshal(transaction.GetRawData())

	if err != nil {
		log.Fatalf("sign transaction error: %v", err)
	}

	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)

	contractList := transaction.GetRawData().GetContract()

	for range contractList {
		signature, err := crypto.Sign(hash, key)

		if err != nil {
			log.Fatalf("sign transaction error: %v", err)
		}

		transaction.Signature = append(transaction.Signature, signature)
	}
}
