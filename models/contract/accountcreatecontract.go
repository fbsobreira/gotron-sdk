package contract

import (
	"github.com/fbsobreira/go-client-api/common/base58"
	"github.com/fbsobreira/go-client-api/common/global"
	"github.com/fbsobreira/go-client-api/core"
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
