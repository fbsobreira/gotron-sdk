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
	address := flag.String("address", "18.182.51.36:50051",
		"gRPC address: localhost:50051")

	flag.Parse()

	if strings.EqualFold("", *address) && len(*address) == 0 {
		log.Fatalln("./get-now-block -address localhost:50051")
	}

	client := service.NewGrpcClient(*address)
	client.Start()
	defer client.Conn.Close()

	block := client.GetNowBlock()

	blockHash := util.GetBlockHash(*block)

	fmt.Printf("now block: block number: %v, hash: %v\n",
		block.BlockHeader.RawData.Number, hexutil.Encode(blockHash))
}
