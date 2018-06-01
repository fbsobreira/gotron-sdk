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
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: localhost:50051")

	flag.Parse()

	if strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0 {
		log.Fatalln("go run client -address localhost:50051")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	account := client.GetAccount("A099357684BC659F5166046B56C95A0E99F1265CBD")

	fmt.Printf("account: type: %s, address: %s, balance: %d\n", account.Type,
		hexutil.Encode(account.Address), account.Balance)

	block := client.GetNowBlock()

	blockHash := util.GetBlockHash(*block)

	fmt.Printf("now block: block number: %v, hash: %v\n",
		block.BlockHeader.RawData.Number, hexutil.Encode(blockHash))
}
