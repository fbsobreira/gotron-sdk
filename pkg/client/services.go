package client

import (
	"context"
	"math/big"

	"github.com/fbsobreira/gotron-sdk/pkg/account"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

// AccountService provides account-related operations.
type AccountService interface {
	GetAccountCtx(ctx context.Context, addr string) (*core.Account, error)
	GetRewardsInfoCtx(ctx context.Context, addr string) (int64, error)
	GetAccountNetCtx(ctx context.Context, addr string) (*api.AccountNetMessage, error)
	CreateAccountCtx(ctx context.Context, from, addr string) (*api.TransactionExtention, error)
	UpdateAccountCtx(ctx context.Context, from, accountName string) (*api.TransactionExtention, error)
	GetAccountDetailedCtx(ctx context.Context, addr string) (*account.Account, error)
	WithdrawBalanceCtx(ctx context.Context, from string) (*api.TransactionExtention, error)
	UpdateAccountPermissionCtx(ctx context.Context, from string, owner, witness map[string]interface{}, actives []map[string]interface{}) (*api.TransactionExtention, error)
}

// ContractService provides smart contract operations.
type ContractService interface {
	UpdateEnergyLimitContractCtx(ctx context.Context, from, contractAddress string, value int64) (*api.TransactionExtention, error)
	UpdateSettingContractCtx(ctx context.Context, from, contractAddress string, value int64) (*api.TransactionExtention, error)
	TriggerConstantContractCtx(ctx context.Context, from, contractAddress, method, jsonString string, opts ...ConstantCallOption) (*api.TransactionExtention, error)
	TriggerContractCtx(ctx context.Context, from, contractAddress, method, jsonString string, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error)
	TriggerConstantContractWithDataCtx(ctx context.Context, from, contractAddress string, data []byte, opts ...ConstantCallOption) (*api.TransactionExtention, error)
	TriggerContractWithDataCtx(ctx context.Context, from, contractAddress string, data []byte, feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error)
	EstimateEnergyCtx(ctx context.Context, from, contractAddress, method, jsonString string, tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error)
	EstimateEnergyWithDataCtx(ctx context.Context, from, contractAddress string, data []byte, tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error)
	DeployContractCtx(ctx context.Context, from, contractName string, abi *core.SmartContract_ABI, codeStr string, feeLimit, curPercent, oeLimit int64) (*api.TransactionExtention, error)
	GetContractABICtx(ctx context.Context, contractAddress string) (*core.SmartContract_ABI, error)
	GetContractABIResolvedCtx(ctx context.Context, contractAddress string) (*core.SmartContract_ABI, error)
}

// TRC20Service provides TRC20 token operations.
type TRC20Service interface {
	TRC20CallCtx(ctx context.Context, from, contractAddress, data string, constant bool, feeLimit int64) (*api.TransactionExtention, error)
	TRC20GetNameCtx(ctx context.Context, contractAddress string) (string, error)
	TRC20GetSymbolCtx(ctx context.Context, contractAddress string) (string, error)
	TRC20GetDecimalsCtx(ctx context.Context, contractAddress string) (*big.Int, error)
	TRC20ContractBalanceCtx(ctx context.Context, addr, contractAddress string) (*big.Int, error)
	TRC20SendCtx(ctx context.Context, from, to, contract string, amount *big.Int, feeLimit int64, opts ...TRC20Option) (*api.TransactionExtention, error)
	TRC20TransferFromCtx(ctx context.Context, owner, from, to, contract string, amount *big.Int, feeLimit int64, opts ...TRC20Option) (*api.TransactionExtention, error)
	TRC20ApproveCtx(ctx context.Context, from, to, contract string, amount *big.Int, feeLimit int64, opts ...TRC20Option) (*api.TransactionExtention, error)
}

// ResourceService provides resource and staking operations.
type ResourceService interface {
	GetAccountResourceCtx(ctx context.Context, addr string) (*api.AccountResourceMessage, error)
	GetDelegatedResourcesCtx(ctx context.Context, address string) ([]*api.DelegatedResourceList, error)
	GetDelegatedResourcesV2Ctx(ctx context.Context, address string) ([]*api.DelegatedResourceList, error)
	GetReceivedDelegatedResourcesV2Ctx(ctx context.Context, address string) ([]*api.DelegatedResourceList, error)
	GetCanDelegatedMaxSizeCtx(ctx context.Context, address string, resource int32) (*api.CanDelegatedMaxSizeResponseMessage, error)
	DelegateResourceCtx(ctx context.Context, from, to string, resource core.ResourceCode, delegateBalance int64, lock bool, lockPeriod int64) (*api.TransactionExtention, error)
	UnDelegateResourceCtx(ctx context.Context, owner, receiver string, resource core.ResourceCode, delegateBalance int64) (*api.TransactionExtention, error)
	FreezeBalanceCtx(ctx context.Context, from, delegateTo string, resource core.ResourceCode, frozenBalance int64) (*api.TransactionExtention, error)
	FreezeBalanceV2Ctx(ctx context.Context, from string, resource core.ResourceCode, frozenBalance int64) (*api.TransactionExtention, error)
	UnfreezeBalanceCtx(ctx context.Context, from, delegateTo string, resource core.ResourceCode) (*api.TransactionExtention, error)
	UnfreezeBalanceV2Ctx(ctx context.Context, from string, resource core.ResourceCode, unfreezeBalance int64) (*api.TransactionExtention, error)
	GetAvailableUnfreezeCountCtx(ctx context.Context, from string) (*api.GetAvailableUnfreezeCountResponseMessage, error)
	GetCanWithdrawUnfreezeAmountCtx(ctx context.Context, from string, timestamp int64) (*api.CanWithdrawUnfreezeAmountResponseMessage, error)
	WithdrawExpireUnfreezeCtx(ctx context.Context, from string, timestamp int64) (*api.TransactionExtention, error)
}

// GovernanceService provides witness and proposal operations.
type GovernanceService interface {
	ListWitnessesCtx(ctx context.Context) (*api.WitnessList, error)
	ListWitnessesPaginatedCtx(ctx context.Context, page int64, limit ...int) (*api.WitnessList, error)
	CreateWitnessCtx(ctx context.Context, from, urlStr string) (*api.TransactionExtention, error)
	UpdateWitnessCtx(ctx context.Context, from, urlStr string) (*api.TransactionExtention, error)
	VoteWitnessAccountCtx(ctx context.Context, from string, witnessMap map[string]int64) (*api.TransactionExtention, error)
	GetWitnessBrokerageCtx(ctx context.Context, witness string) (float64, error)
	UpdateBrokerageCtx(ctx context.Context, from string, commission int32) (*api.TransactionExtention, error)
	ProposalsListCtx(ctx context.Context) (*api.ProposalList, error)
	ProposalCreateCtx(ctx context.Context, from string, parameters map[int64]int64) (*api.TransactionExtention, error)
	ProposalApproveCtx(ctx context.Context, from string, id int64, confirm bool) (*api.TransactionExtention, error)
	ProposalWithdrawCtx(ctx context.Context, from string, id int64) (*api.TransactionExtention, error)
}

// TransferService provides TRX transfer operations.
type TransferService interface {
	TransferCtx(ctx context.Context, from, toAddress string, amount int64) (*api.TransactionExtention, error)
}

// AssetService provides TRC10 asset operations.
type AssetService interface {
	GetAssetIssueByAccountCtx(ctx context.Context, address string) (*api.AssetIssueList, error)
	GetAssetIssueByNameCtx(ctx context.Context, name string) (*core.AssetIssueContract, error)
	GetAssetIssueByIDCtx(ctx context.Context, tokenID string) (*core.AssetIssueContract, error)
	GetAssetIssueListCtx(ctx context.Context, page int64, limit ...int) (*api.AssetIssueList, error)
	AssetIssueCtx(ctx context.Context, from, name, description, abbr, urlStr string, precision int32, totalSupply, startTime, endTime, FreeAssetNetLimit, PublicFreeAssetNetLimit int64, trxNum, icoNum, voteScore int32, frozenSupply map[string]string) (*api.TransactionExtention, error)
	UpdateAssetIssueCtx(ctx context.Context, from, description, urlStr string, newLimit, newPublicLimit int64) (*api.TransactionExtention, error)
	TransferAssetCtx(ctx context.Context, from, toAddress, assetName string, amount int64) (*api.TransactionExtention, error)
	ParticipateAssetIssueCtx(ctx context.Context, from, issuerAddress, tokenID string, amount int64) (*api.TransactionExtention, error)
	UnfreezeAssetCtx(ctx context.Context, from string) (*api.TransactionExtention, error)
}

// ExchangeService provides TRC10 exchange (bancor) operations.
type ExchangeService interface {
	ExchangeListCtx(ctx context.Context, page int64, limit ...int) (*api.ExchangeList, error)
	ExchangeByIDCtx(ctx context.Context, id int64) (*core.Exchange, error)
	ExchangeCreateCtx(ctx context.Context, from string, tokenID1 string, amountToken1 int64, tokenID2 string, amountToken2 int64) (*api.TransactionExtention, error)
	ExchangeInjectCtx(ctx context.Context, from string, exchangeID int64, tokenID string, amountToken int64) (*api.TransactionExtention, error)
	ExchangeWithdrawCtx(ctx context.Context, from string, exchangeID int64, tokenID string, amountToken int64) (*api.TransactionExtention, error)
	ExchangeTradeCtx(ctx context.Context, from string, exchangeID int64, tokenID string, amountToken int64, amountExpected int64) (*api.TransactionExtention, error)
}

// BlockService provides block query operations.
type BlockService interface {
	GetNowBlockCtx(ctx context.Context) (*api.BlockExtention, error)
	GetBlockByNumCtx(ctx context.Context, num int64) (*api.BlockExtention, error)
	GetBlockInfoByNumCtx(ctx context.Context, num int64) (*api.TransactionInfoList, error)
	GetBlockByIDCtx(ctx context.Context, id string) (*core.Block, error)
	GetBlockByLimitNextCtx(ctx context.Context, start, end int64) (*api.BlockListExtention, error)
	GetBlockByLatestNumCtx(ctx context.Context, num int64) (*api.BlockListExtention, error)
}

// NetworkService provides network and transaction query operations.
type NetworkService interface {
	ListNodesCtx(ctx context.Context) (*api.NodeList, error)
	GetNextMaintenanceTimeCtx(ctx context.Context) (*api.NumberMessage, error)
	TotalTransactionCtx(ctx context.Context) (*api.NumberMessage, error)
	GetTransactionByIDCtx(ctx context.Context, id string) (*core.Transaction, error)
	GetTransactionInfoByIDCtx(ctx context.Context, id string) (*core.TransactionInfo, error)
	BroadcastCtx(ctx context.Context, tx *core.Transaction) (*api.Return, error)
	GetNodeInfoCtx(ctx context.Context) (*core.NodeInfo, error)
	GetEnergyPricesCtx(ctx context.Context) (*api.PricesResponseMessage, error)
	GetBandwidthPricesCtx(ctx context.Context) (*api.PricesResponseMessage, error)
	GetMemoFeeCtx(ctx context.Context) (*api.PricesResponseMessage, error)
	GetEnergyPriceHistoryCtx(ctx context.Context) ([]PriceEntry, error)
	GetBandwidthPriceHistoryCtx(ctx context.Context) ([]PriceEntry, error)
	GetMemoFeeHistoryCtx(ctx context.Context) ([]PriceEntry, error)
	GetTransactionSignWeightCtx(ctx context.Context, tx *core.Transaction) (*api.TransactionSignWeight, error)
}

// PendingService provides pending transaction pool operations.
type PendingService interface {
	GetTransactionFromPendingCtx(ctx context.Context, id string) (*core.Transaction, error)
	GetTransactionListFromPendingCtx(ctx context.Context) (*api.TransactionIdList, error)
	GetPendingSizeCtx(ctx context.Context) (*api.NumberMessage, error)
	IsTransactionPendingCtx(ctx context.Context, id string) (bool, error)
	GetPendingTransactionsByAddressCtx(ctx context.Context, address string) ([]*core.Transaction, error)
}

// Compile-time interface satisfaction checks.
var (
	_ AccountService    = (*GrpcClient)(nil)
	_ ContractService   = (*GrpcClient)(nil)
	_ TRC20Service      = (*GrpcClient)(nil)
	_ ResourceService   = (*GrpcClient)(nil)
	_ GovernanceService = (*GrpcClient)(nil)
	_ TransferService   = (*GrpcClient)(nil)
	_ AssetService      = (*GrpcClient)(nil)
	_ ExchangeService   = (*GrpcClient)(nil)
	_ BlockService      = (*GrpcClient)(nil)
	_ NetworkService    = (*GrpcClient)(nil)
	_ PendingService    = (*GrpcClient)(nil)
)

// Account returns the AccountService backed by this client.
func (g *GrpcClient) Account() AccountService { return g }

// Contract returns the ContractService backed by this client.
func (g *GrpcClient) Contract() ContractService { return g }

// TRC20 returns the TRC20Service backed by this client.
func (g *GrpcClient) TRC20() TRC20Service { return g }

// Resource returns the ResourceService backed by this client.
func (g *GrpcClient) Resource() ResourceService { return g }

// Governance returns the GovernanceService backed by this client.
func (g *GrpcClient) Governance() GovernanceService { return g }

// Transfers returns the TransferService backed by this client.
func (g *GrpcClient) Transfers() TransferService { return g }

// Assets returns the AssetService backed by this client.
func (g *GrpcClient) Assets() AssetService { return g }

// Exchange returns the ExchangeService backed by this client.
func (g *GrpcClient) Exchange() ExchangeService { return g }

// Block returns the BlockService backed by this client.
func (g *GrpcClient) Block() BlockService { return g }

// Network returns the NetworkService backed by this client.
func (g *GrpcClient) Network() NetworkService { return g }

// Pending returns the PendingService backed by this client.
func (g *GrpcClient) Pending() PendingService { return g }
