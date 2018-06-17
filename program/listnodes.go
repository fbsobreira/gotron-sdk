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
		log.Fatalln("./list-nodes -grpcAddress localhost:50051")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	nodes := client.ListNodes()

	for i, v := range nodes.Nodes {
		host := string(v.Address.Host)
		port := v.Address.Port
		fmt.Printf("index: %d, node: host: %v, port: %d\n", i, host, port)
	}
}
