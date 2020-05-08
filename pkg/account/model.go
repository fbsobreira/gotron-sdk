package account

import (
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// FrozenResource by account
type FrozenResource struct {
	Type       core.ResourceCode
	Amount     int64
	DelegateTo string
	Expire     int64
}

// Account detailed view
type Account struct {
	Address         string
	Name            string
	ID              string
	Balance         int64
	Allowance       int64
	Assets          map[string]int64
	TronPower       int64
	TronPowerUsed   int64
	FrozenBalance   int64
	FrozenResources []FrozenResource
	Votes           map[string]int64
	BWTotal         int64
	BWUsed          int64
	EnergyTotal     int64
	EnergyUsed      int64
}
