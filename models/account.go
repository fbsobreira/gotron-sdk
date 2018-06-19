package models

import (
	"github.com/sasaxie/go-client-api/common/base58"
	"github.com/sasaxie/go-client-api/common/global"
	"github.com/sasaxie/go-client-api/common/hexutil"
)

type Account struct {
	AccountName              string
	AccountType              string
	Address                  string
	Balance                  int64
	Votes                    []Vote
	Asset                    map[string]int64
	Frozen                   []Frozen
	NetUsage                 int64
	CreateTime               int64
	LatestOprationTime       int64
	Allowance                int64
	LatestWithdrawTime       int64
	Code                     string
	IsWitness                bool
	IsCommittee              bool
	FrozenSupply             []Frozen
	AssetIssuedName          string
	LatestAssetOperationTime map[string]int64
	FreeNetUsage             int64
	FreeAssetNetUsage        map[string]int64
	LatestConsumeTime        int64
	LatestConsumeFreeTime    int64
}

type Vote struct {
	VoteAddress string
	VoteCount   int64
}

type Frozen struct {
	FrozenBalance int64
	ExpireTime    int64
}

func GetAccountByAddress(address string) (*Account, error) {
	grpcAccount := global.TronClient.GetAccount(address)

	resultAccount := new(Account)

	resultAccount.AccountName = string(grpcAccount.AccountName)
	resultAccount.AccountType = grpcAccount.Type.String()
	resultAccount.Address = base58.EncodeCheck(grpcAccount.Address)
	resultAccount.Balance = grpcAccount.Balance

	resultAccount.Votes = make([]Vote, 0)
	for _, v := range grpcAccount.Votes {
		var vote Vote
		vote.VoteAddress = base58.EncodeCheck(v.VoteAddress)
		vote.VoteCount = v.VoteCount
		resultAccount.Votes = append(resultAccount.Votes, vote)
	}

	resultAccount.Asset = make(map[string]int64)
	for k, v := range grpcAccount.Asset {
		resultAccount.Asset[k] = v
	}

	resultAccount.Frozen = make([]Frozen, 0)
	for _, v := range grpcAccount.Frozen {
		var frozen Frozen
		frozen.FrozenBalance = v.FrozenBalance
		frozen.ExpireTime = v.ExpireTime
		resultAccount.Frozen = append(resultAccount.Frozen, frozen)
	}

	resultAccount.NetUsage = grpcAccount.NetUsage
	resultAccount.CreateTime = grpcAccount.CreateTime
	resultAccount.LatestOprationTime = grpcAccount.LatestOprationTime
	resultAccount.Allowance = grpcAccount.Allowance
	resultAccount.LatestWithdrawTime = grpcAccount.LatestWithdrawTime
	resultAccount.Code = hexutil.Encode(grpcAccount.Code)
	resultAccount.IsWitness = grpcAccount.IsWitness
	resultAccount.IsCommittee = grpcAccount.IsCommittee

	resultAccount.FrozenSupply = make([]Frozen, 0)
	for _, v := range grpcAccount.FrozenSupply {
		var frozen Frozen
		frozen.FrozenBalance = v.FrozenBalance
		frozen.ExpireTime = v.ExpireTime
		resultAccount.FrozenSupply = append(resultAccount.FrozenSupply, frozen)
	}

	resultAccount.AssetIssuedName = hexutil.Encode(grpcAccount.AssetIssuedName)

	resultAccount.LatestAssetOperationTime = make(map[string]int64)
	for k, v := range grpcAccount.LatestAssetOperationTime {
		resultAccount.LatestAssetOperationTime[k] = v
	}

	resultAccount.FreeNetUsage = grpcAccount.FreeNetUsage

	resultAccount.FreeAssetNetUsage = make(map[string]int64)
	for k, v := range grpcAccount.FreeAssetNetUsage {
		resultAccount.FreeAssetNetUsage[k] = v
	}

	resultAccount.LatestConsumeTime = grpcAccount.LatestConsumeTime
	resultAccount.LatestConsumeFreeTime = grpcAccount.LatestConsumeFreeTime

	return resultAccount, nil
}
