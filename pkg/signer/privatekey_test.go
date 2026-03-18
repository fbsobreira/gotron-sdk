package signer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrivateKeySigner(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySigner(key)
	require.NoError(t, err)

	addr := s.Address()
	assert.Len(t, addr, 21)
	assert.Equal(t, byte(0x41), addr[0])
}

func TestNewPrivateKeySignerFromBTCEC(t *testing.T) {
	key, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySignerFromBTCEC(key)
	require.NoError(t, err)

	addr := s.Address()
	assert.Len(t, addr, 21)
	assert.Equal(t, byte(0x41), addr[0])
}

func TestPrivateKeySigner_Sign(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySigner(key)
	require.NoError(t, err)

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x01, 0x02},
		},
	}

	signed, err := s.Sign(tx)
	require.NoError(t, err)
	assert.Len(t, signed.Signature, 1)
	assert.Len(t, signed.Signature[0], 65) // secp256k1 signature
}

func TestPrivateKeySigner_SignMultiple(t *testing.T) {
	key1, err := crypto.GenerateKey()
	require.NoError(t, err)
	key2, err := crypto.GenerateKey()
	require.NoError(t, err)

	s1, err := NewPrivateKeySigner(key1)
	require.NoError(t, err)
	s2, err := NewPrivateKeySigner(key2)
	require.NoError(t, err)

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x01, 0x02},
		},
	}

	tx, err = s1.Sign(tx)
	require.NoError(t, err)
	tx, err = s2.Sign(tx)
	require.NoError(t, err)

	assert.Len(t, tx.Signature, 2)
}

func TestNewPrivateKeySigner_RejectsP256(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	_, err = NewPrivateKeySigner(key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported curve")
}
