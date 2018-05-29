package main

import (
	"flag"
	"fmt"
	"github.com/tronprotocol/go-client-api/common/hexutil"
	"github.com/tronprotocol/go-client-api/service"
	"github.com/tronprotocol/go-client-api/util"
	"log"
	"strings"
)

func main() {
	address := flag.String("address", "",
		"gRPC address: localhost:50051")

	flag.Parse()

	if strings.EqualFold("", *address) && len(*address) == 0 {
		log.Fatalln("go run client -address localhost:50051")
	}

	client := service.NewGrpcClient(*address)
	client.Start()
	defer client.Conn.Close()

	witnesses := client.ListWitnesses()

	for i, v := range witnesses.Witnesses {
		addr := hexutil.Encode(v.Address)
		u := v.Url
		totalProduced := v.TotalProduced
		totalMissed := v.TotalMissed
		latestBlockNum := v.LatestBlockNum
		latestSlotNum := v.LatestSlotNum
		isJobs := v.IsJobs
		fmt.Printf("index: %d, witness: address: %s, url: %s, "+
			"total produced: %d, total missed: %d, latest block num: %d, "+
			"latest slot num: %d, is jobs: %v\n", i,
			addr, u,
			totalProduced, totalMissed, latestBlockNum, latestSlotNum, isJobs)
	}

	nodes := client.ListNodes()

	for i, v := range nodes.Nodes {
		host := string(v.Address.Host)
		port := v.Address.Port
		fmt.Printf("index: %d, node: host: %v, port: %d\n", i, host, port)
	}

	account := client.GetAccount("A099357684BC659F5166046B56C95A0E99F1265CBD")

	fmt.Printf("account: type: %s, address: %s, balance: %d\n", account.Type,
		hexutil.Encode(account.Address), account.Balance)

	block := client.GetNowBlock()

	blockHash := util.GetBlockHash(*block)

	fmt.Printf("now block: block number: %v, hash: %v\n",
		block.BlockHeader.RawData.Number, hexutil.Encode(blockHash))
}
