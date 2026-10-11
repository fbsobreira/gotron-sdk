package txcore

import (
	"context"
	"errors"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pqSigner is a fake signer that reports a post-quantum scheme.
type pqSigner struct {
	scheme core.PQScheme
	signed int
}

func (s *pqSigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	s.signed++
	tx.Signature = append(tx.Signature, []byte("fakepqsig"))
	return tx, nil
}

func (s *pqSigner) Address() address.Address { return nil }

func (s *pqSigner) PQScheme() core.PQScheme { return s.scheme }

// countingSigner is an ECDSA-style signer without a PQScheme method.
type countingSigner struct {
	signed int
}

func (s *countingSigner) Sign(tx *core.Transaction) (*core.Transaction, error) {
	s.signed++
	tx.Signature = append(tx.Signature, []byte("fakesig"))
	return tx, nil
}

func (s *countingSigner) Address() address.Address { return nil }

// pqBroadcaster is a Broadcaster that also implements the capability probe.
type pqBroadcaster struct {
	mockBroadcaster
	allowed    bool
	probeErr   error
	probeCalls int
	probed     []core.PQScheme
	broadcasts int
}

func (b *pqBroadcaster) PQSchemeAllowedCtx(_ context.Context, scheme core.PQScheme) (bool, error) {
	b.probeCalls++
	b.probed = append(b.probed, scheme)
	return b.allowed, b.probeErr
}

func (b *pqBroadcaster) BroadcastCtx(_ context.Context, _ *core.Transaction) (*api.Return, error) {
	b.broadcasts++
	return &api.Return{Result: true}, nil
}

// plainBroadcaster is a Broadcaster without the probe method.
type plainBroadcaster struct {
	mockBroadcaster
	broadcasts int
}

func (b *plainBroadcaster) BroadcastCtx(_ context.Context, _ *core.Transaction) (*api.Return, error) {
	b.broadcasts++
	return &api.Return{Result: true}, nil
}

func TestSend_PQ_NonPQSignerSkipsProbe(t *testing.T) {
	b := &pqBroadcaster{allowed: false}
	s := &countingSigner{}

	receipt, err := Send(context.Background(), b, s, newDummyTx())
	require.NoError(t, err)
	assert.NotEmpty(t, receipt.TxID)
	assert.Equal(t, 0, b.probeCalls)
	assert.Equal(t, 1, s.signed)
	assert.Equal(t, 1, b.broadcasts)
}

func TestSend_PQ_Allowed(t *testing.T) {
	b := &pqBroadcaster{allowed: true}
	s := &pqSigner{scheme: core.PQScheme_FN_DSA_512}

	receipt, err := Send(context.Background(), b, s, newDummyTx())
	require.NoError(t, err)
	assert.NotEmpty(t, receipt.TxID)
	assert.Equal(t, []core.PQScheme{core.PQScheme_FN_DSA_512}, b.probed)
	assert.Equal(t, 1, s.signed)
	assert.Equal(t, 1, b.broadcasts)
}

func TestSend_PQ_Disabled(t *testing.T) {
	b := &pqBroadcaster{allowed: false}
	s := &pqSigner{scheme: core.PQScheme_ML_DSA_44}

	receipt, err := Send(context.Background(), b, s, newDummyTx())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPQSchemeDisabled)
	assert.Contains(t, err.Error(), "ML_DSA_44")
	assert.Nil(t, receipt)
	assert.Equal(t, 1, b.probeCalls)
	assert.Equal(t, 0, s.signed, "must not sign when the scheme is disabled")
	assert.Equal(t, 0, b.broadcasts, "must not broadcast when the scheme is disabled")
}

func TestSend_PQ_ProbeError(t *testing.T) {
	probeErr := errors.New("node unreachable")
	b := &pqBroadcaster{probeErr: probeErr, allowed: true}
	s := &pqSigner{scheme: core.PQScheme_FN_DSA_512}

	receipt, err := Send(context.Background(), b, s, newDummyTx())
	require.Error(t, err)
	assert.ErrorIs(t, err, probeErr)
	assert.NotErrorIs(t, err, ErrPQSchemeDisabled)
	assert.Contains(t, err.Error(), "checking post-quantum capability")
	assert.Nil(t, receipt)
	assert.Equal(t, 0, s.signed)
	assert.Equal(t, 0, b.broadcasts)
}

func TestSend_PQ_BroadcasterWithoutProbe(t *testing.T) {
	b := &plainBroadcaster{}
	s := &pqSigner{scheme: core.PQScheme_FN_DSA_512}

	receipt, err := Send(context.Background(), b, s, newDummyTx())
	require.NoError(t, err)
	assert.NotEmpty(t, receipt.TxID)
	assert.Equal(t, 1, s.signed)
	assert.Equal(t, 1, b.broadcasts)
}

func TestSendAndConfirm_PQ_Disabled(t *testing.T) {
	b := &pqBroadcaster{allowed: false}
	s := &pqSigner{scheme: core.PQScheme_FN_DSA_512}

	_, err := SendAndConfirm(context.Background(), b, s, newDummyTx(), 0)
	assert.ErrorIs(t, err, ErrPQSchemeDisabled)
	assert.Equal(t, 0, s.signed)
	assert.Equal(t, 0, b.broadcasts)
}
