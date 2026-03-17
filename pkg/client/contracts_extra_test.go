package client_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

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

func TestGetContractABIResolved_DirectABI(t *testing.T) {
	// When the contract has a non-empty ABI, return it directly without
	// attempting proxy resolution.
	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{
				Abi: &core.SmartContract_ABI{
					Entrys: []*core.SmartContract_ABI_Entry{
						{Name: "transfer"},
						{Name: "balanceOf"},
					},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABIResolved("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	require.Len(t, abi.Entrys, 2)
	assert.Equal(t, "transfer", abi.Entrys[0].Name)
}

func TestGetContractABIResolved_ProxyResolution(t *testing.T) {
	// Proxy contract has empty ABI; implementation() returns an address
	// whose contract has the real ABI.

	// ABI-encoded address: 20 bytes left-padded to 32 bytes.
	// Using TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH (hex 41...) — the EVM
	// part is the last 20 bytes without the 0x41 prefix.
	implResult := mustDecodeHex(t,
		"0000000000000000000000007a1c816367bae03d04eb1836f027314d9ebcea16",
	)

	proxyAddr := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"
	callCount := 0

	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			callCount++
			if callCount == 1 {
				// First call: proxy — empty ABI.
				return &core.SmartContract{Abi: &core.SmartContract_ABI{}}, nil
			}
			// Second call: implementation — real ABI.
			return &core.SmartContract{
				Abi: &core.SmartContract_ABI{
					Entrys: []*core.SmartContract_ABI_Entry{
						{Name: "mint"},
						{Name: "redeem"},
					},
				},
			}, nil
		},
		TriggerConstantContractFunc: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			// Verify the selector is implementation() = 0x5c60da1b.
			assert.Equal(t, []byte{0x5c, 0x60, 0xda, 0x1b}, in.Data)
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true},
				ConstantResult: [][]byte{implResult},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABIResolved(proxyAddr)
	require.NoError(t, err)
	require.Len(t, abi.Entrys, 2)
	assert.Equal(t, "mint", abi.Entrys[0].Name)
	assert.Equal(t, "redeem", abi.Entrys[1].Name)
	assert.Equal(t, 2, callCount, "should call GetContract twice (proxy + impl)")
}

func TestGetContractABIResolved_ImplementationCallFails(t *testing.T) {
	// When implementation() reverts (not a proxy), return the original ABI.
	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{Abi: &core.SmartContract_ABI{}}, nil
		},
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return nil, fmt.Errorf("REVERT opcode executed")
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABIResolved("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	assert.Empty(t, abi.GetEntrys())
}

func TestGetContractABIResolved_ZeroImplementation(t *testing.T) {
	// When implementation() returns the zero address, don't try to fetch ABI.
	zeroResult := make([]byte, 32)

	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{Abi: &core.SmartContract_ABI{}}, nil
		},
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true},
				ConstantResult: [][]byte{zeroResult},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABIResolved("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	assert.Empty(t, abi.GetEntrys())
}

func TestGetContractABIResolved_EmptyResult(t *testing.T) {
	// When implementation() returns an empty result, treat as non-proxy.
	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			return &core.SmartContract{Abi: &core.SmartContract_ABI{}}, nil
		},
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true},
				ConstantResult: [][]byte{},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABIResolved("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	assert.Empty(t, abi.GetEntrys())
}

func TestGetContractABIResolved_ProxyABIWithImplementationFunc(t *testing.T) {
	// TransparentUpgradeableProxy: proxy ABI has entries (admin functions
	// including "implementation"), but the real ABI is on the implementation.
	implResult := mustDecodeHex(t,
		"0000000000000000000000007a1c816367bae03d04eb1836f027314d9ebcea16",
	)

	callCount := 0
	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			callCount++
			if callCount == 1 {
				// Proxy ABI with "implementation" function — looks like a proxy.
				return &core.SmartContract{
					Abi: &core.SmartContract_ABI{
						Entrys: []*core.SmartContract_ABI_Entry{
							{Name: "admin", Type: core.SmartContract_ABI_Entry_Function},
							{Name: "implementation", Type: core.SmartContract_ABI_Entry_Function},
							{Name: "upgradeTo", Type: core.SmartContract_ABI_Entry_Function},
							{Name: "Upgraded", Type: core.SmartContract_ABI_Entry_Event},
						},
					},
				}, nil
			}
			// Implementation ABI with business logic.
			return &core.SmartContract{
				Abi: &core.SmartContract_ABI{
					Entrys: []*core.SmartContract_ABI_Entry{
						{Name: "execute", Type: core.SmartContract_ABI_Entry_Function},
						{Name: "owner", Type: core.SmartContract_ABI_Entry_Function},
						{Name: "initialize", Type: core.SmartContract_ABI_Entry_Function},
					},
				},
			}, nil
		},
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true},
				ConstantResult: [][]byte{implResult},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABIResolved("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	require.Len(t, abi.Entrys, 3)
	assert.Equal(t, "execute", abi.Entrys[0].Name)
	assert.Equal(t, 2, callCount)
}

