package contract

import (
	"github.com/fbsobreira/gotron/common/base58"
	"github.com/fbsobreira/gotron/common/global"
	"github.com/fbsobreira/gotron/core"
)

type AccountCreateContract struct {
	OwnerAddress   string           `json:"ownerAddress"`
	AccountAddress string           `json:"accountAddress"`
	Type           core.AccountType `json:"type"`
}

func CreateAccount(contract AccountCreateContract) (*core.Transaction,
	error) {

	grpcContract := new(core.AccountCreateContract)

	grpcContract.OwnerAddress = base58.DecodeCheck(contract.OwnerAddress)
	grpcContract.AccountAddress = base58.DecodeCheck(contract.AccountAddress)

	return global.TronClient.CreateAccountByContract(grpcContract)
}
