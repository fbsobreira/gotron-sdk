package client

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/golang/protobuf/proto"
)

// GetAccount from BASE58 address
func (g *GrpcClient) GetAccount(address string) (*core.Account, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	return g.Client.GetAccount(ctx, account)
}

// GetAccountNet return account resources from BASE58 address
func (g *GrpcClient) GetAccountNet(address string) (*api.AccountNetMessage, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	return g.Client.GetAccountNet(ctx, account)
}

// GetAccountResource from BASE58 address
func (g *GrpcClient) GetAccountResource(address string) (*api.AccountResourceMessage, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), grpcTimeout)
	defer cancel()

	return g.Client.GetAccountResource(ctx, account)
}

// CreateAccount activate tron account
func (g *GrpcClient) CreateAccount(from, accountAddress string) (*api.TransactionExtention, error) {
	var err error

	contract := &core.AccountCreateContract{}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}
	if contract.AccountAddress, err = common.DecodeCheck(accountAddress); err != nil {
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
