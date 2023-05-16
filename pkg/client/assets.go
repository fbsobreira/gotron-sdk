package client

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// GetAssetIssueByAccount list asset issued by account
func (g *GrpcClient) GetAssetIssueByAccount(address string) (*api.AssetIssueList, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.GetAssetIssueByAccount(ctx, account)
}

// GetAssetIssueByName list asset issued by name
func (g *GrpcClient) GetAssetIssueByName(name string) (*core.AssetIssueContract, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.GetAssetIssueByName(ctx, GetMessageBytes([]byte(name)))
}

// GetAssetIssueByID list asset issued by ID
func (g *GrpcClient) GetAssetIssueByID(tokenID string) (*core.AssetIssueContract, error) {
	bn := new(big.Int).SetBytes([]byte(tokenID))

	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.GetAssetIssueById(ctx, GetMessageBytes(bn.Bytes()))
}

// GetAssetIssueList list all TRC10
func (g *GrpcClient) GetAssetIssueList(page int64, limit ...int) (*api.AssetIssueList, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	if page == -1 {
		return g.Client.GetAssetIssueList(ctx, new(api.EmptyMessage))
	}

	useLimit := int64(10)
	if len(limit) == 1 {
		useLimit = int64(limit[0])
	}
	return g.Client.GetPaginatedAssetIssueList(ctx, GetPaginatedMessage(page*useLimit, useLimit))
}

// AssetIssue create a new asset TRC10
func (g *GrpcClient) AssetIssue(from, name, description, abbr, urlStr string,
	precision int32, totalSupply, startTime, endTime, FreeAssetNetLimit, PublicFreeAssetNetLimit int64,
	trxNum, icoNum, voteScore int32, frozenSupply map[string]string) (*api.TransactionExtention, error) {
	var err error

	contract := &core.AssetIssueContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	contract.Name = []byte(name)
	contract.Abbr = []byte(abbr)
	if precision < 0 || precision > 6 {
		return nil, fmt.Errorf("create asset issue error: precision < 0 || precision > 6")
	}
	contract.Precision = precision
	if totalSupply <= 0 {
		return nil, fmt.Errorf("create asset issue error: total supply <= 0")
	}
	contract.TotalSupply = totalSupply
	if trxNum <= 0 {
		return nil, fmt.Errorf("create asset issue error: trxNum <= 0")
	}
	contract.TrxNum = trxNum

	if icoNum <= 0 {
		return nil, fmt.Errorf("create asset issue error: num <= 0")
	}
	contract.Num = icoNum

	now := time.Now().UnixNano() / 1000000
	if startTime <= now {
		return nil, fmt.Errorf("create asset issue error: start time <= current time")
	}
	contract.StartTime = startTime

	if endTime <= startTime {
		return nil, fmt.Errorf("create asset issue error: end time <= start time")
	}
	contract.EndTime = endTime

	if FreeAssetNetLimit < 0 {
		return nil, fmt.Errorf("create asset issue error: free asset net limit < 0")
	}
	contract.FreeAssetNetLimit = FreeAssetNetLimit

	if PublicFreeAssetNetLimit < 0 {
		return nil, fmt.Errorf("create asset issue error: public free asset net limit < 0")
	}
	contract.PublicFreeAssetNetLimit = PublicFreeAssetNetLimit

	contract.VoteScore = voteScore
	contract.Description = []byte(description)
	contract.Url = []byte(urlStr)

	for key, value := range frozenSupply {
		amount, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("create asset issue: convert error: %v", err)
		}
		days, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("create asset issue error: convert error: %v", err)
		}
		assetIssueContractFrozenSupply := new(core.
			AssetIssueContract_FrozenSupply)
		assetIssueContractFrozenSupply.FrozenAmount = amount
		assetIssueContractFrozenSupply.FrozenDays = days
		// add supply to contract
		contract.FrozenSupply = append(contract.
			FrozenSupply, assetIssueContractFrozenSupply)
	}

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.CreateAssetIssue2(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}

// UpdateAssetIssue information
func (g *GrpcClient) UpdateAssetIssue(from, description, urlStr string,
	newLimit, newPublicLimit int64) (*api.TransactionExtention, error) {
	var err error

	contract := &core.UpdateAssetContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	contract.Description = []byte(description)
	contract.Url = []byte(urlStr)
	contract.NewLimit = newLimit
	contract.NewPublicLimit = newPublicLimit

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.UpdateAsset2(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}

// TransferAsset from to  base58 address
func (g *GrpcClient) TransferAsset(from, toAddress,
	assetName string, amount int64) (*api.TransactionExtention, error) {
	var err error
	contract := &core.TransferAssetContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	if contract.ToAddress, err = common.DecodeCheck(toAddress); err != nil {
		return nil, err
	}

	contract.AssetName = []byte(assetName)
	contract.Amount = amount

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.TransferAsset2(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}

// ParticipateAssetIssue TRC10 ICO
func (g *GrpcClient) ParticipateAssetIssue(from, issuerAddress,
	tokenID string, amount int64) (*api.TransactionExtention, error) {
	var err error
	contract := &core.ParticipateAssetIssueContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	if contract.ToAddress, err = common.DecodeCheck(issuerAddress); err != nil {
		return nil, err
	}

	contract.AssetName = []byte(tokenID)
	contract.Amount = amount

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.ParticipateAssetIssue2(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}

// UnfreezeAsset from owner
func (g *GrpcClient) UnfreezeAsset(from string) (*api.TransactionExtention, error) {
	var err error

	contract := &core.UnfreezeAssetContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.UnfreezeAsset2(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}
