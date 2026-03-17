//go:build integration

package client_test

import (
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestIntegration_GetContractABI(t *testing.T) {
	c := newIntegrationClient(t)

	abi, err := c.GetContractABI(nileUSDTContract)
	require.NoError(t, err)
	require.NotNil(t, abi)
	require.NotEmpty(t, abi.GetEntrys(), "USDT ABI should have entries")

	// Verify well-known methods exist in the ABI.
	methods := make(map[string]bool)
	for _, entry := range abi.GetEntrys() {
		methods[entry.GetName()] = true
	}
	assert.True(t, methods["transfer"], "ABI should contain transfer method")
	assert.True(t, methods["balanceOf"], "ABI should contain balanceOf method")
	assert.True(t, methods["approve"], "ABI should contain approve method")
}

func TestIntegration_GetContractABIResolved_NonProxy(t *testing.T) {
	c := newIntegrationClient(t)

	// USDT is not a proxy — ABI should be returned directly.
	abi, err := c.GetContractABIResolved(nileUSDTContract)
	require.NoError(t, err)
	require.NotNil(t, abi)
	require.NotEmpty(t, abi.GetEntrys(), "non-proxy ABI should have entries")
}

func TestIntegration_GetContractABIResolved_InvalidAddress(t *testing.T) {
	c := newIntegrationClient(t)

	_, err := c.GetContractABIResolved("invalid-address")
	require.Error(t, err)
}

func TestIntegration_GetContractABIResolved_Proxy(t *testing.T) {
	// ERC-1967 proxy on mainnet whose implementation stores its ABI on-chain.
	// Proxy:          T9yDMyUdQDTVKuMANP6S6zMFCnC6wZtVML  (0 ABI entries)
	// Implementation: TJdimGcAMywGfryiVsGFQsk1Uo58oi5s8x  (30 ABI entries)
	const (
		mainnetEndpoint = "grpc.trongrid.io:50051"
		proxyContract   = "T9yDMyUdQDTVKuMANP6S6zMFCnC6wZtVML"
	)

	mc := client.NewGrpcClientWithTimeout(mainnetEndpoint, 30*time.Second)
	err := mc.Start(grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Skipf("cannot connect to mainnet at %s: %v", mainnetEndpoint, err)
	}
	t.Cleanup(mc.Stop)

	// Plain GetContractABI should return an empty ABI for the proxy.
	directABI, err := mc.GetContractABI(proxyContract)
	if err != nil && strings.Contains(err.Error(), "429") {
		t.Skip("rate-limited by TronGrid, skipping")
	}
	require.NoError(t, err)
	assert.Empty(t, directABI.GetEntrys(), "proxy contract should have empty ABI")

	// Brief pause to avoid TronGrid rate limits between calls.
	time.Sleep(1 * time.Second)

	// GetContractABIResolved should resolve through to the implementation ABI.
	resolved, err := mc.GetContractABIResolved(proxyContract)
	if err != nil && strings.Contains(err.Error(), "429") {
		t.Skip("rate-limited by TronGrid, skipping")
	}
	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.NotEmpty(t, resolved.GetEntrys(),
		"resolved proxy ABI should contain implementation entries")

	// Verify well-known methods from the implementation ABI.
	methods := make(map[string]bool)
	for _, entry := range resolved.GetEntrys() {
		methods[entry.GetName()] = true
	}
	assert.True(t, methods["execute"], "resolved ABI should contain execute()")
	assert.True(t, methods["owner"], "resolved ABI should contain owner()")
}

func TestIntegration_TriggerConstantContract(t *testing.T) {
	c := newIntegrationClient(t)

	// Call totalSupply() on the USDT contract — a read-only call.
	result, err := c.TriggerConstantContract(
		"",
		nileUSDTContract,
		"totalSupply()",
		"[]",
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.GetConstantResult(), "totalSupply() should return a result")
	assert.Len(t, result.GetConstantResult()[0], 32, "totalSupply() should return a 32-byte uint256")
}

func TestIntegration_DecodeOutput(t *testing.T) {
	c := newIntegrationClient(t)

	// Fetch the USDT contract ABI from chain.
	contractABI, err := c.GetContractABI(nileUSDTContract)
	require.NoError(t, err)
	require.NotNil(t, contractABI)

	t.Run("totalSupply_uint256", func(t *testing.T) {
		tx, err := c.TriggerConstantContract("", nileUSDTContract, "totalSupply()", "[]")
		require.NoError(t, err)
		require.NotEmpty(t, tx.GetConstantResult())

		values, err := abi.DecodeOutput(contractABI, "totalSupply", tx.GetConstantResult()[0])
		require.NoError(t, err)
		require.Len(t, values, 1)

		supply, ok := values[0].(*big.Int)
		require.True(t, ok, "expected *big.Int, got %T", values[0])
		assert.True(t, supply.Sign() > 0, "totalSupply should be positive, got %s", supply)
		t.Logf("totalSupply = %s", supply)
	})

	t.Run("name_string", func(t *testing.T) {
		tx, err := c.TriggerConstantContract("", nileUSDTContract, "name()", "[]")
		require.NoError(t, err)
		require.NotEmpty(t, tx.GetConstantResult())

		values, err := abi.DecodeOutput(contractABI, "name", tx.GetConstantResult()[0])
		require.NoError(t, err)
		require.Len(t, values, 1)

		name, ok := values[0].(string)
		require.True(t, ok, "expected string, got %T", values[0])
		assert.NotEmpty(t, name, "name should not be empty")
		t.Logf("name = %q", name)
	})

	t.Run("decimals_uint8", func(t *testing.T) {
		tx, err := c.TriggerConstantContract("", nileUSDTContract, "decimals()", "[]")
		require.NoError(t, err)
		require.NotEmpty(t, tx.GetConstantResult())

		values, err := abi.DecodeOutput(contractABI, "decimals", tx.GetConstantResult()[0])
		require.NoError(t, err)
		require.Len(t, values, 1)

		decimals, ok := values[0].(uint8)
		require.True(t, ok, "expected uint8, got %T", values[0])
		assert.Equal(t, uint8(6), decimals, "USDT should have 6 decimals")
		t.Logf("decimals = %d", decimals)
	})

	t.Run("owner_address", func(t *testing.T) {
		tx, err := c.TriggerConstantContract("", nileUSDTContract, "owner()", "[]")
		require.NoError(t, err)
		require.NotEmpty(t, tx.GetConstantResult())

		values, err := abi.DecodeOutput(contractABI, "owner", tx.GetConstantResult()[0])
		require.NoError(t, err)
		require.Len(t, values, 1)

		ownerAddr, ok := values[0].(address.Address)
		require.True(t, ok, "expected address.Address, got %T", values[0])
		assert.Equal(t, byte(0x41), ownerAddr[0], "should be TRON address")
		t.Logf("owner = %s", ownerAddr.String())
	})
}

func TestIntegration_TriggerContract(t *testing.T) {
	c := newIntegrationClient(t)

	tx, err := c.TriggerContract(
		nileTestAccountAddress,
		nileUSDTContract,
		"transfer(address,uint256)",
		`[{"address": "`+nileTestAddress2+`"}, {"uint256": "1"}]`,
		10000000,
		0, "", 0,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "TriggerContract should produce a transaction ID")
	assert.NotNil(t, tx.GetTransaction().GetRawData(), "transaction should have raw data")
}

func TestIntegration_UpdateEnergyLimitContract(t *testing.T) {
	c := newIntegrationClient(t)

	// Not the contract owner — expected error.
	_, err := c.UpdateEnergyLimitContract(nileTestAccountAddress, nileUSDTContract, 10000000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not the owner of the contract")
}

func TestIntegration_UpdateSettingContract(t *testing.T) {
	c := newIntegrationClient(t)

	// Not the contract owner — expected error.
	_, err := c.UpdateSettingContract(nileTestAccountAddress, nileUSDTContract, 50)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not the owner of the contract")
}

func TestIntegration_EstimateEnergy(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.EstimateEnergy(
		nileTestAccountAddress,
		nileUSDTContract,
		"balanceOf(address)",
		`[{"address": "`+nileTestAccountAddress+`"}]`,
		0, "", 0,
	)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Greater(t, result.GetEnergyRequired(), int64(0), "energy estimate should be positive")
}

func TestIntegration_EstimateEnergyNotSupported(t *testing.T) {
	endpoint := os.Getenv("TRON_UNSUPPORTED_ENDPOINT")
	if endpoint == "" {
		t.Skip("TRON_UNSUPPORTED_ENDPOINT not set — skipping unsupported-node test")
	}

	c := client.NewGrpcClientWithTimeout(endpoint, 30*time.Second)
	err := c.Start(grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "failed to connect to %s", endpoint)
	t.Cleanup(c.Stop)

	_, err = c.EstimateEnergy(
		nileTestAccountAddress,
		nileUSDTContract,
		"balanceOf(address)",
		`[{"address": "`+nileTestAccountAddress+`"}]`,
		0, "", 0,
	)
	require.Error(t, err)
	assert.True(t, errors.Is(err, client.ErrEstimateEnergyNotSupported),
		"expected ErrEstimateEnergyNotSupported, got: %v", err)
}

func TestIntegration_DeployContract(t *testing.T) {
	c := newIntegrationClient(t)

	// Minimal contract bytecode (PUSH1 0x00 PUSH1 0x00 RETURN).
	minimalBytecode := "0x60006000f3"

	tx, err := c.DeployContract(
		nileTestAccountAddress,
		"TestContract",
		nil,
		minimalBytecode,
		100000000,
		0, 1,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetTxid(), "DeployContract should produce a transaction ID")
}
