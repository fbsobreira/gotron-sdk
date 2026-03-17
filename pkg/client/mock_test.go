package client_test

import (
	"context"
	"net"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

// mockWalletServer embeds UnimplementedWalletServer so we only need to
// override the methods each test cares about.
type mockWalletServer struct {
	api.UnimplementedWalletServer

	// Contract / TRC20
	TriggerConstantContractFunc func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error)
	TriggerContractFunc         func(context.Context, *core.TriggerSmartContract) (*api.TransactionExtention, error)
	GetContractFunc             func(context.Context, *api.BytesMessage) (*core.SmartContract, error)
	EstimateEnergyFunc          func(context.Context, *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error)

	// Account
	GetAccountFunc         func(context.Context, *core.Account) (*core.Account, error)
	GetAccountResourceFunc func(context.Context, *core.Account) (*api.AccountResourceMessage, error)
	GetAccountNetFunc      func(context.Context, *core.Account) (*api.AccountNetMessage, error)
	GetRewardInfoFunc      func(context.Context, *api.BytesMessage) (*api.NumberMessage, error)

	// Block
	GetBlockByNum2Func               func(context.Context, *api.NumberMessage) (*api.BlockExtention, error)
	GetNowBlock2Func                 func(context.Context, *api.EmptyMessage) (*api.BlockExtention, error)
	GetTransactionInfoByBlockNumFunc func(context.Context, *api.NumberMessage) (*api.TransactionInfoList, error)

	// Transaction
	CreateTransaction2Func     func(context.Context, *core.TransferContract) (*api.TransactionExtention, error)
	BroadcastTransactionFunc   func(context.Context, *core.Transaction) (*api.Return, error)
	GetTransactionByIdFunc     func(context.Context, *api.BytesMessage) (*core.Transaction, error)
	GetTransactionInfoByIdFunc func(context.Context, *api.BytesMessage) (*core.TransactionInfo, error)

	// Delegation
	GetDelegatedResourceAccountIndexFunc   func(context.Context, *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error)
	GetDelegatedResourceAccountIndexV2Func func(context.Context, *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error)
	GetDelegatedResourceFunc               func(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
	GetDelegatedResourceV2Func             func(context.Context, *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error)
	GetCanDelegatedMaxSizeFunc             func(context.Context, *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error)
	GetAvailableUnfreezeCountFunc          func(context.Context, *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error)
	GetCanWithdrawUnfreezeAmountFunc       func(context.Context, *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error)

	// Freeze / Unfreeze
	FreezeBalanceV2Func    func(context.Context, *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error)
	UnfreezeBalanceV2Func  func(context.Context, *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error)
	DelegateResourceFunc   func(context.Context, *core.DelegateResourceContract) (*api.TransactionExtention, error)
	UnDelegateResourceFunc func(context.Context, *core.UnDelegateResourceContract) (*api.TransactionExtention, error)

	// Network
	ListNodesFunc                  func(context.Context, *api.EmptyMessage) (*api.NodeList, error)
	GetNextMaintenanceTimeFunc     func(context.Context, *api.EmptyMessage) (*api.NumberMessage, error)
	TotalTransactionFunc           func(context.Context, *api.EmptyMessage) (*api.NumberMessage, error)
	GetNodeInfoFunc                func(context.Context, *api.EmptyMessage) (*core.NodeInfo, error)
	GetBrokerageInfoFunc           func(context.Context, *api.BytesMessage) (*api.NumberMessage, error)
	ListWitnessesFunc              func(context.Context, *api.EmptyMessage) (*api.WitnessList, error)
	GetPaginatedNowWitnessListFunc func(context.Context, *api.PaginatedMessage) (*api.WitnessList, error)
	VoteWitnessAccount2Func        func(context.Context, *core.VoteWitnessContract) (*api.TransactionExtention, error)
	CreateWitness2Func             func(context.Context, *core.WitnessCreateContract) (*api.TransactionExtention, error)
	GetTransactionSignWeightFunc   func(context.Context, *core.Transaction) (*api.TransactionSignWeight, error)
	GetEnergyPricesFunc            func(context.Context, *api.EmptyMessage) (*api.PricesResponseMessage, error)
	GetBandwidthPricesFunc         func(context.Context, *api.EmptyMessage) (*api.PricesResponseMessage, error)
	GetMemoFeeFunc                 func(context.Context, *api.EmptyMessage) (*api.PricesResponseMessage, error)

	// Assets
	GetAssetIssueByAccountFunc     func(context.Context, *core.Account) (*api.AssetIssueList, error)
	GetAssetIssueByNameFunc        func(context.Context, *api.BytesMessage) (*core.AssetIssueContract, error)
	GetAssetIssueByIdFunc          func(context.Context, *api.BytesMessage) (*core.AssetIssueContract, error)
	GetAssetIssueListFunc          func(context.Context, *api.EmptyMessage) (*api.AssetIssueList, error)
	GetPaginatedAssetIssueListFunc func(context.Context, *api.PaginatedMessage) (*api.AssetIssueList, error)
	CreateAssetIssue2Func          func(context.Context, *core.AssetIssueContract) (*api.TransactionExtention, error)
	UpdateAsset2Func               func(context.Context, *core.UpdateAssetContract) (*api.TransactionExtention, error)
	TransferAsset2Func             func(context.Context, *core.TransferAssetContract) (*api.TransactionExtention, error)
	ParticipateAssetIssue2Func     func(context.Context, *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error)
	UnfreezeAsset2Func             func(context.Context, *core.UnfreezeAssetContract) (*api.TransactionExtention, error)

	// Exchange
	ListExchangesFunc            func(context.Context, *api.EmptyMessage) (*api.ExchangeList, error)
	GetPaginatedExchangeListFunc func(context.Context, *api.PaginatedMessage) (*api.ExchangeList, error)
	GetExchangeByIdFunc          func(context.Context, *api.BytesMessage) (*core.Exchange, error)
	ExchangeCreateFunc           func(context.Context, *core.ExchangeCreateContract) (*api.TransactionExtention, error)
	ExchangeInjectFunc           func(context.Context, *core.ExchangeInjectContract) (*api.TransactionExtention, error)
	ExchangeWithdrawFunc         func(context.Context, *core.ExchangeWithdrawContract) (*api.TransactionExtention, error)
	ExchangeTransactionFunc      func(context.Context, *core.ExchangeTransactionContract) (*api.TransactionExtention, error)

	// Proposal
	ProposalCreateFunc  func(context.Context, *core.ProposalCreateContract) (*api.TransactionExtention, error)
	ProposalApproveFunc func(context.Context, *core.ProposalApproveContract) (*api.TransactionExtention, error)
	ProposalDeleteFunc  func(context.Context, *core.ProposalDeleteContract) (*api.TransactionExtention, error)
	ListProposalsFunc   func(context.Context, *api.EmptyMessage) (*api.ProposalList, error)

	// Account create/update
	CreateAccount2Func          func(context.Context, *core.AccountCreateContract) (*api.TransactionExtention, error)
	UpdateAccount2Func          func(context.Context, *core.AccountUpdateContract) (*api.TransactionExtention, error)
	WithdrawBalance2Func        func(context.Context, *core.WithdrawBalanceContract) (*api.TransactionExtention, error)
	AccountPermissionUpdateFunc func(context.Context, *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error)

	// Bank v1
	FreezeBalance2Func         func(context.Context, *core.FreezeBalanceContract) (*api.TransactionExtention, error)
	UnfreezeBalance2Func       func(context.Context, *core.UnfreezeBalanceContract) (*api.TransactionExtention, error)
	WithdrawExpireUnfreezeFunc func(context.Context, *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error)

	// Witness update
	UpdateWitness2Func  func(context.Context, *core.WitnessUpdateContract) (*api.TransactionExtention, error)
	UpdateBrokerageFunc func(context.Context, *core.UpdateBrokerageContract) (*api.TransactionExtention, error)

	// Contract management
	UpdateEnergyLimitFunc func(context.Context, *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error)
	UpdateSettingFunc     func(context.Context, *core.UpdateSettingContract) (*api.TransactionExtention, error)
	DeployContractFunc    func(context.Context, *core.CreateSmartContract) (*api.TransactionExtention, error)
}

// --- Method overrides ---

func (m *mockWalletServer) TriggerConstantContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if m.TriggerConstantContractFunc != nil {
		return m.TriggerConstantContractFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.TriggerConstantContract(ctx, in)
}

func (m *mockWalletServer) TriggerContract(ctx context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
	if m.TriggerContractFunc != nil {
		return m.TriggerContractFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.TriggerContract(ctx, in)
}

func (m *mockWalletServer) GetContract(ctx context.Context, in *api.BytesMessage) (*core.SmartContract, error) {
	if m.GetContractFunc != nil {
		return m.GetContractFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetContract(ctx, in)
}

func (m *mockWalletServer) EstimateEnergy(ctx context.Context, in *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	if m.EstimateEnergyFunc != nil {
		return m.EstimateEnergyFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.EstimateEnergy(ctx, in)
}

func (m *mockWalletServer) GetAccount(ctx context.Context, in *core.Account) (*core.Account, error) {
	if m.GetAccountFunc != nil {
		return m.GetAccountFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetAccount(ctx, in)
}

func (m *mockWalletServer) GetAccountResource(ctx context.Context, in *core.Account) (*api.AccountResourceMessage, error) {
	if m.GetAccountResourceFunc != nil {
		return m.GetAccountResourceFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetAccountResource(ctx, in)
}

func (m *mockWalletServer) GetAccountNet(ctx context.Context, in *core.Account) (*api.AccountNetMessage, error) {
	if m.GetAccountNetFunc != nil {
		return m.GetAccountNetFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetAccountNet(ctx, in)
}

func (m *mockWalletServer) GetRewardInfo(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
	if m.GetRewardInfoFunc != nil {
		return m.GetRewardInfoFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetRewardInfo(ctx, in)
}

func (m *mockWalletServer) GetBlockByNum2(ctx context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
	if m.GetBlockByNum2Func != nil {
		return m.GetBlockByNum2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.GetBlockByNum2(ctx, in)
}

func (m *mockWalletServer) GetNowBlock2(ctx context.Context, in *api.EmptyMessage) (*api.BlockExtention, error) {
	if m.GetNowBlock2Func != nil {
		return m.GetNowBlock2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.GetNowBlock2(ctx, in)
}

func (m *mockWalletServer) GetTransactionInfoByBlockNum(ctx context.Context, in *api.NumberMessage) (*api.TransactionInfoList, error) {
	if m.GetTransactionInfoByBlockNumFunc != nil {
		return m.GetTransactionInfoByBlockNumFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetTransactionInfoByBlockNum(ctx, in)
}

func (m *mockWalletServer) CreateTransaction2(ctx context.Context, in *core.TransferContract) (*api.TransactionExtention, error) {
	if m.CreateTransaction2Func != nil {
		return m.CreateTransaction2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.CreateTransaction2(ctx, in)
}

func (m *mockWalletServer) BroadcastTransaction(ctx context.Context, in *core.Transaction) (*api.Return, error) {
	if m.BroadcastTransactionFunc != nil {
		return m.BroadcastTransactionFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.BroadcastTransaction(ctx, in)
}

func (m *mockWalletServer) GetTransactionById(ctx context.Context, in *api.BytesMessage) (*core.Transaction, error) {
	if m.GetTransactionByIdFunc != nil {
		return m.GetTransactionByIdFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetTransactionById(ctx, in)
}

func (m *mockWalletServer) GetTransactionInfoById(ctx context.Context, in *api.BytesMessage) (*core.TransactionInfo, error) {
	if m.GetTransactionInfoByIdFunc != nil {
		return m.GetTransactionInfoByIdFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetTransactionInfoById(ctx, in)
}

func (m *mockWalletServer) GetDelegatedResourceAccountIndex(ctx context.Context, in *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
	if m.GetDelegatedResourceAccountIndexFunc != nil {
		return m.GetDelegatedResourceAccountIndexFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetDelegatedResourceAccountIndex(ctx, in)
}

func (m *mockWalletServer) GetDelegatedResourceAccountIndexV2(ctx context.Context, in *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
	if m.GetDelegatedResourceAccountIndexV2Func != nil {
		return m.GetDelegatedResourceAccountIndexV2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.GetDelegatedResourceAccountIndexV2(ctx, in)
}

func (m *mockWalletServer) GetDelegatedResource(ctx context.Context, in *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	if m.GetDelegatedResourceFunc != nil {
		return m.GetDelegatedResourceFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetDelegatedResource(ctx, in)
}

func (m *mockWalletServer) GetDelegatedResourceV2(ctx context.Context, in *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
	if m.GetDelegatedResourceV2Func != nil {
		return m.GetDelegatedResourceV2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.GetDelegatedResourceV2(ctx, in)
}

func (m *mockWalletServer) GetCanDelegatedMaxSize(ctx context.Context, in *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
	if m.GetCanDelegatedMaxSizeFunc != nil {
		return m.GetCanDelegatedMaxSizeFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetCanDelegatedMaxSize(ctx, in)
}

func (m *mockWalletServer) GetAvailableUnfreezeCount(ctx context.Context, in *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
	if m.GetAvailableUnfreezeCountFunc != nil {
		return m.GetAvailableUnfreezeCountFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetAvailableUnfreezeCount(ctx, in)
}

func (m *mockWalletServer) GetCanWithdrawUnfreezeAmount(ctx context.Context, in *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
	if m.GetCanWithdrawUnfreezeAmountFunc != nil {
		return m.GetCanWithdrawUnfreezeAmountFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetCanWithdrawUnfreezeAmount(ctx, in)
}

func (m *mockWalletServer) FreezeBalanceV2(ctx context.Context, in *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	if m.FreezeBalanceV2Func != nil {
		return m.FreezeBalanceV2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.FreezeBalanceV2(ctx, in)
}

func (m *mockWalletServer) UnfreezeBalanceV2(ctx context.Context, in *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
	if m.UnfreezeBalanceV2Func != nil {
		return m.UnfreezeBalanceV2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.UnfreezeBalanceV2(ctx, in)
}

func (m *mockWalletServer) DelegateResource(ctx context.Context, in *core.DelegateResourceContract) (*api.TransactionExtention, error) {
	if m.DelegateResourceFunc != nil {
		return m.DelegateResourceFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.DelegateResource(ctx, in)
}

func (m *mockWalletServer) UnDelegateResource(ctx context.Context, in *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
	if m.UnDelegateResourceFunc != nil {
		return m.UnDelegateResourceFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.UnDelegateResource(ctx, in)
}

func (m *mockWalletServer) ListNodes(ctx context.Context, in *api.EmptyMessage) (*api.NodeList, error) {
	if m.ListNodesFunc != nil {
		return m.ListNodesFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ListNodes(ctx, in)
}

func (m *mockWalletServer) GetNextMaintenanceTime(ctx context.Context, in *api.EmptyMessage) (*api.NumberMessage, error) {
	if m.GetNextMaintenanceTimeFunc != nil {
		return m.GetNextMaintenanceTimeFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetNextMaintenanceTime(ctx, in)
}

func (m *mockWalletServer) TotalTransaction(ctx context.Context, in *api.EmptyMessage) (*api.NumberMessage, error) {
	if m.TotalTransactionFunc != nil {
		return m.TotalTransactionFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.TotalTransaction(ctx, in)
}

func (m *mockWalletServer) GetNodeInfo(ctx context.Context, in *api.EmptyMessage) (*core.NodeInfo, error) {
	if m.GetNodeInfoFunc != nil {
		return m.GetNodeInfoFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetNodeInfo(ctx, in)
}

func (m *mockWalletServer) GetBrokerageInfo(ctx context.Context, in *api.BytesMessage) (*api.NumberMessage, error) {
	if m.GetBrokerageInfoFunc != nil {
		return m.GetBrokerageInfoFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetBrokerageInfo(ctx, in)
}

func (m *mockWalletServer) ListWitnesses(ctx context.Context, in *api.EmptyMessage) (*api.WitnessList, error) {
	if m.ListWitnessesFunc != nil {
		return m.ListWitnessesFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ListWitnesses(ctx, in)
}

func (m *mockWalletServer) GetPaginatedNowWitnessList(ctx context.Context, in *api.PaginatedMessage) (*api.WitnessList, error) {
	if m.GetPaginatedNowWitnessListFunc != nil {
		return m.GetPaginatedNowWitnessListFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetPaginatedNowWitnessList(ctx, in)
}

func (m *mockWalletServer) VoteWitnessAccount2(ctx context.Context, in *core.VoteWitnessContract) (*api.TransactionExtention, error) {
	if m.VoteWitnessAccount2Func != nil {
		return m.VoteWitnessAccount2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.VoteWitnessAccount2(ctx, in)
}

func (m *mockWalletServer) CreateWitness2(ctx context.Context, in *core.WitnessCreateContract) (*api.TransactionExtention, error) {
	if m.CreateWitness2Func != nil {
		return m.CreateWitness2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.CreateWitness2(ctx, in)
}

func (m *mockWalletServer) GetTransactionSignWeight(ctx context.Context, in *core.Transaction) (*api.TransactionSignWeight, error) {
	if m.GetTransactionSignWeightFunc != nil {
		return m.GetTransactionSignWeightFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetTransactionSignWeight(ctx, in)
}

func (m *mockWalletServer) GetEnergyPrices(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	if m.GetEnergyPricesFunc != nil {
		return m.GetEnergyPricesFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetEnergyPrices(ctx, in)
}

func (m *mockWalletServer) GetBandwidthPrices(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	if m.GetBandwidthPricesFunc != nil {
		return m.GetBandwidthPricesFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetBandwidthPrices(ctx, in)
}

func (m *mockWalletServer) GetMemoFee(ctx context.Context, in *api.EmptyMessage) (*api.PricesResponseMessage, error) {
	if m.GetMemoFeeFunc != nil {
		return m.GetMemoFeeFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetMemoFee(ctx, in)
}

func (m *mockWalletServer) GetAssetIssueByAccount(ctx context.Context, in *core.Account) (*api.AssetIssueList, error) {
	if m.GetAssetIssueByAccountFunc != nil {
		return m.GetAssetIssueByAccountFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetAssetIssueByAccount(ctx, in)
}

func (m *mockWalletServer) GetAssetIssueByName(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error) {
	if m.GetAssetIssueByNameFunc != nil {
		return m.GetAssetIssueByNameFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetAssetIssueByName(ctx, in)
}

func (m *mockWalletServer) GetAssetIssueById(ctx context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error) {
	if m.GetAssetIssueByIdFunc != nil {
		return m.GetAssetIssueByIdFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetAssetIssueById(ctx, in)
}

func (m *mockWalletServer) GetAssetIssueList(ctx context.Context, in *api.EmptyMessage) (*api.AssetIssueList, error) {
	if m.GetAssetIssueListFunc != nil {
		return m.GetAssetIssueListFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetAssetIssueList(ctx, in)
}

func (m *mockWalletServer) GetPaginatedAssetIssueList(ctx context.Context, in *api.PaginatedMessage) (*api.AssetIssueList, error) {
	if m.GetPaginatedAssetIssueListFunc != nil {
		return m.GetPaginatedAssetIssueListFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetPaginatedAssetIssueList(ctx, in)
}

func (m *mockWalletServer) CreateAssetIssue2(ctx context.Context, in *core.AssetIssueContract) (*api.TransactionExtention, error) {
	if m.CreateAssetIssue2Func != nil {
		return m.CreateAssetIssue2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.CreateAssetIssue2(ctx, in)
}

func (m *mockWalletServer) UpdateAsset2(ctx context.Context, in *core.UpdateAssetContract) (*api.TransactionExtention, error) {
	if m.UpdateAsset2Func != nil {
		return m.UpdateAsset2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.UpdateAsset2(ctx, in)
}

func (m *mockWalletServer) TransferAsset2(ctx context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error) {
	if m.TransferAsset2Func != nil {
		return m.TransferAsset2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.TransferAsset2(ctx, in)
}

func (m *mockWalletServer) ParticipateAssetIssue2(ctx context.Context, in *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
	if m.ParticipateAssetIssue2Func != nil {
		return m.ParticipateAssetIssue2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.ParticipateAssetIssue2(ctx, in)
}

func (m *mockWalletServer) UnfreezeAsset2(ctx context.Context, in *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
	if m.UnfreezeAsset2Func != nil {
		return m.UnfreezeAsset2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.UnfreezeAsset2(ctx, in)
}

func (m *mockWalletServer) ListExchanges(ctx context.Context, in *api.EmptyMessage) (*api.ExchangeList, error) {
	if m.ListExchangesFunc != nil {
		return m.ListExchangesFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ListExchanges(ctx, in)
}

func (m *mockWalletServer) GetPaginatedExchangeList(ctx context.Context, in *api.PaginatedMessage) (*api.ExchangeList, error) {
	if m.GetPaginatedExchangeListFunc != nil {
		return m.GetPaginatedExchangeListFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetPaginatedExchangeList(ctx, in)
}

func (m *mockWalletServer) GetExchangeById(ctx context.Context, in *api.BytesMessage) (*core.Exchange, error) {
	if m.GetExchangeByIdFunc != nil {
		return m.GetExchangeByIdFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.GetExchangeById(ctx, in)
}

func (m *mockWalletServer) ExchangeCreate(ctx context.Context, in *core.ExchangeCreateContract) (*api.TransactionExtention, error) {
	if m.ExchangeCreateFunc != nil {
		return m.ExchangeCreateFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ExchangeCreate(ctx, in)
}

func (m *mockWalletServer) ExchangeInject(ctx context.Context, in *core.ExchangeInjectContract) (*api.TransactionExtention, error) {
	if m.ExchangeInjectFunc != nil {
		return m.ExchangeInjectFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ExchangeInject(ctx, in)
}

func (m *mockWalletServer) ExchangeWithdraw(ctx context.Context, in *core.ExchangeWithdrawContract) (*api.TransactionExtention, error) {
	if m.ExchangeWithdrawFunc != nil {
		return m.ExchangeWithdrawFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ExchangeWithdraw(ctx, in)
}

func (m *mockWalletServer) ExchangeTransaction(ctx context.Context, in *core.ExchangeTransactionContract) (*api.TransactionExtention, error) {
	if m.ExchangeTransactionFunc != nil {
		return m.ExchangeTransactionFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ExchangeTransaction(ctx, in)
}

func (m *mockWalletServer) ListProposals(ctx context.Context, in *api.EmptyMessage) (*api.ProposalList, error) {
	if m.ListProposalsFunc != nil {
		return m.ListProposalsFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ListProposals(ctx, in)
}

func (m *mockWalletServer) ProposalCreate(ctx context.Context, in *core.ProposalCreateContract) (*api.TransactionExtention, error) {
	if m.ProposalCreateFunc != nil {
		return m.ProposalCreateFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ProposalCreate(ctx, in)
}

func (m *mockWalletServer) ProposalApprove(ctx context.Context, in *core.ProposalApproveContract) (*api.TransactionExtention, error) {
	if m.ProposalApproveFunc != nil {
		return m.ProposalApproveFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ProposalApprove(ctx, in)
}

func (m *mockWalletServer) ProposalDelete(ctx context.Context, in *core.ProposalDeleteContract) (*api.TransactionExtention, error) {
	if m.ProposalDeleteFunc != nil {
		return m.ProposalDeleteFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.ProposalDelete(ctx, in)
}

func (m *mockWalletServer) CreateAccount2(ctx context.Context, in *core.AccountCreateContract) (*api.TransactionExtention, error) {
	if m.CreateAccount2Func != nil {
		return m.CreateAccount2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.CreateAccount2(ctx, in)
}

func (m *mockWalletServer) UpdateAccount2(ctx context.Context, in *core.AccountUpdateContract) (*api.TransactionExtention, error) {
	if m.UpdateAccount2Func != nil {
		return m.UpdateAccount2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.UpdateAccount2(ctx, in)
}

func (m *mockWalletServer) WithdrawBalance2(ctx context.Context, in *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
	if m.WithdrawBalance2Func != nil {
		return m.WithdrawBalance2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.WithdrawBalance2(ctx, in)
}

func (m *mockWalletServer) AccountPermissionUpdate(ctx context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
	if m.AccountPermissionUpdateFunc != nil {
		return m.AccountPermissionUpdateFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.AccountPermissionUpdate(ctx, in)
}

func (m *mockWalletServer) FreezeBalance2(ctx context.Context, in *core.FreezeBalanceContract) (*api.TransactionExtention, error) {
	if m.FreezeBalance2Func != nil {
		return m.FreezeBalance2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.FreezeBalance2(ctx, in)
}

func (m *mockWalletServer) UnfreezeBalance2(ctx context.Context, in *core.UnfreezeBalanceContract) (*api.TransactionExtention, error) {
	if m.UnfreezeBalance2Func != nil {
		return m.UnfreezeBalance2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.UnfreezeBalance2(ctx, in)
}

func (m *mockWalletServer) WithdrawExpireUnfreeze(ctx context.Context, in *core.WithdrawExpireUnfreezeContract) (*api.TransactionExtention, error) {
	if m.WithdrawExpireUnfreezeFunc != nil {
		return m.WithdrawExpireUnfreezeFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.WithdrawExpireUnfreeze(ctx, in)
}

func (m *mockWalletServer) UpdateWitness2(ctx context.Context, in *core.WitnessUpdateContract) (*api.TransactionExtention, error) {
	if m.UpdateWitness2Func != nil {
		return m.UpdateWitness2Func(ctx, in)
	}
	return m.UnimplementedWalletServer.UpdateWitness2(ctx, in)
}

func (m *mockWalletServer) UpdateBrokerage(ctx context.Context, in *core.UpdateBrokerageContract) (*api.TransactionExtention, error) {
	if m.UpdateBrokerageFunc != nil {
		return m.UpdateBrokerageFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.UpdateBrokerage(ctx, in)
}

func (m *mockWalletServer) UpdateEnergyLimit(ctx context.Context, in *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
	if m.UpdateEnergyLimitFunc != nil {
		return m.UpdateEnergyLimitFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.UpdateEnergyLimit(ctx, in)
}

func (m *mockWalletServer) UpdateSetting(ctx context.Context, in *core.UpdateSettingContract) (*api.TransactionExtention, error) {
	if m.UpdateSettingFunc != nil {
		return m.UpdateSettingFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.UpdateSetting(ctx, in)
}

func (m *mockWalletServer) DeployContract(ctx context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error) {
	if m.DeployContractFunc != nil {
		return m.DeployContractFunc(ctx, in)
	}
	return m.UnimplementedWalletServer.DeployContract(ctx, in)
}

// fakeTxExtention returns a minimal TransactionExtention that passes
// proto.Size > 0 checks in the client methods.
func fakeTxExtention() *api.TransactionExtention {
	return &api.TransactionExtention{
		Txid: []byte{0x01},
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{},
		},
		Result: &api.Return{Result: true},
	}
}

// newMockClient creates a GrpcClient connected to an in-memory gRPC server
// running the given mock. The server is shut down when the test ends.
func newMockClient(t *testing.T, mock *mockWalletServer) *client.GrpcClient {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, mock)

	go func() {
		_ = srv.Serve(lis)
	}()

	t.Cleanup(func() {
		srv.GracefulStop()
		_ = lis.Close()
	})

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() { _ = conn.Close() })

	c := client.NewGrpcClient("bufconn")
	c.Conn = conn
	c.Client = api.NewWalletClient(conn)
	return c
}
