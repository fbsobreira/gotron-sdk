package cmd

import (
	"errors"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/fbsobreira/gotron-sdk/pkg/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSignerLedger(t *testing.T) {
	prev := useLedgerWallet
	useLedgerWallet = true
	t.Cleanup(func() { useLedgerWallet = prev })

	var addr tronAddress
	require.NoError(t, addr.Set("TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"))

	s, err := resolveSigner(addr)
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Equal(t, addr.GetAddress().String(), s.Address().String())
}

func TestResolveSignerKeystoreError(t *testing.T) {
	prev := useLedgerWallet
	useLedgerWallet = false
	t.Cleanup(func() { useLedgerWallet = prev })

	// A fresh random address is in no local keystore.
	var addr tronAddress
	require.NoError(t, addr.Set("TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"))

	s, err := resolveSigner(addr)
	require.Error(t, err)
	assert.Nil(t, s)

	ctrlr, err := newController(addr, &core.Transaction{})
	require.Error(t, err)
	assert.Nil(t, ctrlr)
}

func TestNewControllerAppliesRuntimeOptions(t *testing.T) {
	prevLedger, prevRT := useLedgerWallet, *rt
	t.Cleanup(func() { useLedgerWallet, *rt = prevLedger, prevRT })

	var addr tronAddress
	require.NoError(t, addr.Set("TJRabPrwbZy45sbavfcjinPJC18kjpRTv8"))

	tests := []struct {
		name     string
		dryRun   bool
		noWait   bool
		timeout  uint32
		wantWait uint32
	}{
		{name: "dry run with timeout", dryRun: true, timeout: 42, wantWait: 42},
		{name: "no wait overrides timeout", noWait: true, timeout: 42, wantWait: 0},
		{name: "defaults", wantWait: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useLedgerWallet = true
			rt.DryRun, rt.NoWait, rt.Timeout = tt.dryRun, tt.noWait, tt.timeout

			ctrlr, err := newController(addr, &core.Transaction{})
			require.NoError(t, err)
			require.NotNil(t, ctrlr)
			assert.Equal(t, tt.dryRun, ctrlr.Behavior.DryRun)
			assert.Equal(t, tt.wantWait, ctrlr.Behavior.ConfirmationWaitTime)
		})
	}
}

// TestKeystoreSigningLifetime checks that the keystore resolveSigner unlocks
// stays usable until the command ends and is closed, its key zeroed, after.
func TestKeystoreSigningLifetime(t *testing.T) {
	pqTestEnv(t, "pw")
	sk, err := keys.GetPrivateKeyFromHex(runtimeKSTestKey)
	require.NoError(t, err)
	ks := store.FromAccountName("ks-life")
	acct, err := ks.ImportECDSA(sk.ToECDSA(), "pw")
	require.NoError(t, err)
	ks.Close()

	var a tronAddress
	require.NoError(t, a.Set(acct.Address.String()))

	s, err := resolveSigner(a)
	require.NoError(t, err)
	signed, err := s.Sign(&core.Transaction{RawData: &core.TransactionRaw{Timestamp: 1}})
	require.NoError(t, err, "the keystore must stay unlocked until the command ends")
	require.NotNil(t, signed)

	RootCmd.PersistentPostRun(RootCmd, nil)
	_, err = s.Sign(&core.Transaction{RawData: &core.TransactionRaw{Timestamp: 2}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, keystore.ErrLocked), "got %v", err)

	// Idempotent.
	releaseSigningKeys()
}
