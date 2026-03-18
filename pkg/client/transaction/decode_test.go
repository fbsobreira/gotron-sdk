package transaction

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	proto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// testAddr returns a valid 21-byte TRON address for testing.
func testAddr(suffix byte) []byte {
	addr := make([]byte, address.AddressLength)
	addr[0] = address.TronBytePrefix
	addr[address.AddressLength-1] = suffix
	return addr
}

func addrString(b []byte) string {
	return address.Address(b).String()
}

func buildTx(contractType core.Transaction_Contract_ContractType, msg proto.Message) *core.Transaction {
	paramBytes, _ := proto.Marshal(msg)
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type: contractType,
					Parameter: &anypb.Any{
						Value: paramBytes,
					},
				},
			},
		},
	}
}

func TestDecodeContractData_NilTransaction(t *testing.T) {
	_, err := DecodeContractData(nil)
	assert.ErrorIs(t, err, ErrNilTransaction)
}

func TestDecodeContractData_NilRawData(t *testing.T) {
	_, err := DecodeContractData(&core.Transaction{})
	assert.ErrorIs(t, err, ErrNilTransaction)
}

func TestDecodeContractData_NoContracts(t *testing.T) {
	tx := &core.Transaction{
		RawData: &core.TransactionRaw{},
	}
	_, err := DecodeContractData(tx)
	assert.ErrorIs(t, err, ErrNoContracts)
}

func TestDecodeContractData_NilParameter(t *testing.T) {
	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type: core.Transaction_Contract_TransferContract,
				},
			},
		},
	}
	_, err := DecodeContractData(tx)
	assert.ErrorIs(t, err, ErrNilParameter)
}

func TestDecodeContractData_UnsupportedType(t *testing.T) {
	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type: core.Transaction_Contract_AccountCreateContract,
					Parameter: &anypb.Any{
						Value: []byte{},
					},
				},
			},
		},
	}
	_, err := DecodeContractData(tx)
	assert.ErrorIs(t, err, ErrUnsupportedContract)
}

