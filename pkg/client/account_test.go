package client_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	accountAddress                    = "TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b"
	accountAddressWitness             = "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"
	testnetNileAddressExample         = "TUoHaVjx7n5xz8LwPRDckgFrDWhMhuSuJM"
	testnetNileAddressDelegateExample = "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g"
)

// newAccountMock returns a mockWalletServer pre-configured for
// GetAccountDetailed with deterministic data for assertion.
func newAccountMock() *mockWalletServer {
	witnessAddr, _ := common.DecodeCheck(accountAddressWitness)
	delegateToAddr, _ := common.DecodeCheck(testnetNileAddressDelegateExample)

	return &mockWalletServer{
		GetAccountFunc: func(_ context.Context, in *core.Account) (*core.Account, error) {
			return &core.Account{
				Address:   in.Address,
				Balance:   5_000_000,
				IsWitness: true,
				Allowance: 500,
				Frozen: []*core.Account_Frozen{
					{FrozenBalance: 2_000_000, ExpireTime: 1700000000000},
				},
				AccountResource: &core.Account_AccountResource{
					FrozenBalanceForEnergy: &core.Account_Frozen{
						FrozenBalance: 3_000_000,
						ExpireTime:    1700000000000,
					},
					DelegatedFrozenV2BalanceForEnergy: 500_000,
				},
				DelegatedFrozenV2BalanceForBandwidth: 600_000,
				FrozenV2: []*core.Account_FreezeV2{
					{Type: core.ResourceCode_BANDWIDTH, Amount: 1_000_000},
					{Type: core.ResourceCode_ENERGY, Amount: 2_000_000},
				},
				Votes: []*core.Vote{
					{VoteAddress: witnessAddr, VoteCount: 10},
				},
			}, nil
		},
		GetAccountResourceFunc: func(_ context.Context, _ *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{
				FreeNetLimit: 5000,
				NetLimit:     1000,
				NetUsed:      200,
				FreeNetUsed:  100,
				EnergyLimit:  50000,
				EnergyUsed:   3000,
			}, nil
		},
		GetRewardInfoFunc: func(_ context.Context, _ *api.BytesMessage) (*api.NumberMessage, error) {
			return &api.NumberMessage{Num: 42000}, nil
		},
		GetDelegatedResourceAccountIndexFunc: func(_ context.Context, _ *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
			return &core.DelegatedResourceAccountIndex{
				ToAccounts: [][]byte{delegateToAddr},
			}, nil
		},
		GetDelegatedResourceFunc: func(_ context.Context, _ *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
			return &api.DelegatedResourceList{
				DelegatedResource: []*core.DelegatedResource{
					{FrozenBalanceForBandwidth: 800_000, ExpireTimeForBandwidth: 1700000000000},
				},
			}, nil
		},
		GetDelegatedResourceAccountIndexV2Func: func(_ context.Context, _ *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
			return &core.DelegatedResourceAccountIndex{
				ToAccounts: [][]byte{delegateToAddr},
			}, nil
		},
		GetDelegatedResourceV2Func: func(_ context.Context, _ *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
			return &api.DelegatedResourceList{
				DelegatedResource: []*core.DelegatedResource{
					{FrozenBalanceForEnergy: 700_000, ExpireTimeForEnergy: 1700000000000},
				},
			}, nil
		},
		GetCanDelegatedMaxSizeFunc: func(_ context.Context, _ *api.CanDelegatedMaxSizeRequestMessage) (*api.CanDelegatedMaxSizeResponseMessage, error) {
			return &api.CanDelegatedMaxSizeResponseMessage{MaxSize: 10_000_000}, nil
		},
		GetAvailableUnfreezeCountFunc: func(_ context.Context, _ *api.GetAvailableUnfreezeCountRequestMessage) (*api.GetAvailableUnfreezeCountResponseMessage, error) {
			return &api.GetAvailableUnfreezeCountResponseMessage{Count: 5}, nil
		},
		GetCanWithdrawUnfreezeAmountFunc: func(_ context.Context, _ *api.CanWithdrawUnfreezeAmountRequestMessage) (*api.CanWithdrawUnfreezeAmountResponseMessage, error) {
			return &api.CanWithdrawUnfreezeAmountResponseMessage{Amount: 250_000}, nil
		},
	}
}

