package client

import (
	"bytes"
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
)

// GetAccount from BASE58 address
func (g *GrpcClient) GetAccount(addr string) (*core.Account, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	acc, err := g.Client.GetAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(acc.Address, account.Address) {
		return nil, fmt.Errorf("account not found")
	}
	return acc, nil
}

// GetAccountNet return account resources from BASE58 address
func (g *GrpcClient) GetAccountNet(addr string) (*api.AccountNetMessage, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	return g.Client.GetAccountNet(ctx, account)
}

// CreateAccount activate tron account
func (g *GrpcClient) CreateAccount(from, addr string) (*api.TransactionExtention, error) {
	var err error

	contract := &core.AccountCreateContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	if contract.AccountAddress, err = common.DecodeCheck(addr); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	tx, err := g.Client.CreateAccount2(ctx, contract)
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

// UpdateAccount change account name
func (g *GrpcClient) UpdateAccount(from, accountName string) (*api.TransactionExtention, error) {
	var err error
	contract := &core.AccountUpdateContract{}
	contract.AccountName = []byte(accountName)
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	tx, err := g.Client.UpdateAccount2(ctx, contract)
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

// GetAccountDetailed from BASE58 address
func (g *GrpcClient) GetAccountDetailed(addr string) (*account.Account, error) {

	acc, err := g.GetAccount(addr)
	if err != nil {
		return nil, err
	}

	accR, err := g.GetAccountResource(addr)
	if err != nil {
		return nil, err
	}

	accDeleagated, err := g.GetDelegatedResources(addr)
	if err != nil {
		return nil, err
	}

	// SUM Total freeze
	totalFrozen := int64(0)
	frozenList := make([]account.FrozenResource, 0)
	if acc.GetAccountResource().GetFrozenBalanceForEnergy().GetFrozenBalance() > 0 {
		frozenList = append(frozenList, account.FrozenResource{
			Type:       core.ResourceCode_ENERGY,
			Amount:     acc.GetAccountResource().GetFrozenBalanceForEnergy().GetFrozenBalance(),
			Expire:     acc.GetAccountResource().GetFrozenBalanceForEnergy().GetExpireTime(),
			DelegateTo: "",
		})
		totalFrozen += acc.GetAccountResource().GetFrozenBalanceForEnergy().GetFrozenBalance()
	}
	for _, f := range acc.Frozen {
		frozenList = append(frozenList, account.FrozenResource{
			Type:       core.ResourceCode_BANDWIDTH,
			Amount:     f.GetFrozenBalance(),
			Expire:     f.GetExpireTime(),
			DelegateTo: "",
		})
		totalFrozen += f.GetFrozenBalance()
	}

	// Fill Delegated
	for _, delegated := range accDeleagated {
		for _, d := range delegated.GetDelegatedResource() {
			if d.GetFrozenBalanceForBandwidth() > 0 {
				frozenList = append(frozenList, account.FrozenResource{
					Type:       core.ResourceCode_BANDWIDTH,
					Amount:     d.GetFrozenBalanceForBandwidth(),
					Expire:     d.GetExpireTimeForBandwidth(),
					DelegateTo: address.Address(d.GetTo()).String(),
				})
				totalFrozen += d.GetFrozenBalanceForBandwidth()
			}
			if d.GetFrozenBalanceForEnergy() > 0 {
				frozenList = append(frozenList, account.FrozenResource{
					Type:       core.ResourceCode_ENERGY,
					Amount:     d.GetFrozenBalanceForEnergy(),
					Expire:     d.GetExpireTimeForEnergy(),
					DelegateTo: address.Address(d.GetTo()).String(),
				})
				totalFrozen += d.GetFrozenBalanceForEnergy()
			}
		}
	}

	voteList := make(map[string]int64)

	totalVotes := int64(0)
	for _, vote := range acc.GetVotes() {
		voteList[address.Address(vote.GetVoteAddress()).String()] = vote.GetVoteCount()
		totalVotes += vote.GetVoteCount()
	}

	accDet := &account.Account{
		Address:         address.Address(acc.GetAddress()).String(),
		Name:            string(acc.GetAccountName()),
		ID:              string(acc.GetAccountId()),
		Balance:         acc.GetBalance(),
		Allowance:       acc.GetAllowance(),
		LastWithdraw:    acc.LatestWithdrawTime,
		IsWitness:       acc.IsWitness,
		IsElected:       acc.IsWitness,
		Assets:          acc.GetAssetV2(),
		TronPower:       totalFrozen / 1000000,
		TronPowerUsed:   totalVotes,
		FrozenBalance:   totalFrozen,
		FrozenResources: frozenList,
		Votes:           voteList,
		BWTotal:         accR.GetFreeNetLimit() + accR.GetNetLimit(),
		BWUsed:          accR.GetFreeNetUsed() + accR.GetNetUsed(),
		EnergyTotal:     accR.GetEnergyLimit(),
		EnergyUsed:      accR.GetEnergyUsed(),
	}

	return accDet, nil
}

// WithdrawBalance rewards from account
func (g *GrpcClient) WithdrawBalance(from string) (*api.TransactionExtention, error) {
	var err error
	contract := &core.WithdrawBalanceContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	tx, err := g.Client.WithdrawBalance2(ctx, contract)
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
