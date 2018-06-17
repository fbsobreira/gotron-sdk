package main

import (
	"flag"
	"fmt"
	"github.com/sasaxie/go-client-api/common/crypto"
	"github.com/sasaxie/go-client-api/service"
	"log"
	"strings"
)

func main() {
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	ownerPrivateKey := flag.String("ownerPrivateKey", "",
		"ownerPrivateKey: <account private key>")

	address := flag.String("toAddress", "",
		"toAddress: <to account address>")

	amount := flag.Int64("amount", 0,
		"amount: <transfer amount>")

	flag.Parse()

	if (*amount <= 0) || (strings.EqualFold("", *ownerPrivateKey) && len(
		*ownerPrivateKey) == 0) || (strings.
		EqualFold("", *address) && len(*address) == 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./transfer -grpcAddress localhost" +
			":50051 -ownerPrivateKey <your private key> -toAddress" +
			" <to account address> -amount <transfer amount>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	key, err := crypto.GetPrivateKeyByHexString(*ownerPrivateKey)

	if err != nil {
		log.Fatalf("get private key by hex string error: %v", err)
	}

	result := client.Transfer(key, *address, *amount)

	fmt.Printf("result: %v\n", result)
}
