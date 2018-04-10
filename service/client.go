package service

import (
	"google.golang.org/grpc"
	"log"
	"context"
	"fmt"
	"go-client-api/api"
)

const address = "47.91.216.69:50051"

func StartClient() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := api.NewWalletClient(conn)

	accountList, err := client.ListAccounts(context.Background(),
		new(api.EmptyMessage))
	if err != nil {
		log.Fatalf("get accounts error: %v", err)
	}

	fmt.Printf("accounts: %v", accountList)
}