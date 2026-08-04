package keystore_test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

const compatPassphrase = "keystore compatibility test passphrase"

// compatTestKey derives a throwaway signing key from a fixed label.
//
// It is derived rather than written as a hex literal on purpose: a 32-byte hex
// string in a wallet SDK reads like a real private key, and nothing that looks
// like one belongs in the repository. Deriving it keeps the expected address
// stable across runs without storing a key.
func compatTestKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	seed := sha256.Sum256([]byte("gotron-sdk keystore compatibility test — throwaway key, never funded"))
	priv, err := ethcrypto.ToECDSA(seed[:])
	require.NoError(t, err)
	return priv
}

// buildV3Keystore produces a Web3 Secret Storage v3 document for the given KDF
// parameters, following the same construction as EncryptDataV3 but without its
// hardcoded r=8 / dklen=32. That is the point: the SDK only ever *writes* one
// parameter set, so a round trip through EncryptKey/DecryptKey cannot show
// whether files written by other tools — or by older versions — still open.
func buildV3Keystore(t *testing.T, kdf string, params map[string]interface{}) []byte {
	t.Helper()

	priv := compatTestKey(t)
	keyBytes := ethcrypto.FromECDSA(priv)

	salt, err := hex.DecodeString(params["salt"].(string))
	require.NoError(t, err)
	dklen := params["dklen"].(int)

	var derivedKey []byte
	switch kdf {
	case "scrypt":
		derivedKey, err = scrypt.Key([]byte(compatPassphrase), salt,
			params["n"].(int), params["r"].(int), params["p"].(int), dklen)
		require.NoError(t, err)
	case "pbkdf2":
		derivedKey = pbkdf2.Key([]byte(compatPassphrase), salt,
			params["c"].(int), dklen, sha256.New)
	default:
		t.Fatalf("unsupported kdf %q", kdf)
	}

	iv := make([]byte, aes.BlockSize)
	block, err := aes.NewCipher(derivedKey[:16])
	require.NoError(t, err)
	cipherText := make([]byte, len(keyBytes))
	cipher.NewCTR(block, iv).XORKeyStream(cipherText, keyBytes)

	mac := ethcrypto.Keccak256(derivedKey[16:32], cipherText)

	doc := map[string]interface{}{
		"address": hex.EncodeToString(address.PubkeyToAddress(priv.PublicKey)),
		"crypto": map[string]interface{}{
			"cipher":       "aes-128-ctr",
			"ciphertext":   hex.EncodeToString(cipherText),
			"cipherparams": map[string]interface{}{"iv": hex.EncodeToString(iv)},
			"kdf":          kdf,
			"kdfparams":    params,
			"mac":          hex.EncodeToString(mac),
		},
		"id":      "3198bc9c-6672-5ab3-d995-4942343ae5b6",
		"version": 3,
	}
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	return b
}

func compatSalt() string {
	return "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"
}

