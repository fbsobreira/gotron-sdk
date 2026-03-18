package client_test

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	client "github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestProtoParse(t *testing.T) {
	raw := &core.TransactionRaw{}

	mb, _ := hex.DecodeString("0a020cd222081e6d180d0ea1be1340c082fc94c22e5a8e01081f1289010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412540a15419df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d121541157a629d8e8d7d43218b83240afaa02e8c300b36222497a5d5b50000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d7085c1f894c22e")

	err := proto.Unmarshal(mb, raw)
	require.Nil(t, err)

	c := raw.GetContract()[0]
	trig := &core.TriggerSmartContract{}
	err = c.GetParameter().UnmarshalTo(trig)
	require.Nil(t, err)
	assert.Equal(t, "97a5d5b50000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d", hex.EncodeToString(trig.Data))
}

func TestProtoParseR(t *testing.T) {
	mock := &mockWalletServer{
		GetBlockByNum2Func: func(_ context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
			// Return a block with one transaction containing a TriggerSmartContract
			trigData, _ := hex.DecodeString("97a5d5b50000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d")
			tsc := &core.TriggerSmartContract{
				OwnerAddress:    []byte{0x41, 0x01},
				ContractAddress: []byte{0x41, 0x02},
				Data:            trigData,
			}
			anyParam, _ := anypb.New(tsc)

			return &api.BlockExtention{
				Transactions: []*api.TransactionExtention{
					{
						Transaction: &core.Transaction{
							RawData: &core.TransactionRaw{
								Contract: []*core.Transaction_Contract{
									{
										Type:      core.Transaction_Contract_TriggerSmartContract,
										Parameter: anyParam,
									},
								},
							},
						},
					},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	block, err := c.GetBlockByNum(48763870)
	require.NoError(t, err)

	for _, tx := range block.Transactions {
		for _, contract := range tx.GetTransaction().GetRawData().GetContract() {
			switch contract.Type {
			case core.Transaction_Contract_TriggerSmartContract:
				tsc := core.TriggerSmartContract{}
				err := contract.Parameter.UnmarshalTo(&tsc)
				require.NoError(t, err)
			}
		}
	}
}

func TestEstimateEnergy(t *testing.T) {
	mock := &mockWalletServer{
		EstimateEnergyFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
			return &api.EstimateEnergyMessage{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				EnergyRequired: 20000,
			}, nil
		},
	}

	c := newMockClient(t, mock)

	estimate, err := c.EstimateEnergy(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		"transfer(address,uint256)",
		`[{"address": "TE4c73WubeWPhSF1nAovQDmQytjcaLZyY9"},{"uint256": "100"}]`,
		0, "", 0,
	)
	require.NoError(t, err)
	assert.True(t, estimate.Result.Result)
	assert.Equal(t, int64(20000), estimate.EnergyRequired)
}

func TestEstimateEnergyNotSupported(t *testing.T) {
	mock := &mockWalletServer{
		EstimateEnergyFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
			return nil, status.Error(codes.Unimplemented, "method not found")
		},
	}

	c := newMockClient(t, mock)

	_, err := c.EstimateEnergy(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		"transfer(address,uint256)",
		`[{"address": "TE4c73WubeWPhSF1nAovQDmQytjcaLZyY9"},{"uint256": "100"}]`,
		0, "", 0,
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, client.ErrEstimateEnergyNotSupported))
}

func TestGetAccount(t *testing.T) {
	// statusOf returns 9 values — mock returns a fixed ABI-encoded response
	statusOfResult, _ := hex.DecodeString(
		"0000000000000000000000000000000000000000000000000000000000000001" + // frozen
			"0000000000000000000000000000000000000000000000000000000000000002" + // unfreezeAvailableOn
			"0000000000000000000000000000000000000000000000000000000000000003" + // frozenDate
			"0000000000000000000000000000000000000000000000000000000000000004" + // pendingInterest
			"0000000000000000000000000000000000000000000000000000000000000005" + // realizedInterest
			"0000000000000000000000000000000000000000000000000000000000000006" + // APR
			"0000000000000000000000000000000000000000000000000000000000000007" + // unfrozen
			"0000000000000000000000000000000000000000000000000000000000000008" + // availableOn
			"0000000000000000000000000000000000000000000000000000000000000009", // lastClaim
	)

	statusOfABI := `[{"outputs":[{"name":"frozen","type":"uint256"},{"name":"unfreezeAvailableOn","type":"uint64"},{"name":"frozenDate","type":"uint64"},{"name":"pendingInterest","type":"uint256"},{"name":"realizedInterest","type":"uint256"},{"name":"APR","type":"uint32"},{"name":"unfrozen","type":"uint256"},{"name":"availableOn","type":"uint64"},{"name":"lastClaim","type":"uint64"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"statusOf","stateMutability":"View","type":"function"}]`

	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				ConstantResult: [][]byte{statusOfResult},
			}, nil
		},
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			a, _ := contract.JSONtoABI(statusOfABI)
			return &core.SmartContract{
				Abi: a,
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerConstantContract("",
		"TBvmoZWgmx3wqvJoDyejSXqWWogy6kCNGp",
		"statusOf(address)", `[{"address": "TQNKDtPaeSSGhtbDAykLeHEpMpfUYmSuj1"}]`)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)

	a, err := c.GetContractABI("TBvmoZWgmx3wqvJoDyejSXqWWogy6kCNGp")
	require.NoError(t, err)
	arg, err := abi.GetParser(a, "statusOf")
	require.NoError(t, err)

	result := map[string]interface{}{}
	err = arg.UnpackIntoMap(result, tx.ConstantResult[0])
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1), result["frozen"])
}

