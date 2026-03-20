package keystore

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestWallet(t *testing.T, password string) (*keystoreWallet, *KeyStore, Account) {
	t.Helper()
	tmpDir := t.TempDir()
	ks := NewKeyStore(tmpDir, LightScryptN, LightScryptP)
	acct, err := ks.NewAccount(password)
	require.NoError(t, err)

	wallet := &keystoreWallet{
		account:  acct,
		keystore: ks,
	}
	return wallet, ks, acct
}

func TestKeystoreWallet_Status_ReflectsLockState(t *testing.T) {
	w, ks, acct := newTestWallet(t, "pass")

	status, err := w.Status()
	require.NoError(t, err)
	assert.Equal(t, "Locked", status)

	require.NoError(t, ks.Unlock(acct, "pass"))
	status, err = w.Status()
	require.NoError(t, err)
	assert.Equal(t, "Unlocked", status)
}

func TestKeystoreWallet_Contains_MatchesOnAddressAndURL(t *testing.T) {
	w, _, acct := newTestWallet(t, "pass")

	// Exact match
	assert.True(t, w.Contains(acct))

	// Empty URL matches any account with the same address
	assert.True(t, w.Contains(Account{Address: acct.Address}))

	// Wrong URL with right address should NOT match
	wrongURL := Account{Address: acct.Address, URL: URL{Scheme: "keystore", Path: "/wrong/path"}}
	assert.False(t, w.Contains(wrongURL))

	// Different address should not match
	otherAddr := make([]byte, 21)
	otherAddr[0] = 0x41
	otherAddr[20] = 0x01
	assert.False(t, w.Contains(Account{Address: otherAddr}))
}

func TestKeystoreWallet_SignData_ProducesRecoverableSignature(t *testing.T) {
	w, ks, acct := newTestWallet(t, "pass")
	require.NoError(t, ks.Unlock(acct, "pass"))

	data := []byte("hello tron")
	sig, err := w.SignData(acct, "text/plain", data)
	require.NoError(t, err)
	assert.Len(t, sig, 65)

	// Recover the public key from the signature and verify it matches the account
	hash := crypto.Keccak256(data)
	pubBytes, err := crypto.Ecrecover(hash, sig)
	require.NoError(t, err)

	pubKey, err := UnmarshalPublic(pubBytes)
	require.NoError(t, err)
	recoveredAddr := address.PubkeyToAddress(*pubKey)
	assert.Equal(t, acct.Address, recoveredAddr,
		"recovered address should match the signing account")
}

func TestKeystoreWallet_SignData_RejectsUnknownAccount(t *testing.T) {
	w, ks, acct := newTestWallet(t, "pass")
	require.NoError(t, ks.Unlock(acct, "pass"))

	otherAddr := make([]byte, 21)
	otherAddr[0] = 0x41
	otherAddr[20] = 0x01
	_, err := w.SignData(Account{Address: otherAddr}, "text/plain", []byte("hello"))
	assert.ErrorIs(t, err, ErrUnknownAccount)
}

func TestKeystoreWallet_SignTx_ProducesValidSignature(t *testing.T) {
	w, ks, acct := newTestWallet(t, "pass")
	require.NoError(t, ks.Unlock(acct, "pass"))

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0x01, 0x02},
		},
	}

	signed, err := w.SignTx(acct, tx)
	require.NoError(t, err)
	require.Len(t, signed.Signature, 1)
	assert.Len(t, signed.Signature[0], 65, "secp256k1 signature must be 65 bytes")
}

func TestKeystoreWallet_SignTx_RejectsUnknownAccount(t *testing.T) {
	w, ks, acct := newTestWallet(t, "pass")
	require.NoError(t, ks.Unlock(acct, "pass"))

	tx := &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}}}
	otherAddr := make([]byte, 21)
	otherAddr[0] = 0x41
	otherAddr[20] = 0x01
	_, err := w.SignTx(Account{Address: otherAddr}, tx)
	assert.ErrorIs(t, err, ErrUnknownAccount)
}

func TestKeystoreWallet_SignTxWithPassphrase_DecryptsAndSigns(t *testing.T) {
	w, _, acct := newTestWallet(t, "secret")

	tx := &core.Transaction{
		RawData: &core.TransactionRaw{
			RefBlockBytes: []byte{0xAA, 0xBB},
		},
	}

	// Should succeed — passphrase decrypts the key each time
	signed, err := w.SignTxWithPassphrase(acct, "secret", tx)
	require.NoError(t, err)
	require.Len(t, signed.Signature, 1)
	assert.Len(t, signed.Signature[0], 65)

	// Wrong passphrase should fail
	tx2 := &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}}}
	_, err = w.SignTxWithPassphrase(acct, "wrong", tx2)
	assert.Error(t, err)
}

func TestKeystoreWallet_Open_IsNoop(t *testing.T) {
	w, _, _ := newTestWallet(t, "pass")
	err := w.Open("any-passphrase")
	assert.NoError(t, err, "Open should be a no-op and always succeed")
}

func TestKeystoreWallet_Close_IsNoop(t *testing.T) {
	w, _, _ := newTestWallet(t, "pass")
	err := w.Close()
	assert.NoError(t, err, "Close should be a no-op and always succeed")
}

