package main

import (
	"flag"
	"fmt"
	"github.com/fbsobreira/gotron/common/hexutil"
	"github.com/fbsobreira/gotron/service"
	"log"
	"strings"
)

func main() {
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	address := flag.String("address", "",
		"address: <account address>")

	flag.Parse()

	if (strings.EqualFold("", *address) && len(*address) == 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./get-account-by-address -grpcAddress localhost" +
			":50051 -address <account address>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	account := client.GetAccount(*address)

	fmt.Printf("account: type: %s, address: %s, balance: %d\n", account.Type,
		hexutil.Encode(account.Address), account.Balance)
}