func TestTriggerConstantContractWithData(t *testing.T) {
	// Pre-packed ABI data for "balanceOf(address)" with a known address
	packedData, _ := hex.DecodeString(
		"70a08231" + // function selector: balanceOf(address)
			"0000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d",
	)
	expectedResult, _ := hex.DecodeString(
		"0000000000000000000000000000000000000000000000000000000005f5e100", // 100000000
	)

	var capturedData []byte
	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			capturedData = ct.Data
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				ConstantResult: [][]byte{expectedResult},
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerConstantContractWithData(
		"",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		packedData,
	)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)
	assert.Equal(t, packedData, capturedData)
	assert.Equal(t, expectedResult, tx.ConstantResult[0])
}

func TestTriggerConstantContractWithData_WithFrom(t *testing.T) {
	packedData, _ := hex.DecodeString("70a08231" +
		"0000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d")

	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			// Verify owner address is set (not zero address)
			assert.NotEmpty(t, ct.OwnerAddress)
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				ConstantResult: [][]byte{{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerConstantContractWithData(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		packedData,
	)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)
}

func TestTriggerContractWithData(t *testing.T) {
	// Pre-packed ABI data for "transfer(address,uint256)"
	packedData, _ := hex.DecodeString(
		"a9059cbb" + // function selector: transfer(address,uint256)
			"0000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d" +
			"00000000000000000000000000000000000000000000000000000000000f4240",
	)

	var capturedContract *core.TriggerSmartContract
	mock := &mockWalletServer{
		TriggerContractFunc: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			capturedContract = ct
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerContractWithData(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		packedData,
		100000000, // feeLimit
		0, "", 0,
	)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)
	assert.Equal(t, packedData, capturedContract.Data)
	assert.Equal(t, int64(0), capturedContract.CallValue)
}

func TestTriggerContractWithData_WithCallValue(t *testing.T) {
	packedData, _ := hex.DecodeString("a9059cbb" +
		"0000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d" +
		"00000000000000000000000000000000000000000000000000000000000f4240")

	var capturedContract *core.TriggerSmartContract
	mock := &mockWalletServer{
		TriggerContractFunc: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			capturedContract = ct
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerContractWithData(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		packedData,
		100000000,
		5000000,   // callValue
		"1000001", // tokenID
		2000000,   // tokenAmount
	)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)
	assert.Equal(t, int64(5000000), capturedContract.CallValue)
	assert.Equal(t, int64(2000000), capturedContract.CallTokenValue)
	assert.Equal(t, int64(1000001), capturedContract.TokenId)
}

func TestTriggerContractWithData_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	_, err := c.TriggerContractWithData(
		"invalid-address",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		[]byte{0x01},
		0, 0, "", 0,
	)
	require.Error(t, err)

	_, err = c.TriggerConstantContractWithData(
		"",
		"invalid-address",
		[]byte{0x01},
	)
	require.Error(t, err)
}

