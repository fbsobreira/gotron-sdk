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

	address := flag.String("address", "",
		"address: <account address>")

	flag.Parse()

	if (strings.EqualFold("", *address) && len(*address) == 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./get-account-net -grpcAddress localhost" +
			":50051 -address <account address>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	accountNet := client.GetAccountNet(*address)

	fmt.Printf("account net: %v\n", accountNet)
}
