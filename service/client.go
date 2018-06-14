package service

import (
	"context"
	"crypto/ecdsa"
	"github.com/tronprotocol/go-client-api/api"
	"github.com/tronprotocol/go-client-api/common/base58"
	"github.com/tronprotocol/go-client-api/common/crypto"
	"github.com/tronprotocol/go-client-api/common/hexutil"
	"github.com/tronprotocol/go-client-api/core"
	"github.com/tronprotocol/go-client-api/util"
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

func (g *GrpcClient) GetBlockByLimitNext(start, end int64) *api.BlockList {
	blockLimit := new(api.BlockLimit)
	blockLimit.StartNum = start
	blockLimit.EndNum = end

	result, err := g.Client.GetBlockByLimitNext(context.Background(), blockLimit)

	if err != nil {
		log.Fatalf("get block by limit next error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetTransactionById(id string) *core.Transaction {
	transactionId := new(api.BytesMessage)
	var err error

	transactionId.Value, err = hexutil.Decode(id)

	if err != nil {
		log.Fatalf("get transaction by id error: %v", err)
	}

	result, err := g.Client.GetTransactionById(context.Background(), transactionId)

	if err != nil {
		log.Fatalf("get transaction by limit next error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetBlockByLatestNum(num int64) *api.BlockList {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	result, err := g.Client.GetBlockByLatestNum(context.Background(), numMessage)

	if err != nil {
		log.Fatalf("get block by latest num error: %v", err)
	}

	return result
}

func (g *GrpcClient) CreateAccount(ownerKey *ecdsa.PrivateKey,
	accountAddress string) *api.Return {

	accountCreateContract := new(core.AccountCreateContract)
	accountCreateContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.PublicKey).Bytes()
	accountCreateContract.AccountAddress = base58.DecodeCheck(accountAddress)

	accountCreateTransaction, err := g.Client.CreateAccount(context.
		Background(), accountCreateContract)

	if err != nil {
		log.Fatalf("create account error: %v", err)
	}

	if accountCreateTransaction == nil || len(accountCreateTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("create account error: invalid transaction")
	}

	util.SignTransaction(accountCreateTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		accountCreateTransaction)

	if err != nil {
		log.Fatalf("create account error: %v", err)
	}

	return result
}

func (g *GrpcClient) UpdateAccount(ownerKey *ecdsa.PrivateKey,
	accountName string) *api.Return {

	var err error
	accountUpdateContract := new(core.AccountUpdateContract)
	accountUpdateContract.AccountName = []byte(accountName)
	accountUpdateContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	accountUpdateTransaction, err := g.Client.UpdateAccount(context.
		Background(), accountUpdateContract)

	if err != nil {
		log.Fatalf("update account error: %v", err)
	}

	if accountUpdateTransaction == nil || len(accountUpdateTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("update account error: invalid transaction")
	}

	util.SignTransaction(accountUpdateTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		accountUpdateTransaction)

	if err != nil {
		log.Fatalf("update account error: %v", err)
	}

	return result
}

func (g *GrpcClient) Transfer(ownerKey *ecdsa.PrivateKey, toAddress string,
	amount int64) *api.Return {

	transferContract := new(core.TransferContract)
	transferContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()
	transferContract.ToAddress = base58.DecodeCheck(toAddress)
	transferContract.Amount = amount

	transferTransaction, err := g.Client.CreateTransaction(context.
		Background(), transferContract)

	if err != nil {
		log.Fatalf("transfer error: %v", err)
	}

	if transferTransaction == nil || len(transferTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("transfer error: invalid transaction")
	}

	util.SignTransaction(transferTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		transferTransaction)

	if err != nil {
		log.Fatalf("transfer error: %v", err)
	}

	return result
}

func (g *GrpcClient) FreezeBalance(ownerKey *ecdsa.PrivateKey,
	frozenBalance, frozenDuration int64) *api.Return {
	freezeBalanceContract := new(core.FreezeBalanceContract)
	freezeBalanceContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()
	freezeBalanceContract.FrozenBalance = frozenBalance
	freezeBalanceContract.FrozenDuration = frozenDuration

	freezeBalanceTransaction, err := g.Client.FreezeBalance(context.
		Background(), freezeBalanceContract)

	if err != nil {
		log.Fatalf("freeze balance error: %v", err)
	}

	if freezeBalanceTransaction == nil || len(freezeBalanceTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("freeze balance error: invalid transaction")
	}

	util.SignTransaction(freezeBalanceTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		freezeBalanceTransaction)

	if err != nil {
		log.Fatalf("freeze balance error: %v", err)
	}

	return result
}

func (g *GrpcClient) UnfreezeBalance(ownerKey *ecdsa.PrivateKey) *api.Return {
	unfreezeBalanceContract := new(core.UnfreezeBalanceContract)
	unfreezeBalanceContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.PublicKey).Bytes()

	unfreezeBalanceTransaction, err := g.Client.UnfreezeBalance(context.
		Background(), unfreezeBalanceContract)

	if err != nil {
		log.Fatalf("unfreeze balance error: %v", err)
	}

	if unfreezeBalanceTransaction == nil || len(unfreezeBalanceTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("unfreeze balance error: invalid transaction")
	}

	util.SignTransaction(unfreezeBalanceTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		unfreezeBalanceTransaction)

	if err != nil {
		log.Fatalf("unfreeze balance error: %v", err)
	}

	return result
}