func TestGetAccountDetailed(t *testing.T) {
	c := newMockClient(t, newAccountMock())
	acc, err := c.GetAccountDetailed(accountAddress)
	require.NoError(t, err)

	// Balance
	assert.Equal(t, int64(5_000_000), acc.Balance)
	assert.Equal(t, int64(500), acc.Allowance)
	assert.True(t, acc.IsWitness)

	// Frozen V1: bandwidth 2M + energy 3M + delegated bandwidth 800K = 5.8M
	assert.Equal(t, int64(5_800_000), acc.FrozenBalance)

	// Frozen V2: BW 1M + Energy 2M + delegated BW 600K + delegated Energy 500K = 4.1M
	assert.Equal(t, int64(4_100_000), acc.FrozenBalanceV2)

	// TronPower = (totalFrozenV1 + totalFrozenV2) / 1_000_000
	// = (5_800_000 + 4_100_000) / 1_000_000 = 9
	assert.Equal(t, int64(9), acc.TronPower)
	assert.Equal(t, int64(10), acc.TronPowerUsed)

	// Bandwidth: free(5000) + net(1000) = 6000
	assert.Equal(t, int64(6000), acc.BWTotal)
	// BW used: freeUsed(100) + netUsed(200) = 300
	assert.Equal(t, int64(300), acc.BWUsed)

	// Energy
	assert.Equal(t, int64(50000), acc.EnergyTotal)
	assert.Equal(t, int64(3000), acc.EnergyUsed)

	// Rewards
	assert.Equal(t, int64(42000), acc.Rewards)

	// Withdrawable
	assert.Equal(t, int64(250_000), acc.WithdrawableBalance)

	// Unfreeze left
	assert.Equal(t, int64(5), acc.UnfreezeLeft)

	// Votes
	witnessB58 := address.Address(func() []byte {
		b, _ := common.DecodeCheck(accountAddressWitness)
		return b
	}()).String()
	assert.Equal(t, int64(10), acc.Votes[witnessB58])

	// Max delegate sizes
	assert.Equal(t, int64(10_000_000), acc.MaxCanDelegateBandwidth)
	assert.Equal(t, int64(10_000_000), acc.MaxCanDelegateEnergy)

	// Frozen V2 resources list
	assert.Len(t, acc.FrozenResourcesV2, 3) // 2 self + 1 delegated energy
}

func TestGetAccountDetailed_NotFound(t *testing.T) {
	mock := newAccountMock()
	mock.GetAccountFunc = func(_ context.Context, in *core.Account) (*core.Account, error) {
		// Return a different address to trigger "account not found"
		return &core.Account{Address: []byte{0x00}}, nil
	}

	c := newMockClient(t, mock)
	_, err := c.GetAccountDetailed(accountAddress)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account not found")
}

func TestGetAccount_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetAccount("not-a-valid-base58-address")
	require.Error(t, err)
}

func TestGetAccountNet(t *testing.T) {
	mock := &mockWalletServer{
		GetAccountNetFunc: func(_ context.Context, _ *core.Account) (*api.AccountNetMessage, error) {
			return &api.AccountNetMessage{
				FreeNetLimit: 5000,
				FreeNetUsed:  1234,
				NetLimit:     2000,
			}, nil
		},
	}

	c := newMockClient(t, mock)
	net, err := c.GetAccountNet(accountAddress)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), net.FreeNetLimit)
	assert.Equal(t, int64(1234), net.FreeNetUsed)
	assert.Equal(t, int64(2000), net.NetLimit)
}

func TestGetRewardsInfo(t *testing.T) {
	mock := &mockWalletServer{
		GetRewardInfoFunc: func(_ context.Context, _ *api.BytesMessage) (*api.NumberMessage, error) {
			return &api.NumberMessage{Num: 99999}, nil
		},
	}

	c := newMockClient(t, mock)
	rewards, err := c.GetRewardsInfo(accountAddress)
	require.NoError(t, err)
	assert.Equal(t, int64(99999), rewards)
}

func TestGetRewardsInfo_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetRewardsInfo("invalid")
	require.Error(t, err)
}

