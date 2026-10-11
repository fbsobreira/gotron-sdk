package cmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// EstimateEnergy answers with an energy figure and no Result, which the
// client accepts as success (a nil Result reads as code 0).
func (w *dryRunWallet) EstimateEnergy(context.Context, *core.TriggerSmartContract) (*api.EstimateEnergyMessage, error) {
	return &api.EstimateEnergyMessage{EnergyRequired: 1234}, nil
}

// TestContractTriggerEstimateNilResult runs `contract trigger --estimate`
// against a node that omits Result. The command must print the estimate
// instead of dereferencing the nil Result, and must not sign or broadcast.
func TestContractTriggerEstimateNilResult(t *testing.T) {
	const pw = "estimate-pass"
	pqTestEnv(t, pw)
	from := addECDSAAccount(t, "estimate-ecdsa", pw)
	wallet := &dryRunWallet{}
	withDryRunRuntime(t, newDryRunClient(t, wallet), from, false, false)

	cmd := contractTriggerCmd()
	prevEstimate := estimate
	t.Cleanup(func() { estimate = prevEstimate })
	estimate = true

	var (
		out string
		err error
	)
	require.NotPanics(t, func() {
		out, _, err = runKeysCmd(t, cmd, "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", "ping()")
	})
	require.NoError(t, err)
	assert.Zero(t, wallet.broadcastCalls.Load(), "--estimate must not broadcast")

	var got struct {
		EnergyRequired int64 `json:"EnergyRequired"`
		Result         struct {
			Code string `json:"code"`
		} `json:"result"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got), "stdout must be the estimate JSON:\n%s", out)
	assert.Equal(t, int64(1234), got.EnergyRequired)
	assert.Equal(t, "SUCCESS", got.Result.Code)
}
