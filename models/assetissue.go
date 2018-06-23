package models

import (
	"github.com/sasaxie/go-client-api/common/base58"
	"github.com/sasaxie/go-client-api/common/global"
)

type AssetIssueList struct {
	AssetIssue []AssetIssueContract
}

type AssetIssueContract struct {
	OwnerAddress            string
	Name                    string
	Abbr                    string
	TotalSupply             int64
	FrozenSupply            []FrozenSupply
	TrxNum                  int32
	Num                     int32
	StartTime               int64
	EndTime                 int64
	VoteScore               int32
	Description             string
	Url                     string
	FreeAssetNetLimit       int64
	PublicFreeAssetNetLimit int64
	PublicFreeAssetNetUsage int64
	PublicLatestFreeNetTime int64
}

type FrozenSupply struct {
	FrozenAmount int64
	FrozenDays   int64
}

func GetAssetIssueAccount(address string) AssetIssueList {
	grpcAssetIssueList := global.TronClient.GetAssetIssueByAccount(address)

	var resultAssetIssueList AssetIssueList

	if grpcAssetIssueList == nil {
		return resultAssetIssueList
	}

	resultAssetIssueList.AssetIssue = make([]AssetIssueContract, 0)
	for _, a := range grpcAssetIssueList.AssetIssue {
		var assetIssueContract AssetIssueContract
		assetIssueContract.OwnerAddress = base58.EncodeCheck(a.OwnerAddress)
		assetIssueContract.Name = string(a.Name)
		assetIssueContract.Abbr = string(a.Abbr)
		assetIssueContract.TotalSupply = a.TotalSupply

		assetIssueContract.FrozenSupply = make([]FrozenSupply, 0)
		for _, f := range a.FrozenSupply {
			var frozenSupply FrozenSupply
			frozenSupply.FrozenAmount = f.FrozenAmount
			frozenSupply.FrozenDays = f.FrozenDays
			assetIssueContract.FrozenSupply = append(assetIssueContract.
				FrozenSupply, frozenSupply)
		}

		assetIssueContract.TrxNum = a.TrxNum
		assetIssueContract.Num = a.Num
		assetIssueContract.StartTime = a.StartTime
		assetIssueContract.EndTime = a.EndTime
		assetIssueContract.VoteScore = a.VoteScore
		assetIssueContract.Description = string(a.Description)
		assetIssueContract.Url = string(a.Url)
		assetIssueContract.FreeAssetNetLimit = a.FreeAssetNetLimit
		assetIssueContract.PublicFreeAssetNetLimit = a.PublicFreeAssetNetLimit
		assetIssueContract.PublicFreeAssetNetUsage = a.PublicFreeAssetNetUsage
		assetIssueContract.PublicLatestFreeNetTime = a.PublicLatestFreeNetTime

		resultAssetIssueList.AssetIssue = append(resultAssetIssueList.
			AssetIssue, assetIssueContract)
	}

	return resultAssetIssueList
}
