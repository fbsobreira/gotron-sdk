package keystore

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/require"
)

// These tests look at the key memory itself. The black-box tests in
// keystore_security_test.go only show that signing stops working, which would
// still pass if the key were dropped from the map without being zeroed.

// wiped reports whether every word of k.D is zero. zeroKey clears the words in
// place without normalising the big.Int, so D.Sign() and D.Cmp would still
// report a non-zero value: the words are what matters.
func wiped(k *ecdsa.PrivateKey) bool {
	for _, w := range k.D.Bits() {
		if w != 0 {
			return false
		}
	}
	return true
}

// unlockedKey returns the key held for acc, read under ks.mu because the
// expire goroutine zeroes it under the same lock.
func unlockedKey(t *testing.T, ks *KeyStore, acc Account) *ecdsa.PrivateKey {
	t.Helper()
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	u, ok := ks.unlocked[acc.Address.String()]
	require.True(t, ok, "account must be unlocked")
	require.False(t, wiped(u.PrivateKey), "a freshly unlocked key must not be zero")
	return u.PrivateKey
}

func wipedLocked(ks *KeyStore, k *ecdsa.PrivateKey) bool {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return wiped(k)
}

func newZeroizeTestKeyStore(t *testing.T) (*KeyStore, Account) {
	t.Helper()
	ks := NewKeyStore(t.TempDir(), LightScryptN, LightScryptP)
	t.Cleanup(ks.Close)
	acc, err := ks.NewAccount("pass")
	require.NoError(t, err)
	return ks, acc
}

func TestZeroKey(t *testing.T) {
	k, err := crypto.GenerateKey()
	require.NoError(t, err)
	require.False(t, wiped(k))

	zeroKey(k)
	require.True(t, wiped(k))

	require.NotPanics(t, func() {
		zeroKey(nil)
		zeroKey(&ecdsa.PrivateKey{})
	})
}

func TestZeroKey_LargeD(t *testing.T) {
	// A D with every word set, so a loop that missed a word would show.
	d := new(big.Int).Sub(crypto.S256().Params().N, big.NewInt(1))
	k := &ecdsa.PrivateKey{D: d}
	zeroKey(k)
	require.True(t, wiped(k))
}

func TestLock_WipesUnlockedKey(t *testing.T) {
	ks, acc := newZeroizeTestKeyStore(t)
	require.NoError(t, ks.Unlock(acc, "pass"))
	k := unlockedKey(t, ks, acc)

	require.NoError(t, ks.Lock(acc.Address))
	require.True(t, wipedLocked(ks, k))
}

func TestTimedUnlock_ExpiryWipesKey(t *testing.T) {
	ks, acc := newZeroizeTestKeyStore(t)
	require.NoError(t, ks.TimedUnlock(acc, "pass", 50*time.Millisecond))
	k := unlockedKey(t, ks, acc)

	require.Eventually(t, func() bool { return wipedLocked(ks, k) },
		2*time.Second, 10*time.Millisecond)
}

func TestTimedUnlock_ReplacementWipesOldKey(t *testing.T) {
	ks, acc := newZeroizeTestKeyStore(t)
	require.NoError(t, ks.TimedUnlock(acc, "pass", time.Hour))
	old := unlockedKey(t, ks, acc)

	require.NoError(t, ks.TimedUnlock(acc, "pass", time.Hour))
	require.True(t, wipedLocked(ks, old), "the replaced key must be zeroed")
	unlockedKey(t, ks, acc) // the replacement is live and non-zero
}

func TestLock_AfterTimedUnlockWipesKey(t *testing.T) {
	ks, acc := newZeroizeTestKeyStore(t)
	require.NoError(t, ks.TimedUnlock(acc, "pass", time.Hour))
	k := unlockedKey(t, ks, acc)

	require.NoError(t, ks.Lock(acc.Address))
	require.True(t, wipedLocked(ks, k))
}

func TestClose_WipesUnlockedKeys(t *testing.T) {
	ks, acc := newZeroizeTestKeyStore(t)
	require.NoError(t, ks.Unlock(acc, "pass"))
	k := unlockedKey(t, ks, acc)

	ks.Close()
	require.True(t, wipedLocked(ks, k))
}

func TestGetKey_AddressMismatchFails(t *testing.T) {
	// GetKey zeroes the decrypted key before it reports a mismatch. The key
	// never leaves GetKey, so this only pins the error path it runs on.
	ks, acc := newZeroizeTestKeyStore(t)
	other, err := crypto.GenerateKey()
	require.NoError(t, err)
	otherKey := newKeyFromECDSA(other)

	_, err = ks.storage.GetKey(otherKey.Address, acc.URL.Path, "pass")
	require.ErrorContains(t, err, "key content mismatch")
}

func TestDecryptKey_RejectsKeyOutsideCurveOrder(t *testing.T) {
	n := crypto.S256().Params().N
	for name, d := range map[string][]byte{
		"zero":        make([]byte, 32),
		"curve order": n.FillBytes(make([]byte, 32)),
	} {
		t.Run(name, func(t *testing.T) {
			cj, err := EncryptDataV3(d, []byte("pass"), LightScryptN, LightScryptP)
			require.NoError(t, err)
			doc, err := json.Marshal(encryptedKeyJSONV3{
				Address: "41a614f803b6fd780986a42c78ec9c7f77e6ded13c",
				Crypto:  cj,
				ID:      "3198bc9c-6672-5ab3-d995-4942343ae5b6",
				Version: version,
			})
			require.NoError(t, err)

			require.NotPanics(t, func() {
				key, err := DecryptKey(doc, "pass")
				require.ErrorContains(t, err, "not a valid secp256k1 private key")
				require.Nil(t, key)
			})
		})
	}
}

func TestEncryptKey_LeavesCallerKeyIntact(t *testing.T) {
	// EncryptKey zeroes its own padded copy of D, never the caller's key.
	k, err := crypto.GenerateKey()
	require.NoError(t, err)
	want := new(big.Int).Set(k.D)

	_, err = EncryptKey(newKeyFromECDSA(k), "pass", LightScryptN, LightScryptP)
	require.NoError(t, err)
	require.Zero(t, want.Cmp(k.D))
}

func TestImportECDSA_LeavesCallerKeyIntact(t *testing.T) {
	// newKeyFromECDSA keeps the caller's key, so zeroing after the store would
	// wipe a key the caller still owns.
	ks := NewKeyStore(t.TempDir(), LightScryptN, LightScryptP)
	t.Cleanup(ks.Close)
	k, err := crypto.GenerateKey()
	require.NoError(t, err)
	want := new(big.Int).Set(k.D)

	_, err = ks.ImportECDSA(k, "pass")
	require.NoError(t, err)
	require.False(t, wiped(k))
	require.Zero(t, want.Cmp(k.D))
}

func TestNewAccount_KeyStillDecrypts(t *testing.T) {
	// NewAccount zeroes its in-memory key after writing the file; the file
	// must still hold the real key.
	ks, acc := newZeroizeTestKeyStore(t)
	_, key, err := ks.GetDecryptedKey(acc, "pass")
	require.NoError(t, err)
	defer zeroKey(key.PrivateKey)
	require.Equal(t, acc.Address, key.Address)

	hash := make([]byte, 32)
	_, err = rand.Read(hash)
	require.NoError(t, err)
	sig, err := crypto.Sign(hash, key.PrivateKey)
	require.NoError(t, err)
	pub, err := crypto.SigToPub(hash, sig)
	require.NoError(t, err)
	require.Equal(t, acc.Address, address.PubkeyToAddress(*pub))
}