func TestTriggerConstantContract_WithCallValue(t *testing.T) {
	var capturedContract *core.TriggerSmartContract
	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			capturedContract = ct
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				ConstantResult: [][]byte{{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerConstantContract(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		"swap(uint256)",
		`[{"uint256": "1000"}]`,
		client.WithCallValue(1_000_000),
	)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)
	assert.Equal(t, int64(1_000_000), capturedContract.CallValue)
}

func TestTriggerConstantContract_WithTokenValue(t *testing.T) {
	var capturedContract *core.TriggerSmartContract
	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			capturedContract = ct
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				ConstantResult: [][]byte{{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerConstantContract(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		"deposit(uint256)",
		`[{"uint256": "500"}]`,
		client.WithTokenValue("1000001", 2_000_000),
	)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)
	assert.Equal(t, int64(1000001), capturedContract.TokenId)
	assert.Equal(t, int64(2_000_000), capturedContract.CallTokenValue)
}

func TestTriggerConstantContract_WithMultipleOptions(t *testing.T) {
	var capturedContract *core.TriggerSmartContract
	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			capturedContract = ct
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				ConstantResult: [][]byte{{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerConstantContract(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		"swap(uint256)",
		`[{"uint256": "1000"}]`,
		client.WithCallValue(5_000_000),
		client.WithTokenValue("1000001", 3_000_000),
	)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)
	assert.Equal(t, int64(5_000_000), capturedContract.CallValue)
	assert.Equal(t, int64(1000001), capturedContract.TokenId)
	assert.Equal(t, int64(3_000_000), capturedContract.CallTokenValue)
}

func TestTriggerConstantContractWithData_WithCallValue(t *testing.T) {
	packedData, _ := hex.DecodeString(
		"70a08231" +
			"0000000000000000000000009df085719e7e0bd5bf4fd1b2a6aed6afd2b8416d",
	)

	var capturedContract *core.TriggerSmartContract
	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, ct *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			capturedContract = ct
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				ConstantResult: [][]byte{{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerConstantContractWithData(
		"TTGhREx2pDSxFX555NWz1YwGpiBVPvQA7e",
		"TVSvjZdyDSNocHm7dP3jvCmMNsCnMTPa5W",
		packedData,
		client.WithCallValue(1_000_000),
	)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)
	assert.Equal(t, packedData, capturedContract.Data)
	assert.Equal(t, int64(1_000_000), capturedContract.CallValue)
}

func TestGetAccountMigrationContract(t *testing.T) {
	// frozenAmount returns a single uint256
	frozenResult, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000005f5e100") // 100000000

	frozenAmountABI := `[{"outputs":[{"name":"amount","type":"uint256"}],"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"frozenAmount","stateMutability":"View","type":"function"}]`

	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				ConstantResult: [][]byte{frozenResult},
			}, nil
		},
		GetContractFunc: func(_ context.Context, _ *api.BytesMessage) (*core.SmartContract, error) {
			a, _ := contract.JSONtoABI(frozenAmountABI)
			return &core.SmartContract{
				Abi: a,
			}, nil
		},
	}

	c := newMockClient(t, mock)

	tx, err := c.TriggerConstantContract("TX8h6Df74VpJsXF6sTDz1QJsq3Ec8dABc3",
		"TVoo62PAagTvNvZbB796YfZ7dWtqpPNxnL",
		"frozenAmount(address)", `[{"address": "TX8h6Df74VpJsXF6sTDz1QJsq3Ec8dABc3"}]`)
	require.NoError(t, err)
	assert.Equal(t, api.Return_SUCCESS, tx.Result.Code)

	a, err := c.GetContractABI("TVoo62PAagTvNvZbB796YfZ7dWtqpPNxnL")
	require.NoError(t, err)
	arg, err := abi.GetParser(a, "frozenAmount")
	require.NoError(t, err)

	result := map[string]interface{}{}
	err = arg.UnpackIntoMap(result, tx.ConstantResult[0])
	require.NoError(t, err)
	assert.Equal(t, int64(100000000), result["amount"].(*big.Int).Int64())
}
