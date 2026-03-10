package client_test

import (
	"context"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriggerContract(t *testing.T) {
	mock := &mockWalletServer{
		TriggerContractFunc: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			assert.NotEmpty(t, in.OwnerAddress)
			assert.NotEmpty(t, in.ContractAddress)
			assert.NotEmpty(t, in.Data)
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Txid:        []byte{0x01},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.TriggerContract(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		"transfer(address,uint256)",
		`[{"address": "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"},{"uint256": "100"}]`,
		100_000_000, 0, "", 0,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestTriggerContract_WithFeeLimit(t *testing.T) {
	mock := &mockWalletServer{
		TriggerContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Txid:        []byte{0x01},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.TriggerContract(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		"transfer(address,uint256)",
		`[{"address": "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"},{"uint256": "100"}]`,
		50_000_000, 0, "", 0,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(50_000_000), tx.Transaction.RawData.FeeLimit)
}

func TestTriggerContract_InvalidFrom(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.TriggerContract("invalid", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		"transfer(address,uint256)", `[{"address": "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"},{"uint256": "1"}]`,
		100, 0, "", 0)
	require.Error(t, err)
}

func TestTriggerContract_InvalidContract(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.TriggerContract("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", "invalid",
		"transfer(address,uint256)", `[{"address": "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"},{"uint256": "1"}]`,
		100, 0, "", 0)
	require.Error(t, err)
}

func TestTriggerContract_ResultCodeError(t *testing.T) {
	mock := &mockWalletServer{
		TriggerContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{
					Result:  false,
					Code:    api.Return_CONTRACT_EXE_ERROR,
					Message: []byte("execution reverted"),
				},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	_, err := c.TriggerContract(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		"transfer(address,uint256)",
		`[{"address": "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"},{"uint256": "100"}]`,
		100_000_000, 0, "", 0,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execution reverted")
}

func TestUpdateEnergyLimitContract(t *testing.T) {
	mock := &mockWalletServer{
		UpdateEnergyLimitFunc: func(_ context.Context, in *core.UpdateEnergyLimitContract) (*api.TransactionExtention, error) {
			assert.NotEmpty(t, in.OwnerAddress)
			assert.NotEmpty(t, in.ContractAddress)
			assert.Equal(t, int64(100000), in.OriginEnergyLimit)
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Txid:        []byte{0x01},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.UpdateEnergyLimitContract(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		100000,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestUpdateEnergyLimitContract_InvalidFrom(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.UpdateEnergyLimitContract("invalid", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", 100000)
	require.Error(t, err)
}

func TestUpdateSettingContract(t *testing.T) {
	mock := &mockWalletServer{
		UpdateSettingFunc: func(_ context.Context, in *core.UpdateSettingContract) (*api.TransactionExtention, error) {
			assert.NotEmpty(t, in.OwnerAddress)
			assert.NotEmpty(t, in.ContractAddress)
			assert.Equal(t, int64(50), in.ConsumeUserResourcePercent)
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Txid:        []byte{0x01},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.UpdateSettingContract(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		50,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestUpdateSettingContract_InvalidFrom(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.UpdateSettingContract("invalid", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", 50)
	require.Error(t, err)
}

func TestDeployContract(t *testing.T) {
	mock := &mockWalletServer{
		DeployContractFunc: func(_ context.Context, in *core.CreateSmartContract) (*api.TransactionExtention, error) {
			assert.NotEmpty(t, in.OwnerAddress)
			assert.Equal(t, "TestContract", in.NewContract.Name)
			assert.Equal(t, int64(50), in.NewContract.ConsumeUserResourcePercent)
			assert.Equal(t, int64(10000), in.NewContract.OriginEnergyLimit)
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Txid:        []byte{0x01},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.DeployContract(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TestContract",
		&core.SmartContract_ABI{},
		"608060405234801561001057600080fd5b50",
		100_000_000, 50, 10000,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestDeployContract_InvalidFrom(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.DeployContract("invalid", "TestContract", &core.SmartContract_ABI{}, "6080", 100, 50, 10000)
	require.Error(t, err)
}

func TestDeployContract_InvalidPercent(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.DeployContract("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", "Test", &core.SmartContract_ABI{}, "6080", 100, 101, 10000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "consume_user_resource_percent")

	_, err = c.DeployContract("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", "Test", &core.SmartContract_ABI{}, "6080", 100, -1, 10000)
	require.Error(t, err)
}

func TestDeployContract_InvalidEnergyLimit(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.DeployContract("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", "Test", &core.SmartContract_ABI{}, "6080", 100, 50, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "origin_energy_limit must > 0")
}

func TestGetContractABI(t *testing.T) {
	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{
				Abi: &core.SmartContract_ABI{
					Entrys: []*core.SmartContract_ABI_Entry{
						{Name: "transfer"},
					},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABI("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	require.Len(t, abi.Entrys, 1)
	assert.Equal(t, "transfer", abi.Entrys[0].Name)
}

func TestGetContractABI_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetContractABI("invalid")
	require.Error(t, err)
}

func TestUpdateWitness(t *testing.T) {
	mock := &mockWalletServer{
		UpdateWitness2Func: func(_ context.Context, in *core.WitnessUpdateContract) (*api.TransactionExtention, error) {
			assert.Equal(t, []byte("https://new-url.com"), in.UpdateUrl)
			assert.NotEmpty(t, in.OwnerAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.UpdateWitness(accountAddress, "https://new-url.com")
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestUpdateWitness_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.UpdateWitness("invalid", "https://example.com")
	require.Error(t, err)
}

func TestUpdateBrokerage(t *testing.T) {
	mock := &mockWalletServer{
		UpdateBrokerageFunc: func(_ context.Context, in *core.UpdateBrokerageContract) (*api.TransactionExtention, error) {
			assert.Equal(t, int32(20), in.Brokerage)
			assert.NotEmpty(t, in.OwnerAddress)
			return fakeTxExtention(), nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.UpdateBrokerage(accountAddress, 20)
	require.NoError(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestUpdateBrokerage_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.UpdateBrokerage("invalid", 20)
	require.Error(t, err)
}

func TestUpdateBrokerage_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		UpdateBrokerageFunc: func(_ context.Context, _ *core.UpdateBrokerageContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)
	_, err := c.UpdateBrokerage(accountAddress, 20)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}
