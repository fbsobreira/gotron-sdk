package main

import (
	"github.com/tronprotocol/go-client-api/service"
	"fmt"
	"github.com/tronprotocol/go-client-api/common/hexutil"
)

const address = "47.91.216.69:50051"

func main() {
	client := service.NewGrpcClient(address)
	client.Start()
	defer client.Conn.Close()

	accounts := client.ListAccounts()

	for i, v := range accounts.Accounts {
		addr := hexutil.Encode(v.GetAddress())
		balance := v.Balance
		fmt.Printf("index: %d, account: address: %s, balance: %d\n", i, addr,
			balance)
	}
}
