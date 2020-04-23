package service

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/fbsobreira/gotron/api"
	"github.com/fbsobreira/gotron/common/base58"
	"github.com/fbsobreira/gotron/common/crypto"
	"github.com/fbsobreira/gotron/common/hexutil"
	"github.com/fbsobreira/gotron/core"
	"github.com/fbsobreira/gotron/util"
	"google.golang.org/grpc"
)

const GrpcTimeout = 5 * time.Second

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
	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	witnessList, err := g.Client.ListWitnesses(ctx, new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get witnesses error: %v\n", err)
	}

	return witnessList
}

func (g *GrpcClient) ListNodes() *api.NodeList {
	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	nodeList, err := g.Client.ListNodes(ctx, new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get nodes error: %v\n", err)
	}

	return nodeList
}

func (g *GrpcClient) GetAccount(address string) (*core.Account, error) {
	account := new(core.Account)
	var err error

	account.Address, err = base58.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	return g.Client.GetAccount(ctx, account)
}

func (g *GrpcClient) GetNowBlock() *core.Block {
	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.GetNowBlock(ctx, new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get now block error: %v\n", err)
	}

	return result
}

func (g *GrpcClient) GetAssetIssueByAccount(address string) (*api.AssetIssueList, error) {
	account := new(core.Account)
	var err error

	account.Address, err = base58.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	return g.Client.GetAssetIssueByAccount(ctx, account)
}

func (g *GrpcClient) GetNextMaintenanceTime() (*api.NumberMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	return g.Client.GetNextMaintenanceTime(ctx,
		new(api.EmptyMessage))
}

