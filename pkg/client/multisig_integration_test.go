//go:build integration

package client_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client/transaction"
	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_MultiSig_SetPermissionIdChangesHash verifies that
// SetPermissionId modifies the transaction hash — a key invariant
// documented in the multi-sig guide's "Common Pitfalls" section.
func TestIntegration_MultiSig_SetPermissionIdChangesHash(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.Transfer(nileTestAccountAddress, nileTestAddress2, 1)
	require.NoError(t, err)
	require.NotNil(t, tx)

	origTxid := hex.EncodeToString(tx.GetTxid())
	require.NotEmpty(t, origTxid)

	err = tx.SetPermissionId(2)
	require.NoError(t, err)

	newTxid := hex.EncodeToString(tx.GetTxid())
	assert.NotEqual(t, origTxid, newTxid,
		"SetPermissionId must change the transaction hash")

	for _, contract := range tx.GetTransaction().GetRawData().GetContract() {
		assert.Equal(t, int32(2), contract.PermissionId)
	}
}

// TestIntegration_MultiSig_SignWeightUnsigned builds a transfer and checks
// GetTransactionSignWeight on an unsigned transaction — the network should
// report CurrentWeight=0 since no signatures have been added.
func TestIntegration_MultiSig_SignWeightUnsigned(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.Transfer(nileTestAccountAddress, nileTestAddress2, 1)
	require.NoError(t, err)

	weight, err := c.GetTransactionSignWeight(tx.GetTransaction())
	require.NoError(t, err)
	require.NotNil(t, weight)

	assert.NotNil(t, weight.GetPermission())
	assert.Greater(t, weight.GetPermission().GetThreshold(), int64(0))
	assert.Equal(t, int64(0), weight.GetCurrentWeight(),
		"unsigned transaction should have zero weight")
	assert.Empty(t, weight.GetApprovedList(),
		"unsigned transaction should have no approved signers")
}

// TestIntegration_MultiSig_SignWeightAfterSigning builds a transfer, signs it
// with a random key, and verifies GetTransactionSignWeight accepts the
// signed transaction without error.
func TestIntegration_MultiSig_SignWeightAfterSigning(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.Transfer(nileTestAccountAddress, nileTestAddress2, 1)
	require.NoError(t, err)

	randomKey, err := keys.GenerateKey()
	require.NoError(t, err)

	signedTx, err := transaction.SignTransaction(tx.GetTransaction(), randomKey)
	require.NoError(t, err)
	assert.Len(t, signedTx.GetSignature(), 1)

	weight, err := c.GetTransactionSignWeight(signedTx)
	require.NoError(t, err)
	require.NotNil(t, weight)
	require.NotNil(t, weight.GetPermission())
}

// TestIntegration_MultiSig_SequentialSigning verifies that multiple signatures
// can be added to a transaction sequentially and the network can parse the
// result, as documented in the multi-sig workflow guide.
func TestIntegration_MultiSig_SequentialSigning(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.Transfer(nileTestAccountAddress, nileTestAddress2, 1)
	require.NoError(t, err)

	err = tx.SetPermissionId(2)
	require.NoError(t, err)

	key1, err := keys.GenerateKey()
	require.NoError(t, err)
	key2, err := keys.GenerateKey()
	require.NoError(t, err)

	signedTx, err := transaction.SignTransaction(tx.GetTransaction(), key1)
	require.NoError(t, err)
	assert.Len(t, signedTx.GetSignature(), 1)

	signedTx, err = transaction.SignTransaction(signedTx, key2)
	require.NoError(t, err)
	assert.Len(t, signedTx.GetSignature(), 2)

	weight, err := c.GetTransactionSignWeight(signedTx)
	require.NoError(t, err)
	require.NotNil(t, weight)
}

// TestIntegration_MultiSig_RawDataHexRoundTrip builds a real transaction,
// serializes to raw_data_hex, reconstructs it, signs, and validates via
// GetTransactionSignWeight. Exercises the FromRawDataHex workflow for
// signing externally-built transactions.
func TestIntegration_MultiSig_RawDataHexRoundTrip(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.Transfer(nileTestAccountAddress, nileTestAddress2, 1)
	require.NoError(t, err)

	rawHex, err := transaction.ToRawDataHex(tx.GetTransaction())
	require.NoError(t, err)
	assert.NotEmpty(t, rawHex)

	reconstructed, err := transaction.FromRawDataHex(rawHex)
	require.NoError(t, err)
	require.NotNil(t, reconstructed.GetRawData())

	reEncodedHex, err := transaction.ToRawDataHex(reconstructed)
	require.NoError(t, err)
	assert.Equal(t, rawHex, reEncodedHex, "round-trip should produce identical raw_data_hex")

	key, err := keys.GenerateKey()
	require.NoError(t, err)

	signedTx, err := transaction.SignTransaction(reconstructed, key)
	require.NoError(t, err)
	assert.Len(t, signedTx.GetSignature(), 1)

	weight, err := c.GetTransactionSignWeight(signedTx)
	require.NoError(t, err)
	require.NotNil(t, weight)
	require.NotNil(t, weight.GetPermission())
}

