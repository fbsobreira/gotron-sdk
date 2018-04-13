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

	witnesses := client.ListWitnesses()

	for i, v := range witnesses.Witnesses {
		addr := hexutil.Encode(v.Address)
		u := v.Url
		totalProduced := v.TotalProduced
		totalMissed := v.TotalMissed
		latestBlockNum := v.LatestBlockNum
		latestSlotNum := v.LatestSlotNum
		isJobs := v.IsJobs
		fmt.Printf("index: %d, witness: address: %s, url: %s, " +
			"total produced: %d, total missed: %d, latest block num: %d, " +
			"latest slot num: %d, is jobs: %v\n", i,
			addr, u,
			totalProduced, totalMissed, latestBlockNum, latestSlotNum, isJobs)
	}
}