func (g *GrpcClient) TotalTransaction() *api.NumberMessage {
	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.TotalTransaction(ctx,
		new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("total transaction error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetAccountNet(address string) (*api.AccountNetMessage, error) {
	account := new(core.Account)
	var err error

	account.Address, err = base58.DecodeCheck(address)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	return g.Client.GetAccountNet(ctx, account)
}

func (g *GrpcClient) GetAssetIssueByName(name string) *core.AssetIssueContract {

	assetName := new(api.BytesMessage)
	assetName.Value = []byte(name)

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.GetAssetIssueByName(ctx, assetName)

	if err != nil {
		log.Fatalf("get asset issue by name error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetBlockByNum(num int64) *core.Block {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.GetBlockByNum(ctx, numMessage)

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

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.GetBlockById(ctx, blockId)

	if err != nil {
		log.Fatalf("get block by id error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetAssetIssueList() *api.AssetIssueList {
	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.GetAssetIssueList(ctx, new(api.EmptyMessage))

	if err != nil {
		log.Fatalf("get asset issue list error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetBlockByLimitNext(start, end int64) *api.BlockList {
	blockLimit := new(api.BlockLimit)
	blockLimit.StartNum = start
	blockLimit.EndNum = end

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.GetBlockByLimitNext(ctx, blockLimit)

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

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.GetTransactionById(ctx, transactionId)

	if err != nil {
		log.Fatalf("get transaction by limit next error: %v", err)
	}

	return result
}

func (g *GrpcClient) GetBlockByLatestNum(num int64) *api.BlockList {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.GetBlockByLatestNum(ctx, numMessage)

	if err != nil {
		log.Fatalf("get block by latest num error: %v", err)
	}

	return result
}

func (g *GrpcClient) CreateAccount(ownerKey *ecdsa.PrivateKey,
	accountAddress string) (*api.Return, error) {
	var err error

	accountCreateContract := new(core.AccountCreateContract)
	accountCreateContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.PublicKey).Bytes()
	accountCreateContract.AccountAddress, err = base58.DecodeCheck(accountAddress)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	accountCreateTransaction, err := g.Client.CreateAccount(ctx,
		accountCreateContract)

	if err != nil {
		return nil, err
	}

	if accountCreateTransaction == nil || len(accountCreateTransaction.
		GetRawData().GetContract()) == 0 {
		return nil, fmt.Errorf("create account error: invalid transaction")
	}

	util.SignTransaction(accountCreateTransaction, ownerKey)

	return g.Client.BroadcastTransaction(ctx,
		accountCreateTransaction)
}

func (g *GrpcClient) CreateAccountByContract(accountCreateContract *core.
	AccountCreateContract) (*core.Transaction, error) {

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()
	return g.Client.CreateAccount(ctx, accountCreateContract)
}

func (g *GrpcClient) UpdateAccount(ownerKey *ecdsa.PrivateKey,
	accountName string) *api.Return {

	var err error
	accountUpdateContract := new(core.AccountUpdateContract)
	accountUpdateContract.AccountName = []byte(accountName)
	accountUpdateContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	accountUpdateTransaction, err := g.Client.UpdateAccount(ctx,
		accountUpdateContract)

	if err != nil {
		log.Fatalf("update account error: %v", err)
	}

	if accountUpdateTransaction == nil || len(accountUpdateTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("update account error: invalid transaction")
	}

	util.SignTransaction(accountUpdateTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(ctx,
		accountUpdateTransaction)

	if err != nil {
		log.Fatalf("update account error: %v", err)
	}

	return result
}

func (g *GrpcClient) Transfer(ownerKey *ecdsa.PrivateKey, toAddress string,
	amount int64) (*api.Return, error) {
	var err error

	transferContract := new(core.TransferContract)
	transferContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	transferContract.ToAddress, err = base58.DecodeCheck(toAddress)
	if err != nil {
		return nil, err
	}
	transferContract.Amount = amount

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	transferTransaction, err := g.Client.CreateTransaction(ctx,
		transferContract)

	if err != nil {
		return nil, err
	}

	if transferTransaction == nil || len(transferTransaction.
		GetRawData().GetContract()) == 0 {
		return nil, fmt.Errorf("transfer error: invalid transaction")
	}

	util.SignTransaction(transferTransaction, ownerKey)

	return g.Client.BroadcastTransaction(ctx,
		transferTransaction)
}

func (g *GrpcClient) FreezeBalance(ownerKey *ecdsa.PrivateKey,
	frozenBalance, frozenDuration int64) (*api.Return, error) {
	var err error
	freezeBalanceContract := new(core.FreezeBalanceContract)
	freezeBalanceContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()
	freezeBalanceContract.FrozenBalance = frozenBalance
	freezeBalanceContract.FrozenDuration = frozenDuration

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	freezeBalanceTransaction, err := g.Client.FreezeBalance(ctx,
		freezeBalanceContract)
	if err != nil {
		return nil, err
	}

	if freezeBalanceTransaction == nil || len(freezeBalanceTransaction.
		GetRawData().GetContract()) == 0 {
		return nil, fmt.Errorf("freeze balance error: invalid transaction")
	}

	util.SignTransaction(freezeBalanceTransaction, ownerKey)

	return g.Client.BroadcastTransaction(ctx,
		freezeBalanceTransaction)

}

func (g *GrpcClient) UnfreezeBalance(ownerKey *ecdsa.PrivateKey) (*api.Return, error) {
	var err error

	unfreezeBalanceContract := new(core.UnfreezeBalanceContract)
	unfreezeBalanceContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.PublicKey).Bytes()

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	unfreezeBalanceTransaction, err := g.Client.UnfreezeBalance(ctx,
		unfreezeBalanceContract)
	if err != nil {
		return nil, err
	}

	if unfreezeBalanceTransaction == nil || len(unfreezeBalanceTransaction.
		GetRawData().GetContract()) == 0 {
		return nil, fmt.Errorf("unfreeze balance error: invalid transaction")
	}

	util.SignTransaction(unfreezeBalanceTransaction, ownerKey)

	return g.Client.BroadcastTransaction(ctx,
		unfreezeBalanceTransaction)

}

func (g *GrpcClient) CreateAssetIssue(ownerKey *ecdsa.PrivateKey,
	name, description, abbr, urlStr string, totalSupply, startTime, endTime,
	FreeAssetNetLimit,
	PublicFreeAssetNetLimit int64, trxNum,
	icoNum, voteScore int32, frozenSupply map[string]string) *api.Return {
	assetIssueContract := new(core.AssetIssueContract)

	assetIssueContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	assetIssueContract.Name = []byte(name)

	assetIssueContract.Abbr = []byte(abbr)

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

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	assetIssueTransaction, err := g.Client.CreateAssetIssue(ctx,
		assetIssueContract)

	if err != nil {
		log.Fatalf("create asset issue error: %v", err)
	}

	if assetIssueTransaction == nil || len(assetIssueTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("create asset issue error: invalid transaction")
	}

	util.SignTransaction(assetIssueTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(ctx,
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

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	updateAssetTransaction, err := g.Client.UpdateAsset(ctx,
		updateAssetContract)

	if err != nil {
		log.Fatalf("update asset issue error: %v", err)
	}

	if updateAssetTransaction == nil || len(updateAssetTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("update asset issue error: invalid transaction")
	}

	util.SignTransaction(updateAssetTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(ctx,
		updateAssetTransaction)

	if err != nil {
		log.Fatalf("update asset issue error: %v", err)
	}

	return result
}

func (g *GrpcClient) TransferAsset(ownerKey *ecdsa.PrivateKey, toAddress,
	assetName string, amount int64) (*api.Return, error) {
	var err error

	transferAssetContract := new(core.TransferAssetContract)
	transferAssetContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	transferAssetContract.ToAddress, err = base58.DecodeCheck(toAddress)
	if err != nil {
		return nil, err
	}
	transferAssetContract.AssetName = []byte(assetName)
	transferAssetContract.Amount = amount

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	transferAssetTransaction, err := g.Client.TransferAsset(ctx,
		transferAssetContract)
	if err != nil {
		return nil, err
	}

	if transferAssetTransaction == nil || len(transferAssetTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("transfer asset error: invalid transaction")
	}

	util.SignTransaction(transferAssetTransaction, ownerKey)

	return g.Client.BroadcastTransaction(ctx,
		transferAssetTransaction)

}

func (g *GrpcClient) ParticipateAssetIssue(ownerKey *ecdsa.PrivateKey,
	toAddress,
	assetName string, amount int64) (*api.Return, error) {
	var err error

	participateAssetIssueContract := new(core.ParticipateAssetIssueContract)
	participateAssetIssueContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()
	participateAssetIssueContract.ToAddress, err = base58.DecodeCheck(toAddress)
	if err != nil {
		return nil, err
	}
	participateAssetIssueContract.AssetName = []byte(assetName)
	participateAssetIssueContract.Amount = amount

	participateAssetIssueTransaction, err := g.Client.ParticipateAssetIssue(
		context.
			Background(), participateAssetIssueContract)

	if err != nil {
		return nil, err
	}

	if participateAssetIssueTransaction == nil || len(participateAssetIssueTransaction.
		GetRawData().GetContract()) == 0 {
		return nil, fmt.Errorf("participate asset error: invalid transaction")
	}

	util.SignTransaction(participateAssetIssueTransaction, ownerKey)

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	return g.Client.BroadcastTransaction(ctx,
		participateAssetIssueTransaction)
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

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	result, err := g.Client.BroadcastTransaction(ctx,
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

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	witnessUpdateTransaction, err := g.Client.UpdateWitness(ctx,
		witnessUpdateContract)

	if err != nil {
		log.Fatalf("update witness error: %v", err)
	}

	if witnessUpdateTransaction == nil || len(witnessUpdateTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("update witness error: invalid transaction")
	}

	util.SignTransaction(witnessUpdateTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(ctx,
		witnessUpdateTransaction)

	if err != nil {
		log.Fatalf("update witness error: %v", err)
	}

	return result
}

func (g *GrpcClient) VoteWitnessAccount(ownerKey *ecdsa.PrivateKey,
	witnessMap map[string]string) (*api.Return, error) {

	voteWitnessContract := new(core.VoteWitnessContract)
	voteWitnessContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	for key, value := range witnessMap {
		witnessAddress, err := base58.DecodeCheck(key)
		if err != nil {
			return nil, err
		}
		voteCount, err := strconv.ParseInt(value, 64, 10)
		if err != nil {
			return nil, err
		}

		vote := new(core.VoteWitnessContract_Vote)
		vote.VoteAddress = witnessAddress
		vote.VoteCount = voteCount
		voteWitnessContract.Votes = append(voteWitnessContract.Votes, vote)
	}

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	voteWitnessTransaction, err := g.Client.VoteWitnessAccount(ctx,
		voteWitnessContract)

	if err != nil {
		return nil, err
	}

	if voteWitnessTransaction == nil || len(voteWitnessTransaction.
		GetRawData().GetContract()) == 0 {
		return nil, fmt.Errorf("vote witness account error: invalid transaction")
	}

	util.SignTransaction(voteWitnessTransaction, ownerKey)

	return g.Client.BroadcastTransaction(ctx,
		voteWitnessTransaction)
}

func (g *GrpcClient) UnfreezeAsset(ownerKey *ecdsa.PrivateKey,
	urlStr string) *api.Return {

	unfreezeAssetContract := new(core.UnfreezeAssetContract)
	unfreezeAssetContract.OwnerAddress = crypto.PubkeyToAddress(ownerKey.
		PublicKey).Bytes()

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	unfreezeAssetTransaction, err := g.Client.UnfreezeAsset(ctx,
		unfreezeAssetContract)

	if err != nil {
		log.Fatalf("unfreeze asset error: %v", err)
	}

	if unfreezeAssetTransaction == nil || len(unfreezeAssetTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("unfreeze asset error: invalid transaction")
	}

	util.SignTransaction(unfreezeAssetTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(ctx,
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

	ctx, cancel := context.WithTimeout(context.Background(), GrpcTimeout)
	defer cancel()

	withdrawBalanceTransaction, err := g.Client.WithdrawBalance(ctx,
		withdrawBalanceContract)

	if err != nil {
		log.Fatalf("withdraw balance error: %v", err)
	}

	if withdrawBalanceTransaction == nil || len(withdrawBalanceTransaction.
		GetRawData().GetContract()) == 0 {
		log.Fatalf("withdraw balance error: invalid transaction")
	}

	util.SignTransaction(withdrawBalanceTransaction, ownerKey)

	result, err := g.Client.BroadcastTransaction(ctx,
		withdrawBalanceTransaction)

	if err != nil {
		log.Fatalf("withdraw balance error: %v", err)
	}

	return result
}
