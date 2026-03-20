package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugFromEnv_Defaults(t *testing.T) {
	cfg := DebugFromEnv()
	// Without env vars set, both flags should be false.
	assert.False(t, cfg.GRPC, "GRPC should default to false")
	assert.False(t, cfg.Transaction, "Transaction should default to false")
}

func TestDebugFromEnv_GRPCOnly(t *testing.T) {
	t.Setenv("TRONCTL_GRPC_DEBUG", "1")
	cfg := DebugFromEnv()
	assert.True(t, cfg.GRPC)
	assert.False(t, cfg.Transaction)
}

func TestDebugFromEnv_TxOnly(t *testing.T) {
	t.Setenv("TRONCTL_TX_DEBUG", "1")
	cfg := DebugFromEnv()
	assert.False(t, cfg.GRPC)
	assert.True(t, cfg.Transaction)
}

func TestDebugFromEnv_AllDebug(t *testing.T) {
	t.Setenv("TRONCTL_ALL_DEBUG", "1")
	cfg := DebugFromEnv()
	assert.True(t, cfg.GRPC)
	assert.True(t, cfg.Transaction)
}

func TestDebugConfig_ZeroValue(t *testing.T) {
	var cfg DebugConfig
	assert.False(t, cfg.GRPC)
	assert.False(t, cfg.Transaction)
}

func TestEnableAllVerbose(t *testing.T) {
	// Save original values
	origGRPC := DebugGRPC
	origTx := DebugTransaction
	t.Cleanup(func() {
		DebugGRPC = origGRPC
		DebugTransaction = origTx
	})

	DebugGRPC = false
	DebugTransaction = false

	EnableAllVerbose()

	assert.True(t, DebugGRPC)
	assert.True(t, DebugTransaction)
}
