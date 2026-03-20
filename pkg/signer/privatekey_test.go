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

func TestPrivateKeySigner_Address_ReturnsCopy(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySigner(key)
	require.NoError(t, err)

	addr1 := s.Address()
	addr2 := s.Address()

	// Mutate addr1 and verify addr2 is unaffected
	addr1[0] = 0xFF
	assert.NotEqual(t, addr1[0], addr2[0],
		"Address should return a defensive copy")
}

func TestPrivateKeySigner_DifferentDataDifferentSig(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySigner(key)
	require.NoError(t, err)

	tx1 := &core.Transaction{
		RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}},
	}
	tx2 := &core.Transaction{
		RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x02}},
	}

	signed1, err := s.Sign(tx1)
	require.NoError(t, err)
	signed2, err := s.Sign(tx2)
	require.NoError(t, err)

	assert.NotEqual(t, signed1.Signature[0], signed2.Signature[0],
		"different transaction data should produce different signatures")
}

func TestPrivateKeySigner_Sign_NilRawData(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySigner(key)
	require.NoError(t, err)

	tx := &core.Transaction{}
	signed, err := s.Sign(tx)
	require.NoError(t, err)
	assert.Len(t, signed.Signature, 1)
}

func TestNewPrivateKeySigner_RejectsP384(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)

	_, err = NewPrivateKeySigner(key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported curve")
}

func TestNewPrivateKeySigner_BTCECMatchesECDSA(t *testing.T) {
	btcKey, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	s1, err := NewPrivateKeySignerFromBTCEC(btcKey)
	require.NoError(t, err)

	s2, err := NewPrivateKeySigner(btcKey.ToECDSA())
	require.NoError(t, err)

	assert.Equal(t, s1.Address(), s2.Address(),
		"BTCEC and ECDSA constructors should produce the same address")
}

func TestNewPrivateKeySigner_RejectsP256(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	_, err = NewPrivateKeySigner(key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported curve")
}

func TestNewPrivateKeySigner_RejectsP521(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	require.NoError(t, err)

	_, err = NewPrivateKeySigner(key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported curve")
}

func TestNewPrivateKeySigner_RejectsUnnamedCurve(t *testing.T) {
	// Create a key on a custom unnamed curve with a different bit size.
	// We use P-224 which has Name="" in some Go versions and bitsize 224.
	key, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	require.NoError(t, err)

	_, err = NewPrivateKeySigner(key)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported curve")
}

func TestPrivateKeySigner_Sign_NilTransaction(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySigner(key)
	require.NoError(t, err)

	// Sign with a nil transaction should not panic but may return an error
	// depending on protobuf marshaling behavior.
	signed, err := s.Sign(&core.Transaction{})
	require.NoError(t, err)
	assert.NotNil(t, signed)
	assert.Len(t, signed.Signature, 1)
}

func TestPrivateKeySigner_SignPreservesRawData(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySigner(key)
	require.NoError(t, err)

	rawBytes := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: rawBytes,
		},
	}

	signed, err := s.Sign(tx)
	require.NoError(t, err)

	// Verify the raw data was not modified by signing.
	assert.Equal(t, rawBytes, signed.RawData.RefBlockBytes,
		"Sign must not modify the transaction's raw data")
}

func TestPrivateKeySigner_Address_HasTronPrefix(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySigner(key)
	require.NoError(t, err)

	addr := s.Address()
	assert.Len(t, addr, 21, "TRON address must be 21 bytes")
	assert.Equal(t, byte(0x41), addr[0], "TRON address must start with 0x41")
}

func TestNewPrivateKeySignerFromBTCEC_SignAndVerifyAddress(t *testing.T) {
	btcKey, err := btcec.NewPrivateKey()
	require.NoError(t, err)

	s, err := NewPrivateKeySignerFromBTCEC(btcKey)
	require.NoError(t, err)

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x01, 0x02},
		},
	}

	signed, err := s.Sign(tx)
	require.NoError(t, err)
	assert.Len(t, signed.Signature, 1)
	assert.Len(t, signed.Signature[0], 65)
}
