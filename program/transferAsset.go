package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/fbsobreira/gotron/common/crypto"
	"github.com/fbsobreira/gotron/service"
)

func main() {
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	ownerPrivateKey := flag.String("ownerPrivateKey", "",
		"ownerPrivateKey: <account private key>")

	address := flag.String("toAddress", "",
		"toAddress: <to account address>")

	assetID := flag.String("assetID", "",
		"assetID: <Asset to transfer>")

	amount := flag.Int64("amount", 0,
		"amount: <transfer amount>")

	flag.Parse()

	if (*amount <= 0) || len(*assetID) != 7 || (strings.EqualFold("", *ownerPrivateKey) && len(
		*ownerPrivateKey) == 0) || (strings.
		EqualFold("", *address) && len(*address) == 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./transfer -grpcAddress localhost" +
			":50051 -ownerPrivateKey <your private key> -toAddress" +
			" <to account address> -amount <transfer amount> -assetID <token id>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	key, err := crypto.GetPrivateKeyByHexString(*ownerPrivateKey)

	if err != nil {
		log.Fatalf("get private key by hex string error: %v", err)
	}

	result := client.TransferAsset(key, *address, *assetID, *amount)

	fmt.Printf("result: %v\n", result)
}