func TestFreezeV2(t *testing.T) {
	mock := &mockWalletServer{
		FreezeBalanceV2Func: func(_ context.Context, in *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			require.Equal(t, core.ResourceCode_BANDWIDTH, in.Resource)
			require.Equal(t, int64(1_000_000), in.FrozenBalance)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	freezeTx, err := c.FreezeBalanceV2(testnetNileAddressExample, core.ResourceCode_BANDWIDTH, 1_000_000)
	require.NoError(t, err)
	require.NotNil(t, freezeTx.GetTxid())
}

func TestUnfreezeV2(t *testing.T) {
	mock := &mockWalletServer{
		UnfreezeBalanceV2Func: func(_ context.Context, in *core.UnfreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			require.Equal(t, core.ResourceCode_ENERGY, in.Resource)
			require.Equal(t, int64(500_000), in.UnfreezeBalance)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.UnfreezeBalanceV2(testnetNileAddressExample, core.ResourceCode_ENERGY, 500_000)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestDelegate(t *testing.T) {
	mock := &mockWalletServer{
		DelegateResourceFunc: func(_ context.Context, in *core.DelegateResourceContract) (*api.TransactionExtention, error) {
			assert.Equal(t, core.ResourceCode_BANDWIDTH, in.Resource)
			assert.Equal(t, int64(1_000_000), in.Balance)
			assert.False(t, in.Lock)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.DelegateResource(testnetNileAddressExample, testnetNileAddressDelegateExample, core.ResourceCode_BANDWIDTH, 1_000_000, false, 0)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestDelegate_WithLock(t *testing.T) {
	mock := &mockWalletServer{
		DelegateResourceFunc: func(_ context.Context, in *core.DelegateResourceContract) (*api.TransactionExtention, error) {
			assert.True(t, in.Lock)
			assert.Equal(t, int64(86400), in.LockPeriod)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.DelegateResource(testnetNileAddressExample, testnetNileAddressDelegateExample, core.ResourceCode_ENERGY, 2_000_000, true, 86400)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestUndelegate(t *testing.T) {
	mock := &mockWalletServer{
		UnDelegateResourceFunc: func(_ context.Context, in *core.UnDelegateResourceContract) (*api.TransactionExtention, error) {
			assert.Equal(t, core.ResourceCode_BANDWIDTH, in.Resource)
			assert.Equal(t, int64(1_000_000), in.Balance)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.UnDelegateResource(testnetNileAddressExample, testnetNileAddressDelegateExample, core.ResourceCode_BANDWIDTH, 1_000_000)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestGetDelegatedResourcesV2(t *testing.T) {
	toAddr, _ := common.DecodeCheck(accountAddressWitness)

	mock := &mockWalletServer{
		GetDelegatedResourceAccountIndexV2Func: func(_ context.Context, _ *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
			return &core.DelegatedResourceAccountIndex{
				ToAccounts: [][]byte{toAddr},
			}, nil
		},
		GetDelegatedResourceV2Func: func(_ context.Context, _ *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
			return &api.DelegatedResourceList{
				DelegatedResource: []*core.DelegatedResource{
					{FrozenBalanceForBandwidth: 1_000_000},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	resources, err := c.GetDelegatedResourcesV2(accountAddress)
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, int64(1_000_000), resources[0].DelegatedResource[0].FrozenBalanceForBandwidth)
}

func TestGetReceivedDelegatedResourcesV2(t *testing.T) {
	fromAddr, _ := common.DecodeCheck(accountAddress)

	mock := &mockWalletServer{
		GetDelegatedResourceAccountIndexV2Func: func(_ context.Context, _ *api.BytesMessage) (*core.DelegatedResourceAccountIndex, error) {
			return &core.DelegatedResourceAccountIndex{
				FromAccounts: [][]byte{fromAddr},
			}, nil
		},
		GetDelegatedResourceV2Func: func(_ context.Context, _ *api.DelegatedResourceMessage) (*api.DelegatedResourceList, error) {
			return &api.DelegatedResourceList{
				DelegatedResource: []*core.DelegatedResource{
					{FrozenBalanceForEnergy: 500_000},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	resources, err := c.GetReceivedDelegatedResourcesV2(accountAddressWitness)
	require.NoError(t, err)
	require.Len(t, resources, 1)
	assert.Equal(t, int64(500_000), resources[0].DelegatedResource[0].FrozenBalanceForEnergy)
}

func TestFreezeV2_RPCError(t *testing.T) {
	mock := &mockWalletServer{
		FreezeBalanceV2Func: func(_ context.Context, _ *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	c := newMockClient(t, mock)
	_, err := c.FreezeBalanceV2(testnetNileAddressExample, core.ResourceCode_BANDWIDTH, 1_000_000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestFreezeV2_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		FreezeBalanceV2Func: func(_ context.Context, _ *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil // zero-size
		},
	}

	c := newMockClient(t, mock)
	_, err := c.FreezeBalanceV2(testnetNileAddressExample, core.ResourceCode_BANDWIDTH, 1_000_000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestFreezeV2_ResultCodeError(t *testing.T) {
	mock := &mockWalletServer{
		FreezeBalanceV2Func: func(_ context.Context, _ *core.FreezeBalanceV2Contract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Txid:        []byte{0x01},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
				Result: &api.Return{
					Result:  false,
					Code:    api.Return_BANDWITH_ERROR,
					Message: []byte("insufficient bandwidth"),
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	_, err := c.FreezeBalanceV2(testnetNileAddressExample, core.ResourceCode_BANDWIDTH, 1_000_000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient bandwidth")
}

func TestFreezeV2_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.FreezeBalanceV2("invalid-address", core.ResourceCode_BANDWIDTH, 1_000_000)
	require.Error(t, err)
}

func TestCreateAccount(t *testing.T) {
	mock := &mockWalletServer{
		CreateAccount2Func: func(_ context.Context, in *core.AccountCreateContract) (*api.TransactionExtention, error) {
			assert.NotEmpty(t, in.OwnerAddress)
			assert.NotEmpty(t, in.AccountAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.CreateAccount(accountAddress, accountAddressWitness)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestCreateAccount_InvalidFrom(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.CreateAccount("invalid", accountAddressWitness)
	require.Error(t, err)
}

func TestCreateAccount_InvalidTo(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.CreateAccount(accountAddress, "invalid")
	require.Error(t, err)
}

func TestCreateAccount_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		CreateAccount2Func: func(_ context.Context, _ *core.AccountCreateContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)
	_, err := c.CreateAccount(accountAddress, accountAddressWitness)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestUpdateAccount(t *testing.T) {
	mock := &mockWalletServer{
		UpdateAccount2Func: func(_ context.Context, in *core.AccountUpdateContract) (*api.TransactionExtention, error) {
			assert.Equal(t, []byte("TestName"), in.AccountName)
			assert.NotEmpty(t, in.OwnerAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.UpdateAccount(accountAddress, "TestName")
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestUpdateAccount_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.UpdateAccount("invalid", "TestName")
	require.Error(t, err)
}

func TestWithdrawBalance(t *testing.T) {
	mock := &mockWalletServer{
		WithdrawBalance2Func: func(_ context.Context, in *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
			assert.NotEmpty(t, in.OwnerAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.WithdrawBalance(accountAddress)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestWithdrawBalance_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.WithdrawBalance("invalid")
	require.Error(t, err)
}

func TestWithdrawBalance_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		WithdrawBalance2Func: func(_ context.Context, _ *core.WithdrawBalanceContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)
	_, err := c.WithdrawBalance(accountAddress)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestUpdateAccountPermission(t *testing.T) {
	mock := &mockWalletServer{
		AccountPermissionUpdateFunc: func(_ context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
			assert.NotNil(t, in.Owner)
			assert.Equal(t, core.Permission_Owner, in.Owner.Type)
			assert.Len(t, in.Actives, 1)
			assert.Equal(t, core.Permission_Active, in.Actives[0].Type)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	owner := map[string]interface{}{
		"threshold": int64(1),
		"keys":      map[string]int64{accountAddress: 1},
	}
	actives := []map[string]interface{}{
		{
			"name":      "active",
			"threshold": int64(1),
			"operations": map[string]bool{
				"TransferContract": true,
			},
			"keys": map[string]int64{accountAddress: 1},
		},
	}

	tx, err := c.UpdateAccountPermission(accountAddress, owner, nil, actives)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestUpdateAccountPermission_WithWitness(t *testing.T) {
	mock := &mockWalletServer{
		AccountPermissionUpdateFunc: func(_ context.Context, in *core.AccountPermissionUpdateContract) (*api.TransactionExtention, error) {
			assert.NotNil(t, in.Witness)
			assert.Equal(t, core.Permission_Witness, in.Witness.Type)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	owner := map[string]interface{}{
		"threshold": int64(1),
		"keys":      map[string]int64{accountAddress: 1},
	}
	witness := map[string]interface{}{
		"threshold": int64(1),
		"keys":      map[string]int64{accountAddress: 1},
	}

	tx, err := c.UpdateAccountPermission(accountAddress, owner, witness, nil)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestUpdateAccountPermission_NilOwner(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.UpdateAccountPermission(accountAddress, nil, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner is manadory")
}

func TestUpdateAccountPermission_TooManyActives(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	actives := make([]map[string]interface{}, 9) // > 8
	_, err := c.UpdateAccountPermission(accountAddress, map[string]interface{}{}, nil, actives)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cant have more than 8")
}

func TestUpdateAccountPermission_TooManyKeys(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	owner := map[string]interface{}{
		"threshold": int64(1),
		"keys": map[string]int64{
			"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b": 1,
			"TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH": 1,
			"TUoHaVjx7n5xz8LwPRDckgFrDWhMhuSuJM": 1,
			"TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g": 1,
			"TMuA6YqfCeX8EhbfYEg5y7S4DqzSJireY9": 1,
			"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t": 1, // 6 keys > 5
		},
	}
	_, err := c.UpdateAccountPermission(accountAddress, owner, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cant have more than 5 keys")
}

func TestUpdateAccountPermission_ThresholdExceedsWeight(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	owner := map[string]interface{}{
		"threshold": int64(10), // threshold > total weight of 1
		"keys":      map[string]int64{accountAddress: 1},
	}
	_, err := c.UpdateAccountPermission(accountAddress, owner, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key/threshold size")
}
