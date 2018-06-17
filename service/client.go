package service

import (
	"context"
	"crypto/ecdsa"
	"github.com/sasaxie/go-client-api/api"
	"github.com/sasaxie/go-client-api/common/base58"
	"github.com/sasaxie/go-client-api/common/crypto"
	"github.com/sasaxie/go-client-api/common/hexutil"
	"github.com/sasaxie/go-client-api/core"
	"github.com/sasaxie/go-client-api/util"
	"google.golang.org/grpc"
	"log"
	"strconv"
	"time"
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

func (g *GrpcClient) CreateAssetIssue(ownerKey *ecdsa.PrivateKey,
	name, description, urlStr string, totalSupply, startTime, endTime,
	FreeAssetNetLimit,
	PublicFreeAssetNetLimit int64, trxNum,
	icoNum, voteScore int32, frozenSupply map[string]string) *api.Return {
	assetIssueContract := new(core.AssetIssueContract)

	assetIssueContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	assetIssueContract.Name = []byte(name)

	if totalSupply <= 0 {
		log.Fatalf("create asset issue error: total supply <= 0")
	}
	assetIssueContract.TotalSupply = totalSupply

	if trxNum <= 0 {
		log.Fatalf("create asset issue error: trxNum <= 0")
	}
	assetIssueContract.TrxNum = trxNum

	if icoNum <= 0 {
		log.Fatalf("create asset issue error: num <= 0")
	}
	assetIssueContract.Num = icoNum

	now := time.Now().UnixNano() / 1000000
	if startTime <= now {
		log.Fatalf("create asset issue error: start time <= current time")
	}
	assetIssueContract.StartTime = startTime

	if endTime <= startTime {
		log.Fatalf("create asset issue error: end time <= start time")
	}
	assetIssueContract.EndTime = endTime

	if FreeAssetNetLimit < 0 {
		log.Fatalf("create asset issue error: free asset net limit < 0")
	}
	assetIssueContract.FreeAssetNetLimit = FreeAssetNetLimit

	if PublicFreeAssetNetLimit < 0 {
		log.Fatalf("create asset issue error: public free asset net limit < 0")
	}
	assetIssueContract.PublicFreeAssetNetLimit = PublicFreeAssetNetLimit

	assetIssueContract.VoteScore = voteScore
	assetIssueContract.Description = []byte(description)
	assetIssueContract.Url = []byte(urlStr)

	for key, value := range frozenSupply {
		amount, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			log.Fatalf("create asset issue error: convert error: %v", err)
		}
		days, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			log.Fatalf("create asset issue error: convert error: %v", err)
		}
		assetIssueContractFrozenSupply := new(core.
			AssetIssueContract_FrozenSupply)
		assetIssueContractFrozenSupply.FrozenAmount = amount
		assetIssueContractFrozenSupply.FrozenDays = days
		assetIssueContract.FrozenSupply = append(assetIssueContract.
			FrozenSupply, assetIssueContractFrozenSupply)
	}

	assetIssueTransaction, err := g.Client.CreateAssetIssue(context.
		Background(), assetIssueContract)

	if err != nil {
		log.Fatalf("create asset issue error: %v", err)
	}

	if assetIssueTransaction == nil || len(assetIssueTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("create asset issue error: invalid transaction")
	}

	util.SignTransaction(assetIssueTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		assetIssueTransaction)

	if err != nil {
		log.Fatalf("create asset issue error: %v", err)
	}

	return result
}

func (g *GrpcClient) UpdateAssetIssue(ownerKey *ecdsa.PrivateKey,
	description, urlStr string,
	newLimit, newPublicLimit int64) *api.Return {

	updateAssetContract := new(core.UpdateAssetContract)

	updateAssetContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	updateAssetContract.Description = []byte(description)
	updateAssetContract.Url = []byte(urlStr)
	updateAssetContract.NewLimit = newLimit
	updateAssetContract.NewPublicLimit = newPublicLimit

	updateAssetTransaction, err := g.Client.UpdateAsset(context.
		Background(), updateAssetContract)

	if err != nil {
		log.Fatalf("update asset issue error: %v", err)
	}

	if updateAssetTransaction == nil || len(updateAssetTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("update asset issue error: invalid transaction")
	}

	util.SignTransaction(updateAssetTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		updateAssetTransaction)

	if err != nil {
		log.Fatalf("update asset issue error: %v", err)
	}

	return result
}

func (g *GrpcClient) TransferAsset(ownerKey *ecdsa.PrivateKey, toAddress,
	assetName string, amount int64) *api.Return {

	transferAssetContract := new(core.TransferAssetContract)
	transferAssetContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()
	transferAssetContract.ToAddress = base58.DecodeCheck(toAddress)
	transferAssetContract.AssetName = []byte(assetName)
	transferAssetContract.Amount = amount

	transferAssetTransaction, err := g.Client.TransferAsset(context.
		Background(), transferAssetContract)

	if err != nil {
		log.Fatalf("transfer asset error: %v", err)
	}

	if transferAssetTransaction == nil || len(transferAssetTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("transfer asset error: invalid transaction")
	}

	util.SignTransaction(transferAssetTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		transferAssetTransaction)

	if err != nil {
		log.Fatalf("transfer asset error: %v", err)
	}

	return result
}

