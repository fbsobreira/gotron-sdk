package signer

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestKeystoreAndAccount(t *testing.T, password string) (*keystore.KeyStore, keystore.Account) {
	t.Helper()
	tmpDir := t.TempDir()
	ks := keystore.NewKeyStore(tmpDir, keystore.LightScryptN, keystore.LightScryptP)
	acct, err := ks.NewAccount(password)
	require.NoError(t, err)
	return ks, acct
}

func TestNewKeystoreSigner_Address(t *testing.T) {
	ks, acct := newTestKeystoreAndAccount(t, "pass")

	s := NewKeystoreSigner(ks, acct)
	addr := s.Address()

	assert.Len(t, addr, 21)
	assert.Equal(t, byte(0x41), addr[0])
	assert.Equal(t, []byte(acct.Address), []byte(addr))
}

func TestKeystoreSigner_Sign_Unlocked(t *testing.T) {
	ks, acct := newTestKeystoreAndAccount(t, "pass")
	require.NoError(t, ks.Unlock(acct, "pass"))

	s := NewKeystoreSigner(ks, acct)
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

func TestKeystoreSigner_Sign_Locked(t *testing.T) {
	ks, acct := newTestKeystoreAndAccount(t, "pass")
	// Account is locked by default after creation

	s := NewKeystoreSigner(ks, acct)
	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x01, 0x02},
		},
	}

	_, err := s.Sign(tx)
	require.Error(t, err)
}

func TestKeystoreSigner_SignDeterministic(t *testing.T) {
	ks, acct := newTestKeystoreAndAccount(t, "pass")
	require.NoError(t, ks.Unlock(acct, "pass"))

	s := NewKeystoreSigner(ks, acct)

	makeTx := func() *core.Transaction {
		return &core.Transaction{
			RawData: &core.TransactionRaw{
				RefBlockBytes: []byte{0xAA, 0xBB},
			},
		}
	}

	tx1, err := s.Sign(makeTx())
	require.NoError(t, err)
	tx2, err := s.Sign(makeTx())
	require.NoError(t, err)

	assert.Equal(t, tx1.Signature[0], tx2.Signature[0],
		"same key + same data should produce the same signature")
}

func TestNewKeystorePassphraseSigner_Address(t *testing.T) {
	ks, acct := newTestKeystoreAndAccount(t, "secret")

	s := NewKeystorePassphraseSigner(ks, acct, "secret")
	addr := s.Address()

	assert.Len(t, addr, 21)
	assert.Equal(t, byte(0x41), addr[0])
	assert.Equal(t, []byte(acct.Address), []byte(addr))
}

func TestKeystorePassphraseSigner_Sign(t *testing.T) {
	ks, acct := newTestKeystoreAndAccount(t, "secret")
	// No unlock needed — passphrase signer decrypts on each sign call

	s := NewKeystorePassphraseSigner(ks, acct, "secret")
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

func TestKeystorePassphraseSigner_Sign_WrongPassphrase(t *testing.T) {
	ks, acct := newTestKeystoreAndAccount(t, "correct")

	s := NewKeystorePassphraseSigner(ks, acct, "wrong")
	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x01, 0x02},
		},
	}

	_, err := s.Sign(tx)
	require.Error(t, err)
}

func TestKeystoreSignersProduceSameSignature(t *testing.T) {
	ks, acct := newTestKeystoreAndAccount(t, "pass")
	require.NoError(t, ks.Unlock(acct, "pass"))

	sSigner := NewKeystoreSigner(ks, acct)
	pSigner := NewKeystorePassphraseSigner(ks, acct, "pass")

	makeTx := func() *core.Transaction {
		return &core.Transaction{
			RawData: &core.TransactionRaw{
				RefBlockBytes: []byte{0x01, 0x02},
			},
		}
	}

	tx1, err := sSigner.Sign(makeTx())
	require.NoError(t, err)
	tx2, err := pSigner.Sign(makeTx())
	require.NoError(t, err)

	assert.Equal(t, tx1.Signature[0], tx2.Signature[0],
		"keystore signer and passphrase signer should produce the same signature for the same key")
}

func TestKeystoreSigner_MultiSign(t *testing.T) {
	ks1, acct1 := newTestKeystoreAndAccount(t, "pass1")
	ks2, acct2 := newTestKeystoreAndAccount(t, "pass2")
	require.NoError(t, ks1.Unlock(acct1, "pass1"))
	require.NoError(t, ks2.Unlock(acct2, "pass2"))

	s1 := NewKeystoreSigner(ks1, acct1)
	s2 := NewKeystoreSigner(ks2, acct2)

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x01, 0x02},
		},
	}

	tx, err := s1.Sign(tx)
	require.NoError(t, err)
	tx, err = s2.Sign(tx)
	require.NoError(t, err)

	assert.Len(t, tx.Signature, 2)
	assert.NotEqual(t, tx.Signature[0], tx.Signature[1],
		"different keys should produce different signatures")
}
