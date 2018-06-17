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
		log.Fatalln("./get-next-maintenance-time -grpcAddress localhost:50051")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	num := client.GetNextMaintenanceTime()

	fmt.Printf("next maintenance time: %v\n", num)
}
