package main

import (
	"flag"
	"fmt"
	"github.com/sasaxie/go-client-api/common/hexutil"
	"github.com/sasaxie/go-client-api/service"
	"github.com/sasaxie/go-client-api/util"
	"log"
	"strings"
)

func main() {
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	flag.Parse()

	if strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0 {
		log.Fatalln("./get-now-block -grpcAddress localhost:50051")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	block := client.GetNowBlock()

	blockHash := util.GetBlockHash(*block)

	fmt.Printf("now block: block number: %v, hash: %v\n",
		block.BlockHeader.RawData.Number, hexutil.Encode(blockHash))
}
