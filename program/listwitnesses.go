package main

import (
	"flag"
	"fmt"
	"github.com/sasaxie/go-client-api/common/hexutil"
	"github.com/sasaxie/go-client-api/service"
	"log"
	"strings"
)

func main() {
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	flag.Parse()

	if strings.EqualFold("", *grpcAddress) && len(*grpcAddress) == 0 {
		log.Fatalln("./list-witnesses -grpcAddress localhost:50051")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	witnesses := client.ListWitnesses()

	for i, v := range witnesses.Witnesses {
		addr := hexutil.Encode(v.Address)
		u := v.Url
		totalProduced := v.TotalProduced
		totalMissed := v.TotalMissed
		latestBlockNum := v.LatestBlockNum
		latestSlotNum := v.LatestSlotNum
		isJobs := v.IsJobs
		fmt.Printf("index: %d, witness: address: %s, url: %s, "+
			"total produced: %d, total missed: %d, latest block num: %d, "+
			"latest slot num: %d, is jobs: %v\n", i,
			addr, u,
			totalProduced, totalMissed, latestBlockNum, latestSlotNum, isJobs)
	}
}
