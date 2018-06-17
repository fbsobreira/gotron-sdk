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
	grpcAddress := flag.String("grpcAddress", "39.106.178.126:50051",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	ownerPrivateKey := flag.String("ownerPrivateKey", "cbe57d98134c118ed0d219c0c8bc4154372c02c1e13b5cce30dd22ecd7bed19e",
		"ownerPrivateKey: <account private key>")

	flag.Parse()

	if (strings.EqualFold("", *ownerPrivateKey) && len(
		*ownerPrivateKey) == 0) || (strings.EqualFold(
		"", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./unfreeze-balance -grpcAddress localhost" +
			":50051 -ownerPrivateKey <your private key>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	key, err := crypto.GetPrivateKeyByHexString(*ownerPrivateKey)

	if err != nil {
		log.Fatalf("get private key by hex string error: %v", err)
	}

	result := client.UnfreezeBalance(key)

	fmt.Printf("result: %v\n", result)
}
