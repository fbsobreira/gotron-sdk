package models

import (
	"github.com/sasaxie/go-client-api/common/base58"
	"github.com/sasaxie/go-client-api/common/global"
	"github.com/sasaxie/go-client-api/common/hexutil"
)

type Account struct {
	AccountName                                string
	AccountType                                string
	Address                                    string
	Balance                                    int64
	Votes                                      []*Vote
	Asset                                      map[string]int64
	AssetV2                                    map[string]int64
	Frozen                                     []*Frozen
	NetUsage                                   int64
	AcquiredDelegatedFrozenBalanceForBandwidth int64
	DelegatedFrozenBalanceForBandwidth         int64
	CreateTime                                 int64
	LatestOprationTime                         int64
	Allowance                                  int64
	LatestWithdrawTime                         int64
	Code                                       string
	IsWitness                                  bool
	IsCommittee                                bool
	FrozenSupply                               []*Frozen
	AssetIssuedName                            string
	AssetIssuedID                              string
	LatestAssetOperationTime                   map[string]int64
	LatestAssetOperationTimeV2                 map[string]int64
	FreeNetUsage                               int64
	FreeAssetNetUsage                          map[string]int64
	FreeAssetNetUsageV2                        map[string]int64
	LatestConsumeTime                          int64
	LatestConsumeFreeTime                      int64
	AccountID                                  string
	AccountResource                            *AccountResource
}

type Vote struct {
	VoteAddress string
	VoteCount   int64
}

type Frozen struct {
	FrozenBalance int64
	ExpireTime    int64
}

type AccountResource struct {
	EnergyUsage                             int64
	FrozenBalanceForEnergy                  *Frozen
	LatestConsumeTimeForEnergy              int64
	AcquiredDelegatedFrozenBalanceForEnergy int64
	DelegatedFrozenBalanceForEnergy         int64
	StorageLimit                            int64
	StorageUsage                            int64
	LatestExchangeStorageTime               int64
}

func GetAccountByAddress(address string) (*Account, error) {
	grpcAccount := global.TronClient.GetAccount(address)

	resultAccount := new(Account)

	resultAccount.AccountName = string(grpcAccount.AccountName)
	resultAccount.AccountType = grpcAccount.Type.String()
	resultAccount.Address = base58.EncodeCheck(grpcAccount.Address)
	resultAccount.Balance = grpcAccount.Balance

	resultAccount.Votes = make([]*Vote, 0)
	for _, v := range grpcAccount.Votes {
		vote := new(Vote)
		vote.VoteAddress = base58.EncodeCheck(v.VoteAddress)
		vote.VoteCount = v.VoteCount
		resultAccount.Votes = append(resultAccount.Votes, vote)
	}

	resultAccount.Asset = make(map[string]int64)
	for k, v := range grpcAccount.Asset {
		resultAccount.Asset[k] = v
	}

	resultAccount.AssetV2 = make(map[string]int64)
	for k, v := range grpcAccount.AssetV2 {
		resultAccount.AssetV2[k] = v
	}

	resultAccount.Frozen = make([]*Frozen, 0)
	for _, v := range grpcAccount.Frozen {
		frozen := new(Frozen)
		frozen.FrozenBalance = v.FrozenBalance
		frozen.ExpireTime = v.ExpireTime
		resultAccount.Frozen = append(resultAccount.Frozen, frozen)
	}

	resultAccount.NetUsage = grpcAccount.NetUsage
	resultAccount.AcquiredDelegatedFrozenBalanceForBandwidth = grpcAccount.AcquiredDelegatedFrozenBalanceForBandwidth
	resultAccount.DelegatedFrozenBalanceForBandwidth = grpcAccount.DelegatedFrozenBalanceForBandwidth
	resultAccount.CreateTime = grpcAccount.CreateTime
	resultAccount.LatestOprationTime = grpcAccount.LatestOprationTime
	resultAccount.Allowance = grpcAccount.Allowance
	resultAccount.LatestWithdrawTime = grpcAccount.LatestWithdrawTime
	resultAccount.Code = hexutil.Encode(grpcAccount.Code)
	resultAccount.IsWitness = grpcAccount.IsWitness
	resultAccount.IsCommittee = grpcAccount.IsCommittee

	resultAccount.FrozenSupply = make([]*Frozen, 0)
	for _, v := range grpcAccount.FrozenSupply {
		frozen := new(Frozen)
		frozen.FrozenBalance = v.FrozenBalance
		frozen.ExpireTime = v.ExpireTime
		resultAccount.FrozenSupply = append(resultAccount.FrozenSupply, frozen)
	}

	resultAccount.AssetIssuedName = hexutil.Encode(grpcAccount.AssetIssuedName)
	resultAccount.AssetIssuedID = hexutil.Encode(grpcAccount.AssetIssued_ID)

	resultAccount.LatestAssetOperationTime = make(map[string]int64)
	for k, v := range grpcAccount.LatestAssetOperationTime {
		resultAccount.LatestAssetOperationTime[k] = v
	}

	resultAccount.LatestAssetOperationTimeV2 = make(map[string]int64)
	for k, v := range grpcAccount.LatestAssetOperationTimeV2 {
		resultAccount.LatestAssetOperationTimeV2[k] = v
	}

	resultAccount.FreeNetUsage = grpcAccount.FreeNetUsage

	resultAccount.FreeAssetNetUsage = make(map[string]int64)
	for k, v := range grpcAccount.FreeAssetNetUsage {
		resultAccount.FreeAssetNetUsage[k] = v
	}

	resultAccount.FreeAssetNetUsageV2 = make(map[string]int64)
	for k, v := range grpcAccount.FreeAssetNetUsageV2 {
		resultAccount.FreeAssetNetUsageV2[k] = v
	}

	resultAccount.LatestConsumeTime = grpcAccount.LatestConsumeTime
	resultAccount.LatestConsumeFreeTime = grpcAccount.LatestConsumeFreeTime
	resultAccount.AccountID = hexutil.Encode(grpcAccount.AccountId)

	// Account resource.
	if grpcAccount.AccountResource != nil {
		resultAccount.AccountResource = new(AccountResource)
		resultAccount.AccountResource.EnergyUsage = grpcAccount.AccountResource.EnergyUsage

		if grpcAccount.AccountResource.FrozenBalanceForEnergy != nil {
			resultAccount.AccountResource.FrozenBalanceForEnergy = new(Frozen)
			resultAccount.AccountResource.FrozenBalanceForEnergy.FrozenBalance =
				grpcAccount.AccountResource.FrozenBalanceForEnergy.FrozenBalance

			resultAccount.AccountResource.FrozenBalanceForEnergy.ExpireTime =
				grpcAccount.AccountResource.FrozenBalanceForEnergy.ExpireTime
		}

		resultAccount.AccountResource.LatestConsumeTimeForEnergy =
			grpcAccount.AccountResource.LatestConsumeTimeForEnergy

		resultAccount.AccountResource.AcquiredDelegatedFrozenBalanceForEnergy =
			grpcAccount.AccountResource.AcquiredDelegatedFrozenBalanceForEnergy

		resultAccount.AccountResource.DelegatedFrozenBalanceForEnergy =
			grpcAccount.AccountResource.DelegatedFrozenBalanceForEnergy

		resultAccount.AccountResource.StorageLimit = grpcAccount.AccountResource.StorageLimit

		resultAccount.AccountResource.StorageUsage = grpcAccount.AccountResource.StorageUsage

		resultAccount.AccountResource.LatestExchangeStorageTime =
			grpcAccount.AccountResource.LatestExchangeStorageTime
	}

	return resultAccount, nil
}