func (g *GrpcClient) ParticipateAssetIssue(ownerKey *ecdsa.PrivateKey,
	toAddress,
	assetName string, amount int64) *api.Return {

	participateAssetIssueContract := new(core.ParticipateAssetIssueContract)
	participateAssetIssueContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()
	participateAssetIssueContract.ToAddress = base58.DecodeCheck(toAddress)
	participateAssetIssueContract.AssetName = []byte(assetName)
	participateAssetIssueContract.Amount = amount

	participateAssetIssueTransaction, err := g.Client.ParticipateAssetIssue(
		context.
			Background(), participateAssetIssueContract)

	if err != nil {
		log.Fatalf("participate asset error: %v", err)
	}

	if participateAssetIssueTransaction == nil || len(participateAssetIssueTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("participate asset error: invalid transaction")
	}

	util.SignTransaction(participateAssetIssueTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		participateAssetIssueTransaction)

	if err != nil {
		log.Fatalf("participate asset error: %v", err)
	}

	return result
}

func (g *GrpcClient) CreateWitness(ownerKey *ecdsa.PrivateKey,
	urlStr string) *api.Return {

	witnessCreateContract := new(core.WitnessCreateContract)
	witnessCreateContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()
	witnessCreateContract.Url = []byte(urlStr)

	witnessCreateTransaction, err := g.Client.CreateWitness(context.
		Background(), witnessCreateContract)

	if err != nil {
		log.Fatalf("create witness error: %v", err)
	}

	if witnessCreateTransaction == nil || len(witnessCreateTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("create witness error: invalid transaction")
	}

	util.SignTransaction(witnessCreateTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		witnessCreateTransaction)

	if err != nil {
		log.Fatalf("create witness error: %v", err)
	}

	return result
}

func (g *GrpcClient) UpdateWitness(ownerKey *ecdsa.PrivateKey,
	urlStr string) *api.Return {

	witnessUpdateContract := new(core.WitnessUpdateContract)
	witnessUpdateContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()
	witnessUpdateContract.UpdateUrl = []byte(urlStr)

	witnessUpdateTransaction, err := g.Client.UpdateWitness(context.
		Background(), witnessUpdateContract)

	if err != nil {
		log.Fatalf("update witness error: %v", err)
	}

	if witnessUpdateTransaction == nil || len(witnessUpdateTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("update witness error: invalid transaction")
	}

	util.SignTransaction(witnessUpdateTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		witnessUpdateTransaction)

	if err != nil {
		log.Fatalf("update witness error: %v", err)
	}

	return result
}

func (g *GrpcClient) VoteWitnessAccount(ownerKey *ecdsa.PrivateKey,
	witnessMap map[string]string) *api.Return {

	voteWitnessContract := new(core.VoteWitnessContract)
	voteWitnessContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	for key, value := range witnessMap {
		witnessAddress := base58.DecodeCheck(key)
		voteCount, err := strconv.ParseInt(value, 64, 10)

		if err != nil {
			log.Fatalf("vote witness account error: %v", err)
		}

		vote := new(core.VoteWitnessContract_Vote)
		vote.VoteAddress = witnessAddress
		vote.VoteCount = voteCount
		voteWitnessContract.Votes = append(voteWitnessContract.Votes, vote)
	}

	voteWitnessTransaction, err := g.Client.VoteWitnessAccount(context.
		Background(), voteWitnessContract)

	if err != nil {
		log.Fatalf("vote witness account error: %v", err)
	}

	if voteWitnessTransaction == nil || len(voteWitnessTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("vote witness account error: invalid transaction")
	}

	util.SignTransaction(voteWitnessTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		voteWitnessTransaction)

	if err != nil {
		log.Fatalf("vote witness account error: %v", err)
	}

	return result
}

func (g *GrpcClient) UnfreezeAsset(ownerKey *ecdsa.PrivateKey,
	urlStr string) *api.Return {

	unfreezeAssetContract := new(core.UnfreezeAssetContract)
	unfreezeAssetContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	unfreezeAssetTransaction, err := g.Client.UnfreezeAsset(context.
		Background(), unfreezeAssetContract)

	if err != nil {
		log.Fatalf("unfreeze asset error: %v", err)
	}

	if unfreezeAssetTransaction == nil || len(unfreezeAssetTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("unfreeze asset error: invalid transaction")
	}

	util.SignTransaction(unfreezeAssetTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		unfreezeAssetTransaction)

	if err != nil {
		log.Fatalf("unfreeze asset error: %v", err)
	}

	return result
}

func (g *GrpcClient) WithdrawBalance(ownerKey *ecdsa.PrivateKey,
	urlStr string) *api.Return {

	withdrawBalanceContract := new(core.WithdrawBalanceContract)
	withdrawBalanceContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	withdrawBalanceTransaction, err := g.Client.WithdrawBalance(context.
		Background(), withdrawBalanceContract)

	if err != nil {
		log.Fatalf("withdraw balance error: %v", err)
	}

	if withdrawBalanceTransaction == nil || len(withdrawBalanceTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("withdraw balance error: invalid transaction")
	}

	util.SignTransaction(withdrawBalanceTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(context.Background(),
		withdrawBalanceTransaction)

	if err != nil {
		log.Fatalf("withdraw balance error: %v", err)
	}

	return result
}
