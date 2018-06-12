package service

import (
	"context"
	"github.com/tronprotocol/go-client-api/api"
	"github.com/tronprotocol/go-client-api/common/base58"
	"github.com/tronprotocol/go-client-api/common/hexutil"
	"github.com/tronprotocol/go-client-api/core"
	"google.golang.org/grpc"
	"log"
)

type GrpcClient struct {
	Address string
	Conn    *grpc.ClientConn
	Client  api.WalletClient
}

func NewGrpcClient(address string) *GrpcClient {
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

func (g *GrpcClient) ListWitnesses() *api.WitnessList {
	witnessList, err := g.Client.ListWitnesses(context.Background(),
		new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get witnesses error: %v\n", err)
	}

	return witnessList
}

func (g *GrpcClient) ListNodes() *api.NodeList {
	nodeList, err := g.Client.ListNodes(context.Background(),
		new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get nodes error: %v\n", err)
	}

	return nodeList
}

func (g *GrpcClient) GetAccount(address string) *core.Account {
	account := new(core.Account)

	account.Address = base58.DecodeCheck(address)

	result, err := g.Client.GetAccount(context.Background(), account)

	if err != nil {
		log.Fatalf("get account error: %v\n", err)
	}

	return result
}

func (g *GrpcClient) GetNowBlock() *core.Block {
	result, err := g.Client.GetNowBlock(context.Background(), new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get now block error: %v\n", err)
	}

	return result
}

func (g *GrpcClient) GetAssetIssueByAccount(address string) *api.AssetIssueList {
	account := new(core.Account)

	account.Address = base58.DecodeCheck(address)

	result, err := g.Client.GetAssetIssueByAccount(context.Background(),
		account)

	if err != nil {
		log.Fatalf("get asset issue by account error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetNextMaintenanceTime() *api.NumberMessage {

	result, err := g.Client.GetNextMaintenanceTime(context.Background(),
		new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get next maintenance time error: %v", err)
	}

	return result
}

func (g *GrpcClient) TotalTransaction() *api.NumberMessage {

	result, err := g.Client.TotalTransaction(context.Background(),
		new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("total transaction error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetAccountNet(address string) *api.AccountNetMessage {
	account := new(core.Account)

	account.Address = base58.DecodeCheck(address)

	result, err := g.Client.GetAccountNet(context.Background(), account)

	if err != nil {
		log.Fatalf("get account net error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetAssetIssueByName(name string) *core.AssetIssueContract {

	assetName := new(api.BytesMessage)
	assetName.Value = []byte(name)

	result, err := g.Client.GetAssetIssueByName(context.Background(), assetName)

	if err != nil {
		log.Fatalf("get asset issue by name error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetBlockByNum(num int64) *core.Block {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	result, err := g.Client.GetBlockByNum(context.Background(), numMessage)

	if err != nil {
		log.Fatalf("get block by num error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetBlockById(id string) *core.Block {
	blockId := new(api.BytesMessage)
	var err error

	blockId.Value, err = hexutil.Decode(id)

	if err != nil {
		log.Fatalf("get block by id error: %v", err)
	}

	result, err := g.Client.GetBlockById(context.Background(), blockId)

	if err != nil {
		log.Fatalf("get block by id error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetAssetIssueList() *api.AssetIssueList {

	result, err := g.Client.GetAssetIssueList(context.Background(), new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get asset issue list error: %v", err)
	}

	return result
}
