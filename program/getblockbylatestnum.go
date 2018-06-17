package main

import (
	"flag"
	"fmt"
	"github.com/sasaxie/go-client-api/service"
	"log"
	"strings"
)

func main() {
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	num := flag.Int64("number", 0,
		"number: <block number>")

	flag.Parse()

	if (*num < 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./get-block-by-latest-num -grpcAddress localhost" +
			":50051 -number <block number>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	blockList := client.GetBlockByLatestNum(*num)

	fmt.Printf("block list: %v\n", blockList)
}