func TestKeystoreWallet_Derive_ReturnsErrNotSupported(t *testing.T) {
	w, _, _ := newTestWallet(t, "pass")
	_, err := w.Derive(DerivationPath{0, 1, 2}, false)
	assert.ErrorIs(t, err, ErrNotSupported)
}

func TestKeystoreWallet_SignDataWithPassphrase(t *testing.T) {
	w, _, acct := newTestWallet(t, "pass")

	data := []byte("hello tron")
	sig, err := w.SignDataWithPassphrase(acct, "pass", "text/plain", data)
	require.NoError(t, err)
	assert.Len(t, sig, 65)

	// Recover and verify
	hash := crypto.Keccak256(data)
	pubBytes, err := crypto.Ecrecover(hash, sig)
	require.NoError(t, err)
	pubKey, err := UnmarshalPublic(pubBytes)
	require.NoError(t, err)
	recoveredAddr := address.PubkeyToAddress(*pubKey)
	assert.Equal(t, acct.Address, recoveredAddr)
}

func TestKeystoreWallet_SignDataWithPassphrase_RejectsUnknownAccount(t *testing.T) {
	w, _, _ := newTestWallet(t, "pass")

	otherAddr := make([]byte, 21)
	otherAddr[0] = 0x41
	otherAddr[20] = 0x01
	_, err := w.SignDataWithPassphrase(Account{Address: otherAddr}, "pass", "text/plain", []byte("data"))
	assert.ErrorIs(t, err, ErrUnknownAccount)
}

func TestKeystoreWallet_SignText(t *testing.T) {
	w, ks, acct := newTestWallet(t, "pass")
	require.NoError(t, ks.Unlock(acct, "pass"))

	text := []byte("Sign this message")
	sig, err := w.SignText(acct, text)
	require.NoError(t, err)
	assert.Len(t, sig, 65)
}

func TestKeystoreWallet_SignText_RejectsUnknownAccount(t *testing.T) {
	w, ks, acct := newTestWallet(t, "pass")
	require.NoError(t, ks.Unlock(acct, "pass"))

	otherAddr := make([]byte, 21)
	otherAddr[0] = 0x41
	otherAddr[20] = 0x01
	_, err := w.SignText(Account{Address: otherAddr}, []byte("text"))
	assert.ErrorIs(t, err, ErrUnknownAccount)
}

func TestKeystoreWallet_SignTextWithPassphrase(t *testing.T) {
	w, _, acct := newTestWallet(t, "pass")

	text := []byte("Sign this message")
	sig, err := w.SignTextWithPassphrase(acct, "pass", text)
	require.NoError(t, err)
	assert.Len(t, sig, 65)
}

func TestKeystoreWallet_SignTextWithPassphrase_RejectsUnknownAccount(t *testing.T) {
	w, _, _ := newTestWallet(t, "pass")

	otherAddr := make([]byte, 21)
	otherAddr[0] = 0x41
	otherAddr[20] = 0x01
	_, err := w.SignTextWithPassphrase(Account{Address: otherAddr}, "pass", []byte("text"))
	assert.ErrorIs(t, err, ErrUnknownAccount)
}

func TestKeystoreWallet_SignTxWithPassphrase_RejectsUnknownAccount(t *testing.T) {
	w, _, _ := newTestWallet(t, "pass")

	tx := &core.Transaction{RawData: &core.TransactionRaw{RefBlockBytes: []byte{0x01}}}
	otherAddr := make([]byte, 21)
	otherAddr[0] = 0x41
	otherAddr[20] = 0x01
	_, err := w.SignTxWithPassphrase(Account{Address: otherAddr}, "pass", tx)
	assert.ErrorIs(t, err, ErrUnknownAccount)
}

func TestKeystoreWallet_URL(t *testing.T) {
	w, _, acct := newTestWallet(t, "pass")
	url := w.URL()
	assert.Equal(t, acct.URL.Scheme, url.Scheme)
	assert.Equal(t, acct.URL.Path, url.Path)
}

func TestKeystoreWallet_Accounts_ReturnsSingleAccount(t *testing.T) {
	w, _, acct := newTestWallet(t, "pass")
	accounts := w.Accounts()
	require.Len(t, accounts, 1)
	assert.Equal(t, acct.Address, accounts[0].Address)
}

func TestForPath_CreatesUsableKeystore(t *testing.T) {
	tmpDir := t.TempDir()
	ks := ForPath(tmpDir)
	require.NotNil(t, ks)

	acct, err := ks.NewAccount("test")
	require.NoError(t, err)
	assert.Len(t, acct.Address, 21)
	assert.Equal(t, byte(0x41), acct.Address[0])
}

func TestForPathLight_CreatesUsableKeystore(t *testing.T) {
	tmpDir := t.TempDir()
	ks := ForPathLight(tmpDir)
	require.NotNil(t, ks)

	acct, err := ks.NewAccount("test")
	require.NoError(t, err)
	assert.Len(t, acct.Address, 21)
	assert.Equal(t, byte(0x41), acct.Address[0])
}