func TestGetContractABIResolved_ProxyABIImplEmpty(t *testing.T) {
	// Proxy ABI has "implementation" function but the implementation
	// contract also has an empty ABI — should fall back to proxy ABI.
	callCount := 0
	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			callCount++
			if callCount == 1 {
				return &core.SmartContract{
					Abi: &core.SmartContract_ABI{
						Entrys: []*core.SmartContract_ABI_Entry{
							{Name: "implementation", Type: core.SmartContract_ABI_Entry_Function},
							{Name: "admin", Type: core.SmartContract_ABI_Entry_Function},
						},
					},
				}, nil
			}
			return &core.SmartContract{Abi: &core.SmartContract_ABI{}}, nil
		},
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			implResult := mustDecodeHex(t,
				"0000000000000000000000007a1c816367bae03d04eb1836f027314d9ebcea16",
			)
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true},
				ConstantResult: [][]byte{implResult},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABIResolved("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	// Falls back to proxy ABI since implementation ABI is empty.
	require.Len(t, abi.Entrys, 2)
	assert.Equal(t, "implementation", abi.Entrys[0].Name)
}

func TestGetContractABIResolved_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetContractABIResolved("invalid")
	require.Error(t, err)
}

func TestGetContractABIResolved_GetContractError(t *testing.T) {
	// When GetContractABI fails, propagate the error.
	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	c := newMockClient(t, mock)
	_, err := c.GetContractABIResolved("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestGetContractABIResolved_OversizedResult(t *testing.T) {
	// When implementation() returns more than 32 bytes (e.g., extra trailing
	// data), the address must be extracted from bytes [12:32], not the tail.
	// The address 7a1c816367bae03d04eb1836f027314d9ebcea16 sits at offset
	// 12..32; bytes 32..63 are trailing garbage that must be ignored.
	oversized := mustDecodeHex(t,
		"0000000000000000000000007a1c816367bae03d04eb1836f027314d9ebcea16"+
			"00000000000000000000000000000000000000000000000000000000deadbeef",
	)

	callCount := 0
	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			callCount++
			if callCount == 1 {
				return &core.SmartContract{Abi: &core.SmartContract_ABI{}}, nil
			}
			return &core.SmartContract{
				Abi: &core.SmartContract_ABI{
					Entrys: []*core.SmartContract_ABI_Entry{
						{Name: "execute"},
					},
				},
			}, nil
		},
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true},
				ConstantResult: [][]byte{oversized},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABIResolved("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	require.Len(t, abi.Entrys, 1)
	assert.Equal(t, "execute", abi.Entrys[0].Name)
}

func TestGetContractABIResolved_ImplGetContractError(t *testing.T) {
	// When GetContractABI succeeds for the proxy but fails for the
	// implementation, fall back to the proxy ABI.
	callCount := 0
	mock := &mockWalletServer{
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			callCount++
			if callCount == 1 {
				return &core.SmartContract{Abi: &core.SmartContract_ABI{}}, nil
			}
			return nil, fmt.Errorf("rate limited")
		},
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			implResult := mustDecodeHex(t,
				"0000000000000000000000007a1c816367bae03d04eb1836f027314d9ebcea16",
			)
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true},
				ConstantResult: [][]byte{implResult},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	abi, err := c.GetContractABIResolved("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	assert.Empty(t, abi.GetEntrys(), "should fall back to original proxy ABI")
	assert.Equal(t, 2, callCount)
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
