package main

import (
	"flag"
	"fmt"
	"github.com/sasaxie/go-client-api/common/crypto"
	"github.com/sasaxie/go-client-api/service"
	"log"
	"strings"
)

const frozenDuration = 3

func main() {
	grpcAddress := flag.String("grpcAddress", "",
		"gRPC address: <IP:port> example: -grpcAddress localhost:50051")

	ownerPrivateKey := flag.String("ownerPrivateKey", "",
		"ownerPrivateKey: <account private key>")

	frozenBalance := flag.Int64("frozenBalance", 0,
		"frozenBalance: <frozen balance>")

	flag.Parse()

	if (strings.EqualFold("", *ownerPrivateKey) && len(
		*ownerPrivateKey) == 0) || (*frozenBalance <= 0) || (strings.EqualFold(
		"", *grpcAddress) && len(*grpcAddress) == 0) {
		log.Fatalln("./freeze-balance -grpcAddress localhost" +
			":50051 -ownerPrivateKey <your private key> -frozenBalance" +
			" <frozen balance>")
	}

	client := service.NewGrpcClient(*grpcAddress)
	client.Start()
	defer client.Conn.Close()

	key, err := crypto.GetPrivateKeyByHexString(*ownerPrivateKey)

	if err != nil {
		log.Fatalf("get private key by hex string error: %v", err)
	}

	result := client.FreezeBalance(key, *frozenBalance, frozenDuration)

	fmt.Printf("result: %v\n", result)
}
