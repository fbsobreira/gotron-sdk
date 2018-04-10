package main

import (
	"github.com/tronprotocol/go-client-api/service"
	"fmt"
)

const address = "47.91.216.69:50051"

func main() {
	client := service.NewGrpcClient(address)
	client.Start()

	accounts := client.ListAccounts()

	for i, v := range accounts.Accounts {
		fmt.Printf("index: %d, account: %v\n", i, v)
	}
}
