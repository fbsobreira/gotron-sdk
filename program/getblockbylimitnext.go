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

	start := flag.Int64("start", 0,
		"start: <start block number>")

	end := flag.Int64("end", 1,
		"end: <end block number>")

	flag.Parse()

	if (*start < 0) || (*end < 0) || (strings.EqualFold("",
		*grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./get-block-by-limit-next -grpcAddress localhost" +
			":50051 -start <start block number> -end <end block number>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	blockList := client.GetBlockByLimitNext(*start, *end)

	fmt.Printf("block list: %v\n", blockList)
}
