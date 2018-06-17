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
		log.Fatalln("./total-transaction -grpcAddress localhost:50051")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	num := client.TotalTransaction()

	fmt.Printf("total transaction: %v\n", num)
}
