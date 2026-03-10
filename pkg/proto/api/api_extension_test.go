package api

import (
	"bytes"
	"testing"

	core "github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/types/known/anypb"
)

func newTestExtention() *TransactionExtention {
	return &TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{
					{
						Type:      core.Transaction_Contract_TransferContract,
						Parameter: &anypb.Any{},
					},
				},
			},
		},
	}
}

func TestSetData(t *testing.T) {
	tx := newTestExtention()

	err := tx.SetData("hello memo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(tx.Transaction.RawData.Data) != "hello memo" {
		t.Errorf("expected memo 'hello memo', got %q", tx.Transaction.RawData.Data)
	}

	if tx.Txid == nil {
		t.Error("expected Txid to be set after SetData")
	}
}

func TestSetDataAlreadySet(t *testing.T) {
	tx := newTestExtention()
	tx.Transaction.RawData.Data = []byte("existing")

	err := tx.SetData("new memo")
	if err == nil {
		t.Error("expected error when memo is already set")
	}
}

func TestSetDataNil(t *testing.T) {
	var tx *TransactionExtention
	err := tx.SetData("memo")
	if err == nil {
		t.Error("expected error for nil TransactionExtention")
	}
}

func TestUpdateHash(t *testing.T) {
	tx := newTestExtention()

	err := tx.UpdateHash()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tx.Txid == nil {
		t.Fatal("expected Txid to be set")
	}

	// Hash should change when data changes
	hash1 := make([]byte, len(tx.Txid))
	copy(hash1, tx.Txid)

	tx.Transaction.RawData.Data = []byte("some data")
	err = tx.UpdateHash()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bytes.Equal(hash1, tx.Txid) {
		t.Error("expected hash to change after modifying raw data")
	}
}

func TestUpdateHashNil(t *testing.T) {
	var tx *TransactionExtention
	if err := tx.UpdateHash(); err == nil {
		t.Error("expected error for nil TransactionExtention")
	}

	tx = &TransactionExtention{}
	if err := tx.UpdateHash(); err == nil {
		t.Error("expected error for nil Transaction")
	}
}

func TestSetPermissionId(t *testing.T) {
	tx := &TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{
					{
						Type:      core.Transaction_Contract_TransferContract,
						Parameter: &anypb.Any{},
					},
				},
			},
		},
	}

	err := tx.SetPermissionId(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	contracts := tx.Transaction.RawData.GetContract()
	if contracts[0].PermissionId != 2 {
		t.Errorf("expected PermissionId=2, got %d", contracts[0].PermissionId)
	}

	// Verify hash was updated (Txid should not be nil)
	if tx.Txid == nil {
		t.Error("expected Txid to be set after SetPermissionId")
	}
}

func TestSetPermissionIdNil(t *testing.T) {
	var tx *TransactionExtention
	err := tx.SetPermissionId(2)
	if err == nil {
		t.Error("expected error for nil TransactionExtention")
	}
}

func TestSetPermissionIdNilTransaction(t *testing.T) {
	tx := &TransactionExtention{}
	err := tx.SetPermissionId(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.Transaction == nil {
		t.Fatal("expected Transaction to be initialized")
	}
	if tx.Transaction.RawData == nil {
		t.Fatal("expected RawData to be initialized")
	}
}

func TestSetPermissionIdUpdatesHash(t *testing.T) {
	tx := newTestExtention()

	// Get initial hash
	err := tx.UpdateHash()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hash1 := make([]byte, len(tx.Txid))
	copy(hash1, tx.Txid)

	// Set PermissionId and verify hash changed
	err = tx.SetPermissionId(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bytes.Equal(hash1, tx.Txid) {
		t.Error("expected hash to change after SetPermissionId")
	}
}

func TestSetPermissionIdMultipleContracts(t *testing.T) {
	tx := &TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{
					{Type: core.Transaction_Contract_TransferContract, Parameter: &anypb.Any{}},
					{Type: core.Transaction_Contract_TransferAssetContract, Parameter: &anypb.Any{}},
				},
			},
		},
	}

	err := tx.SetPermissionId(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, contract := range tx.Transaction.RawData.GetContract() {
		if contract.PermissionId != 3 {
			t.Errorf("contract[%d]: expected PermissionId=3, got %d", i, contract.PermissionId)
		}
	}
}
