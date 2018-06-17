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

	hash := flag.String("hash",
		"",
		"hash: <block hash>")

	flag.Parse()

	if (strings.EqualFold("", *hash) && len(*hash) == 0) || (strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./get-block-by-id -grpcAddress localhost" +
			":50051 -hash <block hash>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	block := client.GetBlockById(*hash)

	fmt.Printf("block: %v\n", block)
}
