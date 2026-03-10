package transaction

import (
	"bytes"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func newTestTransaction() *core.Transaction {
	return &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{
					Type:         core.Transaction_Contract_TransferContract,
					Parameter:    &anypb.Any{},
					PermissionId: 0,
				},
			},
		},
	}
}

func TestWithPermissionId(t *testing.T) {
	tx := newTestTransaction()

	ctrl := NewController(nil, nil, nil, tx, WithPermissionId(2))

	if ctrl.Behavior.PermissionId == nil || *ctrl.Behavior.PermissionId != 2 {
		t.Errorf("expected PermissionId=2, got %v", ctrl.Behavior.PermissionId)
	}
}

func TestWithPermissionIdZero(t *testing.T) {
	tx := newTestTransaction()
	// Pre-set a non-zero permission on the contract
	tx.GetRawData().GetContract()[0].PermissionId = 2

	ctrl := NewController(nil, nil, nil, tx, WithPermissionId(0))

	// WithPermissionId(0) should be explicitly set (not nil)
	if ctrl.Behavior.PermissionId == nil {
		t.Fatal("expected PermissionId to be set, got nil")
	}
	if *ctrl.Behavior.PermissionId != 0 {
		t.Errorf("expected PermissionId=0, got %d", *ctrl.Behavior.PermissionId)
	}

	// Apply should overwrite the contract's PermissionId back to 0
	ctrl.applyPermissionId()

	if tx.GetRawData().GetContract()[0].PermissionId != 0 {
		t.Errorf("expected contract PermissionId=0 after apply, got %d",
			tx.GetRawData().GetContract()[0].PermissionId)
	}
}

func TestWithPermissionIdDefault(t *testing.T) {
	tx := newTestTransaction()

	ctrl := NewController(nil, nil, nil, tx)

	if ctrl.Behavior.PermissionId != nil {
		t.Errorf("expected default PermissionId=nil, got %d", *ctrl.Behavior.PermissionId)
	}
}

func TestSetPermissionId(t *testing.T) {
	tx := newTestTransaction()

	setPermissionId(tx, 2)

	contracts := tx.GetRawData().GetContract()
	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(contracts))
	}
	if contracts[0].PermissionId != 2 {
		t.Errorf("expected PermissionId=2, got %d", contracts[0].PermissionId)
	}
}

func TestSetPermissionIdMultipleContracts(t *testing.T) {
	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			Contract: []*core.Transaction_Contract{
				{Type: core.Transaction_Contract_TransferContract, Parameter: &anypb.Any{}},
				{Type: core.Transaction_Contract_TransferAssetContract, Parameter: &anypb.Any{}},
			},
		},
	}

	setPermissionId(tx, 3)

	for i, contract := range tx.GetRawData().GetContract() {
		if contract.PermissionId != 3 {
			t.Errorf("contract[%d]: expected PermissionId=3, got %d", i, contract.PermissionId)
		}
	}
}

func TestApplyPermissionId(t *testing.T) {
	tx := newTestTransaction()
	ctrl := NewController(nil, nil, nil, tx, WithPermissionId(2))

	ctrl.applyPermissionId()

	contracts := ctrl.tx.GetRawData().GetContract()
	if contracts[0].PermissionId != 2 {
		t.Errorf("expected PermissionId=2 after apply, got %d", contracts[0].PermissionId)
	}
}

func TestApplyPermissionIdSkipsWhenNotSet(t *testing.T) {
	tx := newTestTransaction()
	// Manually set a non-zero value on the contract
	tx.GetRawData().GetContract()[0].PermissionId = 5

	ctrl := NewController(nil, nil, nil, tx) // no WithPermissionId option

	ctrl.applyPermissionId()

	// Should NOT overwrite because Behavior.PermissionId is nil
	if tx.GetRawData().GetContract()[0].PermissionId != 5 {
		t.Errorf("expected PermissionId=5 to be preserved, got %d",
			tx.GetRawData().GetContract()[0].PermissionId)
	}
}

func TestSetPermissionIdNilSafe(t *testing.T) {
	// Should not panic on nil transaction or nil raw data
	setPermissionId(&core.Transaction{}, 2)
	setPermissionId(&core.Transaction{RawData: &core.TransactionRaw{}}, 2)
}

func TestSignTransactionWithPermissionId(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	makeTx := func() *core.Transaction {
		return &core.Transaction{
			RawData: &core.TransactionRaw{
				Contract: []*core.Transaction_Contract{
					{
						Type:      core.Transaction_Contract_TransferContract,
						Parameter: &anypb.Any{},
					},
				},
			},
		}
	}

	// Sign without PermissionId
	tx1 := makeTx()
	signed1, err := SignTransaction(tx1, privKey)
	if err != nil {
		t.Fatalf("sign tx1: %v", err)
	}

	// Sign with PermissionId = 2
	tx2 := makeTx()
	setPermissionId(tx2, 2)
	signed2, err := SignTransaction(tx2, privKey)
	if err != nil {
		t.Fatalf("sign tx2: %v", err)
	}

	// Signatures must differ because the hash includes PermissionId
	if bytes.Equal(signed1.Signature[0], signed2.Signature[0]) {
		t.Error("expected different signatures for different PermissionId values")
	}

	// Verify PermissionId is preserved in the signed transaction
	if signed2.GetRawData().GetContract()[0].PermissionId != 2 {
		t.Errorf("expected PermissionId=2 after signing, got %d",
			signed2.GetRawData().GetContract()[0].PermissionId)
	}
}

func TestSignTransactionMultiSig(t *testing.T) {
	key1, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate key1: %v", err)
	}
	key2, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate key2: %v", err)
	}

	tx := newTestTransaction()
	setPermissionId(tx, 2)

	// Capture raw data before signing
	rawBefore, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		t.Fatalf("marshal before signing: %v", err)
	}

	// First signature
	tx, err = SignTransaction(tx, key1)
	if err != nil {
		t.Fatalf("first sign: %v", err)
	}
	if len(tx.Signature) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(tx.Signature))
	}

	// Raw data should be unchanged after first signature
	rawAfterFirst, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		t.Fatalf("marshal after first sign: %v", err)
	}
	if !bytes.Equal(rawBefore, rawAfterFirst) {
		t.Error("raw data changed after first signature")
	}

	// Second signature
	tx, err = SignTransaction(tx, key2)
	if err != nil {
		t.Fatalf("second sign: %v", err)
	}
	if len(tx.Signature) != 2 {
		t.Fatalf("expected 2 signatures, got %d", len(tx.Signature))
	}

	// Raw data should be unchanged after second signature
	rawAfterSecond, err := proto.Marshal(tx.GetRawData())
	if err != nil {
		t.Fatalf("marshal after second sign: %v", err)
	}
	if !bytes.Equal(rawBefore, rawAfterSecond) {
		t.Error("raw data changed after second signature")
	}

	// Both signatures should be different (different keys)
	if bytes.Equal(tx.Signature[0], tx.Signature[1]) {
		t.Error("expected different signatures from different keys")
	}

	// PermissionId should still be set
	if tx.GetRawData().GetContract()[0].PermissionId != 2 {
		t.Error("PermissionId should be preserved after multi-sig signing")
	}
}
