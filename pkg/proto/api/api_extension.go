package api

import (
	"crypto/sha256"
	"fmt"

	core "github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

func (x *TransactionExtention) SetData(memo string) error {
	if x == nil {
		return fmt.Errorf("TransactionExtention is nil")
	}

	// check if the memo is already set
	if x.Transaction == nil {
		x.Transaction = &core.Transaction{}
	}

	if x.Transaction.RawData == nil {
		x.Transaction.RawData = &core.TransactionRaw{}
	}

	if len(x.Transaction.RawData.Data) != 0 {
		return fmt.Errorf("memo is already set")
	}

	x.Transaction.RawData.Data = []byte(memo)

	return x.UpdateHash()
}

func (x *TransactionExtention) UpdateHash() error {
	if x == nil || x.Transaction == nil || x.Transaction.RawData == nil {
		return fmt.Errorf("TransactionExtention or Transaction or RawData is nil")
	}

	rawData, err := proto.Marshal(x.Transaction.GetRawData())
	if err != nil {
		return err
	}

	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)
	x.Txid = hash

	return nil
}
