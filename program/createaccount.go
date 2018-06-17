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

	address := flag.String("newAccountAddress", "",
		"newAccountAddress: <new account address>")

	flag.Parse()

	if (strings.EqualFold("", *ownerPrivateKey) && len(*ownerPrivateKey) == 0) || (strings.
		EqualFold("", *address) && len(*address) == 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./create-account -grpcAddress localhost" +
			":50051 -ownerPrivateKey <your private key> -newAccountAddress" +
			" <new account address>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	key, err := crypto.GetPrivateKeyByHexString(*ownerPrivateKey)

	if err != nil {
		log.Fatalf("get private key by hex string error: %v", err)
	}

	result := client.CreateAccount(key, *address)

	fmt.Printf("result: %v\n", result)
}
