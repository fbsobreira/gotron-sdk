package contract_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/contract"
)

const erc20ABIBench = `[
	{
		"constant": true,
		"inputs": [{"name": "_owner", "type": "address"}],
		"name": "balanceOf",
		"outputs": [{"name": "balance", "type": "uint256"}],
		"type": "function",
		"stateMutability": "view"
	},
	{
		"constant": false,
		"inputs": [
			{"name": "_to", "type": "address"},
			{"name": "_value", "type": "uint256"}
		],
		"name": "transfer",
		"outputs": [{"name": "", "type": "bool"}],
		"type": "function",
		"stateMutability": "nonpayable"
	},
	{
		"inputs": [
			{"indexed": true, "name": "from", "type": "address"},
			{"indexed": true, "name": "to", "type": "address"},
			{"indexed": false, "name": "value", "type": "uint256"}
		],
		"name": "Transfer",
		"type": "event"
	}
]`

func BenchmarkJSONtoABI(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = contract.JSONtoABI(erc20ABIBench)
	}
}