// TestIntegration_MultiSig_JSONRoundTrip builds a real transaction,
// serializes to JSON, reconstructs, and verifies raw_data is preserved.
func TestIntegration_MultiSig_JSONRoundTrip(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.Transfer(nileTestAccountAddress, nileTestAddress2, 1)
	require.NoError(t, err)

	jsonData, err := transaction.ToJSON(tx.GetTransaction())
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	reconstructed, err := transaction.FromJSON(jsonData)
	require.NoError(t, err)
	require.NotNil(t, reconstructed.GetRawData())

	origHex, err := transaction.ToRawDataHex(tx.GetTransaction())
	require.NoError(t, err)
	reconHex, err := transaction.ToRawDataHex(reconstructed)
	require.NoError(t, err)
	assert.Equal(t, origHex, reconHex, "JSON round-trip should preserve raw_data")
}

// TestIntegration_MultiSig_UpdateAccountPermission verifies that
// UpdateAccountPermission produces a valid multi-sig permission update
// transaction with multiple keys and a threshold > 1.
func TestIntegration_MultiSig_UpdateAccountPermission(t *testing.T) {
	c := newIntegrationClient(t)

	owner := map[string]any{
		"threshold": int64(1),
		"keys": map[string]int64{
			nileTestAccountAddress: 1,
		},
	}

	actives := []map[string]any{
		{
			"name":      "multi-sig-ops",
			"threshold": int64(2),
			"operations": map[string]bool{
				"TransferContract": true,
			},
			"keys": map[string]int64{
				nileTestAccountAddress: 1,
				nileTestAddress2:       1,
				nileTestAddress:        1,
			},
		},
	}

	tx, err := c.UpdateAccountPermission(nileTestAccountAddress, owner, nil, actives)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid())
	assert.NotNil(t, tx.GetTransaction().GetRawData())
}

// TestIntegration_MultiSig_TRC20Transfer builds a TRC20 transfer via
// TriggerContract, sets PermissionId, and signs with two keys. This
// validates the multi-sig TRC20 example from the docs.
func TestIntegration_MultiSig_TRC20Transfer(t *testing.T) {
	c := newIntegrationClient(t)

	method := "transfer(address,uint256)"
	params := fmt.Sprintf(`[{"address": "%s"}, {"uint256": "1"}]`, nileTestAddress2)

	tx, err := c.TriggerContract(
		nileTestAccountAddress,
		nileUSDTContract,
		method,
		params,
		10_000_000,
		0, "", 0,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid())

	err = tx.SetPermissionId(2)
	require.NoError(t, err)

	key1, err := keys.GenerateKey()
	require.NoError(t, err)
	key2, err := keys.GenerateKey()
	require.NoError(t, err)

	signedTx, err := transaction.SignTransaction(tx.GetTransaction(), key1)
	require.NoError(t, err)
	signedTx, err = transaction.SignTransaction(signedTx, key2)
	require.NoError(t, err)
	assert.Len(t, signedTx.GetSignature(), 2)
}

// TestIntegration_MultiSig_DelegateResource builds a resource delegation
// transaction, sets PermissionId for multi-sig, and signs with two keys.
func TestIntegration_MultiSig_DelegateResource(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.DelegateResource(
		nileTestAccountAddress,
		nileTestAddress2,
		core.ResourceCode_ENERGY,
		1_000_000,
		false,
		0,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)

	err = tx.SetPermissionId(2)
	require.NoError(t, err)

	for _, contract := range tx.GetTransaction().GetRawData().GetContract() {
		assert.Equal(t, int32(2), contract.PermissionId)
	}

	key1, err := keys.GenerateKey()
	require.NoError(t, err)
	key2, err := keys.GenerateKey()
	require.NoError(t, err)

	signedTx, err := transaction.SignTransaction(tx.GetTransaction(), key1)
	require.NoError(t, err)
	signedTx, err = transaction.SignTransaction(signedTx, key2)
	require.NoError(t, err)
	assert.Len(t, signedTx.GetSignature(), 2)

	assert.NotEmpty(t, tx.GetTxid())
}
