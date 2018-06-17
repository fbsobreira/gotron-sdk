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

	assetName := flag.String("assetName", "",
		"assetName: <asset name>")

	flag.Parse()

	if (strings.EqualFold("", *assetName) && len(*assetName) == 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./get-asset-issue-by-name -grpcAddress localhost" +
			":50051 -assetName <asset name>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	asset := client.GetAssetIssueByName(*assetName)

	fmt.Printf("asset issue: %v\n", asset)
}
