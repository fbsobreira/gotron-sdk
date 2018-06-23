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

func GetAssetIssueByName(name string) AssetIssueContract {
	grpcAssetIssue := global.TronClient.GetAssetIssueByName(name)

	var assetIssueContract AssetIssueContract

	if grpcAssetIssue == nil {
		return assetIssueContract
	}

	assetIssueContract.OwnerAddress = base58.EncodeCheck(grpcAssetIssue.OwnerAddress)
	assetIssueContract.Name = string(grpcAssetIssue.Name)
	assetIssueContract.Abbr = string(grpcAssetIssue.Abbr)
	assetIssueContract.TotalSupply = grpcAssetIssue.TotalSupply

	assetIssueContract.FrozenSupply = make([]FrozenSupply, 0)
	for _, f := range grpcAssetIssue.FrozenSupply {
		var frozenSupply FrozenSupply
		frozenSupply.FrozenAmount = f.FrozenAmount
		frozenSupply.FrozenDays = f.FrozenDays
		assetIssueContract.FrozenSupply = append(assetIssueContract.
			FrozenSupply, frozenSupply)
	}

	assetIssueContract.TrxNum = grpcAssetIssue.TrxNum
	assetIssueContract.Num = grpcAssetIssue.Num
	assetIssueContract.StartTime = grpcAssetIssue.StartTime
	assetIssueContract.EndTime = grpcAssetIssue.EndTime
	assetIssueContract.VoteScore = grpcAssetIssue.VoteScore
	assetIssueContract.Description = string(grpcAssetIssue.Description)
	assetIssueContract.Url = string(grpcAssetIssue.Url)
	assetIssueContract.FreeAssetNetLimit = grpcAssetIssue.FreeAssetNetLimit
	assetIssueContract.PublicFreeAssetNetLimit = grpcAssetIssue.PublicFreeAssetNetLimit
	assetIssueContract.PublicFreeAssetNetUsage = grpcAssetIssue.PublicFreeAssetNetUsage
	assetIssueContract.PublicLatestFreeNetTime = grpcAssetIssue.PublicLatestFreeNetTime

	return assetIssueContract
}

func GetAssetIssueList() AssetIssueList {
	grpcAssetIssueList := global.TronClient.GetAssetIssueList()

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
