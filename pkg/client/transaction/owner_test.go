package transaction

import (
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	ownerTestAddr = "TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"
)

func ownerTestAddress(t *testing.T) address.Address {
	t.Helper()
	a, err := address.Base58ToAddress(ownerTestAddr)
	require.NoError(t, err)
	return a
}

// ownerTestTx builds a transaction with one contract per param, each with the
// given permission id and the contract type named by the param's message.
func ownerTestTx(t *testing.T, permissionID int32, params ...*anypb.Any) *core.Transaction {
	t.Helper()
	tx := &core.Transaction{RawData: &core.TransactionRaw{}}
	for _, p := range params {
		var ct core.Transaction_Contract_ContractType
		if p != nil {
			name := strings.TrimPrefix(string(p.MessageName()), "protocol.")
			ct = core.Transaction_Contract_ContractType(core.Transaction_Contract_ContractType_value[name])
		}
		tx.RawData.Contract = append(tx.RawData.Contract, &core.Transaction_Contract{
			Type:         ct,
			Parameter:    p,
			PermissionId: permissionID,
		})
	}
	return tx
}

func mustAny(t *testing.T, m proto.Message) *anypb.Any {
	t.Helper()
	a, err := anypb.New(m)
	require.NoError(t, err)
	return a
}

func TestOwnerAndPermission(t *testing.T) {
	owner := ownerTestAddress(t)

	tests := []struct {
		name         string
		msg          proto.Message
		permissionID int32
		wantType     core.Transaction_Contract_ContractType
	}{
		{
			name: "transfer, owner permission",
			msg: &core.TransferContract{
				OwnerAddress: owner, ToAddress: owner, Amount: 1,
			},
			permissionID: 0,
			wantType:     core.Transaction_Contract_TransferContract,
		},
		{
			name: "transfer, active permission 2",
			msg: &core.TransferContract{
				OwnerAddress: owner, ToAddress: owner, Amount: 1,
			},
			permissionID: 2,
			wantType:     core.Transaction_Contract_TransferContract,
		},
		{
			name: "trigger smart contract, owner permission",
			msg: &core.TriggerSmartContract{
				OwnerAddress: owner, ContractAddress: owner, Data: []byte{0x01},
			},
			permissionID: 0,
			wantType:     core.Transaction_Contract_TriggerSmartContract,
		},
		{
			name: "trigger smart contract, active permission 2",
			msg: &core.TriggerSmartContract{
				OwnerAddress: owner, ContractAddress: owner, Data: []byte{0x01},
			},
			permissionID: 2,
			wantType:     core.Transaction_Contract_TriggerSmartContract,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tx := ownerTestTx(t, tt.permissionID, mustAny(t, tt.msg))

			got, id, ct, err := OwnerAndPermission(tx)
			require.NoError(t, err)
			assert.Equal(t, owner, got)
			assert.Equal(t, ownerTestAddr, got.String())
			assert.Equal(t, tt.permissionID, id)
			assert.Equal(t, tt.wantType, ct)
		})
	}
}

func TestOwnerAndPermission_ReturnsCopy(t *testing.T) {
	owner := ownerTestAddress(t)
	tx := ownerTestTx(t, 0, mustAny(t, &core.TransferContract{OwnerAddress: owner, ToAddress: owner, Amount: 1}))
	before := proto.Clone(tx)

	got, _, _, err := OwnerAndPermission(tx)
	require.NoError(t, err)
	for i := range got {
		got[i] = 0xff
	}

	assert.True(t, proto.Equal(before, tx), "mutating the returned address must not change the transaction")
	again, _, _, err := OwnerAndPermission(tx)
	require.NoError(t, err)
	assert.Equal(t, owner, again)
}

func TestOwnerAndPermission_Errors(t *testing.T) {
	owner := ownerTestAddress(t)
	transfer := func(ownerAddr []byte) *anypb.Any {
		return mustAny(t, &core.TransferContract{OwnerAddress: ownerAddr, ToAddress: owner, Amount: 1})
	}
	badPrefix := append([]byte{0x42}, owner[1:]...)

	tests := []struct {
		name    string
		tx      *core.Transaction
		wantMsg string
	}{
		{name: "nil transaction", tx: nil, wantMsg: "nil transaction"},
		{name: "nil raw data", tx: &core.Transaction{}, wantMsg: "raw data"},
		{name: "no contracts", tx: ownerTestTx(t, 0), wantMsg: "0 contracts"},
		{
			name:    "two contracts",
			tx:      ownerTestTx(t, 0, transfer(owner), transfer(owner)),
			wantMsg: "2 contracts",
		},
		{
			name: "nil contract entry",
			tx: &core.Transaction{RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{nil},
			}},
			wantMsg: "parameter",
		},
		{name: "nil parameter", tx: ownerTestTx(t, 0, nil), wantMsg: "parameter"},
		{
			name:    "unknown type URL",
			tx:      ownerTestTx(t, 0, &anypb.Any{TypeUrl: "type.googleapis.com/does.not.Exist", Value: []byte{0x0a}}),
			wantMsg: "unmarshal",
		},
		{
			name:    "shielded transfer has no owner_address",
			tx:      ownerTestTx(t, 0, mustAny(t, &core.ShieldedTransferContract{TransparentFromAddress: owner})),
			wantMsg: "owner_address",
		},
		{
			name: "type does not match parameter",
			tx: func() *core.Transaction {
				tx := ownerTestTx(t, 2, mustAny(t, &core.TriggerSmartContract{OwnerAddress: owner, ContractAddress: owner}))
				tx.RawData.Contract[0].Type = core.Transaction_Contract_TransferContract
				return tx
			}(),
			wantMsg: "does not match",
		},
		{name: "empty owner address", tx: ownerTestTx(t, 0, transfer(nil)), wantMsg: "owner_address"},
		{name: "short owner address", tx: ownerTestTx(t, 0, transfer(owner[:20])), wantMsg: "not a valid"},
		{name: "long owner address", tx: ownerTestTx(t, 0, transfer(append(append([]byte{}, owner...), 0x00))), wantMsg: "not a valid"},
		{name: "wrong prefix owner address", tx: ownerTestTx(t, 0, transfer(badPrefix)), wantMsg: "not a valid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, id, ct, err := OwnerAndPermission(tt.tx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "transaction: ")
			assert.Contains(t, err.Error(), tt.wantMsg)
			assert.Nil(t, got)
			assert.Zero(t, id)
			assert.Zero(t, ct)
		})
	}
}
