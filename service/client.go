package service

import (
	"google.golang.org/grpc"
	"log"
	"context"
	"github.com/tronprotocol/go-client-api/api"
	"github.com/tronprotocol/go-client-api/core"
	"github.com/tronprotocol/go-client-api/common/hexutil"
)

type GrpcClient struct {
	Address string
	Conn *grpc.ClientConn
	Client api.WalletClient
}

func NewGrpcClient(address string) (*GrpcClient) {
	client := new(GrpcClient)
	client.Address = address
	return client
}

func (g *GrpcClient) Start() {
	var err error
	g.Conn, err = grpc.Dial(g.Address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v\n", err)
	}

	g.Client = api.NewWalletClient(g.Conn)
}

func (g *GrpcClient) ListAccounts() (*api.AccountList) {
	accountList, err := g.Client.ListAccounts(context.Background(),
		new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get accounts error: %v\n", err)
	}

	return accountList
}

func (g *GrpcClient) ListWitnesses() (*api.WitnessList) {
	witnessList, err := g.Client.ListWitnesses(context.Background(),
		new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get witnesses error: %v\n", err)
	}

	return witnessList
}

func (g *GrpcClient) ListNodes() (*api.NodeList) {
	nodeList, err := g.Client.ListNodes(context.Background(),
		new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get nodes error: %v\n", err)
	}

	return nodeList
}

func (g *GrpcClient) GetAccount(address string) (*core.Account) {
	account := new(core.Account)

	var err error
	account.Address, err = hexutil.Decode(address)

	if err != nil {
		log.Fatalf("get account error: %v\n", err)
	}

	result, err := g.Client.GetAccount(context.Background(), account)

	if err != nil {
		log.Fatalf("get account error: %v\n", err)
	}

	return result
}