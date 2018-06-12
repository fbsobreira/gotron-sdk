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
		"id: <block hash>")

	flag.Parse()

	if (strings.EqualFold("", *hash) && len(*hash) == 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./get-block-by-id -grpcAddress localhost" +
			":50051 -id 00000000000000F8E7B8B200907932D74DCC2195FB673CE6E5C194B7382BF64A")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	block := client.GetBlockById(*hash)

	fmt.Printf("block: %v\n", block)
}
