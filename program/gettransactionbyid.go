package main

import (
	"flag"
	"fmt"
	"github.com/tronprotocol/go-client-api/service"
	"log"
	"strings"
)

func main() {
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	hash := flag.String("hash",
		"",
		"hash: <transaction hash>")

	flag.Parse()

	if (strings.EqualFold("", *hash) && len(*hash) == 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./get-transaction-by-id -grpcAddress localhost" +
			":50051 -hash 6c7e1104a824aaba0a8fba5497b35d7f2b5b3032ec833bd3bfcb5e9a938a4dc8")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	transaction := client.GetTransactionById(*hash)

	fmt.Printf("transaction: %v\n", transaction)
}
