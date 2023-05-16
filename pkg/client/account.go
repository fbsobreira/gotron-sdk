package client

import (
	"bytes"
	"fmt"
	"math/big"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// GetAccount from BASE58 address
func (g *GrpcClient) GetAccount(addr string) (*core.Account, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
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

// GetRewardsInfo from BASE58 address
func (g *GrpcClient) GetRewardsInfo(addr string) (int64, error) {
	addrBytes, err := common.DecodeCheck(addr)
	if err != nil {
		return 0, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	rewards, err := g.Client.GetRewardInfo(ctx, GetMessageBytes(addrBytes))
	if err != nil {
		return 0, err
	}
	return rewards.Num, nil
}

// GetAccountNet return account resources from BASE58 address
func (g *GrpcClient) GetAccountNet(addr string) (*api.AccountNetMessage, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
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
	ctx, cancel := g.getContext()
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

	ctx, cancel := g.getContext()
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

	accDeleagatedV2, err := g.GetDelegatedResourcesV2(addr)
	if err != nil {
		return nil, err
	}

	accUnfreezeLeft, err := g.GetAvailableUnfreezeCount(addr)
	if err != nil {
		return nil, err
	}

	rewards, err := g.GetRewardsInfo(addr)
	if err != nil {
		return nil, err
	}

	withdrawableAmount, err := g.GetCanWithdrawUnfreezeAmount(addr, time.Now().UnixMilli())
	if err != nil {
		return nil, err
	}

	maxCanDelegateBandwidth, err := g.GetCanDelegatedMaxSize(addr, int32(core.ResourceCode_BANDWIDTH))
	if err != nil {
		return nil, err
	}
	maxCanDelegateEnergy, err := g.GetCanDelegatedMaxSize(addr, int32(core.ResourceCode_ENERGY))
	if err != nil {
		return nil, err
	}

	// SUM Total freeze V1
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

	// Fill Delegated V1
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

	// SUM Total freeze V2
	totalFrozenV2 := int64(0)
	frozenListV2 := make([]account.FrozenResource, 0)

	// Energy Delegated
	totalFrozenV2 += acc.GetAccountResource().GetDelegatedFrozenV2BalanceForEnergy()
	// Bandwidth Delegated
	totalFrozenV2 += acc.GetDelegatedFrozenV2BalanceForBandwidth()

	// Frozen not delegated
	for _, f := range acc.FrozenV2 {
		frozenListV2 = append(frozenListV2, account.FrozenResource{
			Type:       f.GetType(),
			Amount:     f.GetAmount(),
			DelegateTo: "",
		})
		totalFrozenV2 += f.GetAmount()
	}

	// Fill Delegated V2
	for _, delegated := range accDeleagatedV2 {
		for _, d := range delegated.GetDelegatedResource() {
			if d.GetFrozenBalanceForBandwidth() > 0 {
				frozenListV2 = append(frozenListV2, account.FrozenResource{
					Type:       core.ResourceCode_BANDWIDTH,
					Amount:     d.GetFrozenBalanceForBandwidth(),
					Expire:     d.GetExpireTimeForBandwidth(),
					DelegateTo: address.Address(d.GetTo()).String(),
				})
			}
			if d.GetFrozenBalanceForEnergy() > 0 {
				frozenListV2 = append(frozenListV2, account.FrozenResource{
					Type:       core.ResourceCode_ENERGY,
					Amount:     d.GetFrozenBalanceForEnergy(),
					Expire:     d.GetExpireTimeForEnergy(),
					DelegateTo: address.Address(d.GetTo()).String(),
				})
			}
		}
	}

	unfrozenListV2 := make([]account.UnfrozenResource, 0)
	for _, uf := range acc.UnfrozenV2 {
		unfrozenListV2 = append(unfrozenListV2, account.UnfrozenResource{
			Type:   uf.GetType(),
			Amount: uf.GetUnfreezeAmount(),
			Expire: uf.GetUnfreezeExpireTime(),
		})
	}

	voteList := make(map[string]int64)

	totalVotes := int64(0)
	for _, vote := range acc.GetVotes() {
		voteList[address.Address(vote.GetVoteAddress()).String()] = vote.GetVoteCount()
		totalVotes += vote.GetVoteCount()
	}

	accDet := &account.Account{
		Address:                 address.Address(acc.GetAddress()).String(),
		Type:                    acc.Type.String(),
		Name:                    string(acc.GetAccountName()),
		ID:                      string(acc.GetAccountId()),
		Balance:                 acc.GetBalance(),
		Allowance:               acc.GetAllowance(),
		LastWithdraw:            acc.LatestWithdrawTime,
		IsWitness:               acc.IsWitness,
		IsElected:               acc.IsCommittee,
		Assets:                  acc.GetAssetV2(),
		TronPower:               (totalFrozen + totalFrozenV2) / 1000000,
		TronPowerUsed:           totalVotes,
		FrozenBalance:           totalFrozen,
		FrozenBalanceV2:         totalFrozenV2,
		FrozenResourcesV2:       frozenListV2,
		FrozenResources:         frozenList,
		Votes:                   voteList,
		BWTotal:                 accR.GetFreeNetLimit() + accR.GetNetLimit(),
		BWUsed:                  accR.GetFreeNetUsed() + accR.GetNetUsed(),
		EnergyTotal:             accR.GetEnergyLimit(),
		EnergyUsed:              accR.GetEnergyUsed(),
		Rewards:                 rewards,
		WithdrawableBalance:     withdrawableAmount.GetAmount(),
		UnfrozenResource:        unfrozenListV2,
		UnfreezeLeft:            accUnfreezeLeft.GetCount(),
		MaxCanDelegateBandwidth: maxCanDelegateBandwidth.GetMaxSize(),
		MaxCanDelegateEnergy:    maxCanDelegateEnergy.GetMaxSize(),
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

	ctx, cancel := g.getContext()
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

func makePermission(name string, pType core.Permission_PermissionType, id int32,
	threshold int64, operations map[string]bool, keys map[string]int64) (*core.Permission, error) {

	pKey := make([]*core.Key, 0)

	if len(keys) > 5 {
		return nil, fmt.Errorf("cant have more than 5 keys")
	}
	totalWeight := int64(0)
	for k, w := range keys {
		totalWeight += w
		addr, err := address.Base58ToAddress(k)
		if err != nil {
			return nil, fmt.Errorf("invalid address: %s", k)
		}
		pKey = append(pKey, &core.Key{
			Address: addr,
			Weight:  w,
		})
	}
	var bigOP *big.Int
	if operations != nil && len(operations) > 0 {
		bigOP = big.NewInt(0)
		for k, o := range operations {
			if o {
				// find k in contracts
				value, b := core.Transaction_Contract_ContractType_value[k]
				if !b {
					return nil, fmt.Errorf("permission not found: %s", k)
				}
				bigOP.SetBit(bigOP, int(value), 1)
			}
		}
	} else {
		bigOP = nil
	}

	if threshold > totalWeight {
		return nil, fmt.Errorf("invalid key/threshold size (%d/%d)", threshold, totalWeight)
	}
	var bOP []byte
	if bigOP != nil {
		bOP = make([]byte, 32)
		l := len(bigOP.Bytes()) - 1
		for i, b := range bigOP.Bytes() {
			bOP[l-i] = b
		}
	}

	return &core.Permission{
		Type:           pType,
		Id:             id,
		PermissionName: name,
		Threshold:      threshold,
		Operations:     bOP,
		Keys:           pKey,
	}, nil
}

// UpdateAccountPermission change account permission
func (g *GrpcClient) UpdateAccountPermission(from string, owner, witness map[string]interface{}, actives []map[string]interface{}) (*api.TransactionExtention, error) {

	if len(actives) > 8 {
		return nil, fmt.Errorf("cant have more than 8 active operations")
	}

	if owner == nil {
		return nil, fmt.Errorf("owner is manadory")
	}
	ownerPermission, err := makePermission(
		"owner",
		core.Permission_Owner,
		0,
		owner["threshold"].(int64),
		nil,
		owner["keys"].(map[string]int64),
	)
	if err != nil {
		return nil, err
	}
	contract := &core.AccountPermissionUpdateContract{
		Owner: ownerPermission,
	}

	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	if actives != nil {
		activesPermission := make([]*core.Permission, 0)
		for i, active := range actives {
			activeP, err := makePermission(
				active["name"].(string),
				core.Permission_Active,
				int32(2+i),
				active["threshold"].(int64),
				active["operations"].(map[string]bool),
				active["keys"].(map[string]int64),
			)
			if err != nil {
				return nil, err
			}
			activesPermission = append(activesPermission, activeP)
		}
		contract.Actives = activesPermission
	}

	if witness != nil {
		witnessPermission, err := makePermission(
			"witness",
			core.Permission_Witness,
			1,
			witness["threshold"].(int64),
			nil,
			witness["keys"].(map[string]int64),
		)
		if err != nil {
			return nil, err
		}
		if err != nil {
			return nil, err
		}
		contract.Witness = witnessPermission
	}

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.AccountPermissionUpdate(ctx, contract)
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
