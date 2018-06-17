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

	description := flag.String("description", "",
		"description: <update asset issue description>")

	urlStr := flag.String("url", "",
		"url: <update asset issue url>")

	newLimit := flag.Int64("newLimit", 0,
		"newLimit: <update asset issue free asset net limit>")

	newPublicLimit := flag.Int64("newPublicLimit", 0,
		"newPublicLimit: <update asset issue public free asset net"+
			" limit>")

	flag.Parse()

	if (strings.EqualFold("", *ownerPrivateKey) && len(*ownerPrivateKey) == 0) ||
		(strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) ||
		(strings.EqualFold("", *description) && len(*description) == 0) ||
		(strings.EqualFold("", *urlStr) && len(*urlStr) == 0) ||
		(*newLimit < 0) ||
		(*newPublicLimit < 0) {
		log.Fatalln("./update-asset-issue " +
			"-grpcAddress localhost:50051 " +
			"-ownerPrivateKey <your private key> " +
			"-description <new asset issue description> " +
			"-url <new asset issue url> " +
			"-newLimit <new asset issue free asset net limit> " +
			"-newPublicLimit <new asset issue public free asset net" +
			" limit>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	key, err := crypto.GetPrivateKeyByHexString(*ownerPrivateKey)

	if err != nil {
		log.Fatalf("get private key by hex string error: %v", err)
	}

	result := client.UpdateAssetIssue(
		key,
		*description,
		*urlStr,
		*newLimit,
		*newPublicLimit)

	fmt.Printf("result: %v\n", result)
}