func TestDecodeContractData_TransferContract(t *testing.T) {
	owner := testAddr(0x01)
	to := testAddr(0x02)
	tx := buildTx(core.Transaction_Contract_TransferContract, &core.TransferContract{
		OwnerAddress: owner,
		ToAddress:    to,
		Amount:       100_000_000, // 100 TRX
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "TransferContract", result.Type)
	assert.Equal(t, addrString(owner), result.Fields["owner_address"])
	assert.Equal(t, addrString(to), result.Fields["to_address"])
	assert.Equal(t, "100.000000", result.Fields["amount"])
}

func TestDecodeContractData_TransferContract_ZeroAmount(t *testing.T) {
	tx := buildTx(core.Transaction_Contract_TransferContract, &core.TransferContract{
		OwnerAddress: testAddr(0x01),
		ToAddress:    testAddr(0x02),
		Amount:       0,
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "0.000000", result.Fields["amount"])
}

func TestDecodeContractData_TransferContract_FractionalAmount(t *testing.T) {
	tx := buildTx(core.Transaction_Contract_TransferContract, &core.TransferContract{
		OwnerAddress: testAddr(0x01),
		ToAddress:    testAddr(0x02),
		Amount:       1_500_000, // 1.5 TRX
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "1.500000", result.Fields["amount"])
}

func TestDecodeContractData_TransferAssetContract(t *testing.T) {
	owner := testAddr(0x01)
	to := testAddr(0x02)
	tx := buildTx(core.Transaction_Contract_TransferAssetContract, &core.TransferAssetContract{
		OwnerAddress: owner,
		ToAddress:    to,
		AssetName:    []byte("1000001"),
		Amount:       500,
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "TransferAssetContract", result.Type)
	assert.Equal(t, addrString(owner), result.Fields["owner_address"])
	assert.Equal(t, addrString(to), result.Fields["to_address"])
	assert.Equal(t, "1000001", result.Fields["asset_name"])
	assert.Equal(t, int64(500), result.Fields["amount"])
}

func TestDecodeContractData_TriggerSmartContract(t *testing.T) {
	owner := testAddr(0x01)
	contract := testAddr(0x03)
	data := []byte{0xa9, 0x05, 0x9c, 0xbb}
	tx := buildTx(core.Transaction_Contract_TriggerSmartContract, &core.TriggerSmartContract{
		OwnerAddress:    owner,
		ContractAddress: contract,
		CallValue:       10_000_000, // 10 TRX
		Data:            data,
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "TriggerSmartContract", result.Type)
	assert.Equal(t, addrString(owner), result.Fields["owner_address"])
	assert.Equal(t, addrString(contract), result.Fields["contract_address"])
	assert.Equal(t, "a9059cbb", result.Fields["data"])
	assert.Equal(t, "10.000000", result.Fields["call_value"])
}

func TestDecodeContractData_TriggerSmartContract_EmptyData(t *testing.T) {
	tx := buildTx(core.Transaction_Contract_TriggerSmartContract, &core.TriggerSmartContract{
		OwnerAddress:    testAddr(0x01),
		ContractAddress: testAddr(0x03),
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "", result.Fields["data"])
	assert.Equal(t, "0.000000", result.Fields["call_value"])
}

func TestDecodeContractData_FreezeBalanceV2Contract(t *testing.T) {
	owner := testAddr(0x01)
	tx := buildTx(core.Transaction_Contract_FreezeBalanceV2Contract, &core.FreezeBalanceV2Contract{
		OwnerAddress:  owner,
		FrozenBalance: 50_000_000, // 50 TRX
		Resource:      core.ResourceCode_ENERGY,
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "FreezeBalanceV2Contract", result.Type)
	assert.Equal(t, addrString(owner), result.Fields["owner_address"])
	assert.Equal(t, "50.000000", result.Fields["frozen_balance"])
	assert.Equal(t, "ENERGY", result.Fields["resource"])
}

func TestDecodeContractData_FreezeBalanceV2Contract_Bandwidth(t *testing.T) {
	tx := buildTx(core.Transaction_Contract_FreezeBalanceV2Contract, &core.FreezeBalanceV2Contract{
		OwnerAddress:  testAddr(0x01),
		FrozenBalance: 1_000_000,
		Resource:      core.ResourceCode_BANDWIDTH,
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "BANDWIDTH", result.Fields["resource"])
}

func TestDecodeContractData_UnfreezeBalanceV2Contract(t *testing.T) {
	owner := testAddr(0x01)
	tx := buildTx(core.Transaction_Contract_UnfreezeBalanceV2Contract, &core.UnfreezeBalanceV2Contract{
		OwnerAddress:    owner,
		UnfreezeBalance: 25_000_000, // 25 TRX
		Resource:        core.ResourceCode_BANDWIDTH,
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "UnfreezeBalanceV2Contract", result.Type)
	assert.Equal(t, addrString(owner), result.Fields["owner_address"])
	assert.Equal(t, "25.000000", result.Fields["unfreeze_balance"])
	assert.Equal(t, "BANDWIDTH", result.Fields["resource"])
}

func TestDecodeContractData_VoteWitnessContract(t *testing.T) {
	owner := testAddr(0x01)
	witness1 := testAddr(0x10)
	witness2 := testAddr(0x20)
	tx := buildTx(core.Transaction_Contract_VoteWitnessContract, &core.VoteWitnessContract{
		OwnerAddress: owner,
		Votes: []*core.VoteWitnessContract_Vote{
			{VoteAddress: witness1, VoteCount: 100},
			{VoteAddress: witness2, VoteCount: 200},
		},
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "VoteWitnessContract", result.Type)
	assert.Equal(t, addrString(owner), result.Fields["owner_address"])

	votes, ok := result.Fields["votes"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, votes, 2)
	assert.Equal(t, addrString(witness1), votes[0]["vote_address"])
	assert.Equal(t, int64(100), votes[0]["vote_count"])
	assert.Equal(t, addrString(witness2), votes[1]["vote_address"])
	assert.Equal(t, int64(200), votes[1]["vote_count"])
}

func TestDecodeContractData_VoteWitnessContract_NoVotes(t *testing.T) {
	tx := buildTx(core.Transaction_Contract_VoteWitnessContract, &core.VoteWitnessContract{
		OwnerAddress: testAddr(0x01),
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	votes, ok := result.Fields["votes"].([]map[string]any)
	require.True(t, ok)
	assert.Empty(t, votes)
}

func TestDecodeContractData_DelegateResourceContract(t *testing.T) {
	owner := testAddr(0x01)
	receiver := testAddr(0x02)
	tx := buildTx(core.Transaction_Contract_DelegateResourceContract, &core.DelegateResourceContract{
		OwnerAddress:    owner,
		ReceiverAddress: receiver,
		Balance:         10_000_000, // 10 TRX
		Resource:        core.ResourceCode_ENERGY,
		Lock:            true,
		LockPeriod:      86400,
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "DelegateResourceContract", result.Type)
	assert.Equal(t, addrString(owner), result.Fields["owner_address"])
	assert.Equal(t, addrString(receiver), result.Fields["receiver_address"])
	assert.Equal(t, "10.000000", result.Fields["balance"])
	assert.Equal(t, "ENERGY", result.Fields["resource"])
	assert.Equal(t, true, result.Fields["lock"])
	assert.Equal(t, int64(86400), result.Fields["lock_period"])
}

func TestDecodeContractData_UnDelegateResourceContract(t *testing.T) {
	owner := testAddr(0x01)
	receiver := testAddr(0x02)
	tx := buildTx(core.Transaction_Contract_UnDelegateResourceContract, &core.UnDelegateResourceContract{
		OwnerAddress:    owner,
		ReceiverAddress: receiver,
		Balance:         5_000_000, // 5 TRX
		Resource:        core.ResourceCode_BANDWIDTH,
	})

	result, err := DecodeContractData(tx)
	require.NoError(t, err)
	assert.Equal(t, "UnDelegateResourceContract", result.Type)
	assert.Equal(t, addrString(owner), result.Fields["owner_address"])
	assert.Equal(t, addrString(receiver), result.Fields["receiver_address"])
	assert.Equal(t, "5.000000", result.Fields["balance"])
	assert.Equal(t, "BANDWIDTH", result.Fields["resource"])
}

func TestSunToTRX(t *testing.T) {
	tests := []struct {
		sun      int64
		expected string
	}{
		{0, "0.000000"},
		{1, "0.000001"},
		{999_999, "0.999999"},
		{1_000_000, "1.000000"},
		{1_500_000, "1.500000"},
		{100_000_000, "100.000000"},
		{123_456_789, "123.456789"},
		{-1, "-0.000001"},
		{-999_999, "-0.999999"},
		{-1_000_000, "-1.000000"},
		{-1_500_000, "-1.500000"},
		{-123_456_789, "-123.456789"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, sunToTRX(tt.sun))
		})
	}
}