// TestDecryptKey_ExistingKeystoreParams checks that keystores written with the
// parameter sets found in the wild still open. The KDF validation added for W10
// bounds n, r, p, c and dklen, and a bound that is too tight would lock users out
// of their own wallets — which the existing round-trip tests cannot detect,
// because they only ever encrypt with this SDK's own hardcoded parameters.
func TestDecryptKey_ExistingKeystoreParams(t *testing.T) {
	tests := map[string]struct {
		kdf    string
		params map[string]interface{}
	}{
		"geth light scrypt": {"scrypt", map[string]interface{}{
			"n": 4096, "r": 8, "p": 6, "dklen": 32, "salt": compatSalt(),
		}},
		"metamask / myetherwallet scrypt": {"scrypt", map[string]interface{}{
			"n": 8192, "r": 8, "p": 1, "dklen": 32, "salt": compatSalt(),
		}},
		"gotron-sdk light (LightScryptN/P)": {"scrypt", map[string]interface{}{
			"n": 1 << 12, "r": 8, "p": 1, "dklen": 32, "salt": compatSalt(),
		}},
		"low n": {"scrypt", map[string]interface{}{
			"n": 1024, "r": 8, "p": 1, "dklen": 32, "salt": compatSalt(),
		}},
		"r=1": {"scrypt", map[string]interface{}{
			"n": 4096, "r": 1, "p": 1, "dklen": 32, "salt": compatSalt(),
		}},
		"r=16": {"scrypt", map[string]interface{}{
			"n": 4096, "r": 16, "p": 1, "dklen": 32, "salt": compatSalt(),
		}},
		"p=8": {"scrypt", map[string]interface{}{
			"n": 4096, "r": 8, "p": 8, "dklen": 32, "salt": compatSalt(),
		}},
		"pbkdf2 standard": {"pbkdf2", map[string]interface{}{
			"c": 262144, "dklen": 32, "prf": "hmac-sha256", "salt": compatSalt(),
		}},
		"pbkdf2 low iterations": {"pbkdf2", map[string]interface{}{
			"c": 10240, "dklen": 32, "prf": "hmac-sha256", "salt": compatSalt(),
		}},
		"dklen above 32": {"scrypt", map[string]interface{}{
			"n": 4096, "r": 8, "p": 1, "dklen": 64, "salt": compatSalt(),
		}},
	}

	priv := compatTestKey(t)
	wantAddr := address.PubkeyToAddress(priv.PublicKey)
	wantKey := hex.EncodeToString(ethcrypto.FromECDSA(priv))

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			doc := buildV3Keystore(t, tt.kdf, tt.params)

			key, err := keystore.DecryptKey(doc, compatPassphrase)
			require.NoError(t, err, "a keystore with these parameters must still decrypt")
			require.Equal(t, wantAddr.String(), key.Address.String())
			require.Equal(t, wantKey, hex.EncodeToString(ethcrypto.FromECDSA(key.PrivateKey)))
		})
	}
}

// The geth default is 256 MiB of scrypt working set, so it is slow enough to skip
// under -short. It is the parameter set gotron-sdk itself writes via StandardScryptN.
func TestDecryptKey_StandardScryptParams(t *testing.T) {
	if testing.Short() {
		t.Skip("scrypt N=1<<18 needs 256 MiB and about a second")
	}
	doc := buildV3Keystore(t, "scrypt", map[string]interface{}{
		"n": keystore.StandardScryptN, "r": 8, "p": keystore.StandardScryptP,
		"dklen": 32, "salt": compatSalt(),
	})
	key, err := keystore.DecryptKey(doc, compatPassphrase)
	require.NoError(t, err)
	require.Equal(t,
		hex.EncodeToString(ethcrypto.FromECDSA(compatTestKey(t))),
		hex.EncodeToString(ethcrypto.FromECDSA(key.PrivateKey)))
}

// The wrong passphrase must still be reported as such, not as a parameter error.
func TestDecryptKey_WrongPassphraseStillReported(t *testing.T) {
	doc := buildV3Keystore(t, "scrypt", map[string]interface{}{
		"n": 4096, "r": 8, "p": 1, "dklen": 32, "salt": compatSalt(),
	})
	_, err := keystore.DecryptKey(doc, "not the passphrase")
	require.ErrorIs(t, err, keystore.ErrDecrypt)
}

// The one deliberate incompatibility: dklen below 32 is now rejected. The decrypt
// paths read derivedKey[:16] and derivedKey[16:32], so such a file was never
// usable — it only avoided a panic because the KDFs return spare capacity, and it
// is invalid under the v3 spec, which fixes dklen at 32.
func TestDecryptKey_ShortDKLenIsRejected(t *testing.T) {
	doc := buildV3Keystore(t, "scrypt", map[string]interface{}{
		"n": 4096, "r": 8, "p": 1, "dklen": 16, "salt": compatSalt(),
	})
	_, err := keystore.DecryptKey(doc, compatPassphrase)
	require.Error(t, err)
	require.Contains(t, err.Error(), "dklen")
}
