//go:build integration

package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ProposalsList(t *testing.T) {
	c := newIntegrationClient(t)

	result, err := c.ProposalsList()
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotEmpty(t, result.GetProposals(), "testnet should have at least one proposal")

	p := result.GetProposals()[0]
	assert.Greater(t, p.GetProposalId(), int64(0), "proposal should have a positive ID")
	assert.NotEmpty(t, p.GetProposerAddress(), "proposal should have a proposer")
}

func TestIntegration_ProposalCreate(t *testing.T) {
	c := newIntegrationClient(t)

	// Not a witness — expected error.
	params := map[int64]int64{0: 100000}
	_, err := c.ProposalCreate(nileTestAccountAddress, params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_ProposalApprove(t *testing.T) {
	c := newIntegrationClient(t)

	// Not a witness — expected error.
	_, err := c.ProposalApprove(nileTestAccountAddress, 1, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}

func TestIntegration_ProposalWithdraw(t *testing.T) {
	c := newIntegrationClient(t)

	// Not the proposal creator — expected error.
	_, err := c.ProposalWithdraw(nileTestAccountAddress, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Contract validate error")
}
