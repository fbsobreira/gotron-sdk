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

	flag.Parse()

	if strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0 {
		log.Fatalln("./get-asset-issue-list -grpcAddress localhost:50051")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	assetIssueList := client.GetAssetIssueList()

	fmt.Printf("asset issue list: %v\n", assetIssueList)
}
