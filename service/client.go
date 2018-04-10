package service

import (
	"google.golang.org/grpc"
	"log"
	"context"
	"github.com/tronprotocol/go-client-api/api"
)

type GrpcClient struct {
	Address string
	Client api.WalletClient
}

func NewGrpcClient(address string) (*GrpcClient) {
	client := new(GrpcClient)
	client.Address = address
	return client
}

func (g *GrpcClient) Start() {
	conn, err := grpc.Dial(g.Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	// TODO: defer conn.Close()

	g.Client = api.NewWalletClient(conn)
}

func (g *GrpcClient) ListAccounts() (*api.AccountList) {
	accountList, err := g.Client.ListAccounts(context.Background(),
		new(api.EmptyMessage))
	if err != nil {
		log.Fatalf("get accounts error: %v", err)
	}

	return accountList
}