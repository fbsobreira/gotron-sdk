package models

import (
	"github.com/sasaxie/go-client-api/common/base58"
	"github.com/sasaxie/go-client-api/common/global"
	"github.com/sasaxie/go-client-api/common/hexutil"
)

type Witness struct {
	Address        string
	VoteCount      int64
	PubKey         string
	Url            string
	TotalProduced  int64
	TotalMissed    int64
	LatestBlockNum int64
	LatestSlotNum  int64
	IsJobs         bool
}

func GetWitnessList() []Witness {
	witnesses := make([]Witness, 0)

	grpcWitnesses := global.TronClient.ListWitnesses()

	if grpcWitnesses == nil {
		return witnesses
	}

	for _, w := range grpcWitnesses.Witnesses {
		var witness Witness
		witness.Address = base58.EncodeCheck(w.Address)
		witness.VoteCount = w.VoteCount
		witness.PubKey = hexutil.Encode(w.PubKey)
		witness.Url = w.Url
		witness.TotalProduced = w.TotalProduced
		witness.TotalMissed = w.TotalMissed
		witness.LatestBlockNum = w.LatestBlockNum
		witness.LatestSlotNum = w.LatestSlotNum
		witness.IsJobs = w.IsJobs
		witnesses = append(witnesses, witness)
	}

	return witnesses
}
