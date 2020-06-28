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
	Address         string           `json:"address"`
	Type            string           `json:"type"`
	Name            string           `json:"name"`
	ID              string           `json:"id"`
	Balance         int64            `json:"balance"`
	Allowance       int64            `json:"allowance"`
	LastWithdraw    int64            `json:"lastWithdraw"`
	IsWitness       bool             `json:"isWitness"`
	IsElected       bool             `json:"isElected"`
	Assets          map[string]int64 `json:"assetList"`
	TronPower       int64            `json:"tronPower"`
	TronPowerUsed   int64            `json:"tronPowerUsed"`
	FrozenBalance   int64            `json:"frozenBalance"`
	FrozenResources []FrozenResource `json:"frozenList"`
	Votes           map[string]int64 `json:"voteList"`
	BWTotal         int64            `json:"bandwidthTotal"`
	BWUsed          int64            `json:"bandwidthUsed"`
	EnergyTotal     int64            `json:"energyTotal"`
	EnergyUsed      int64            `json:"energyUsed"`
}
