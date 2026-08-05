package keystore_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/require"
)

// keystoreWithKDFParams builds a v3 keystore whose kdfparams are exactly the
// given map. Everything else is well formed, so the only thing under test is how
// getKDFKey handles the parameters — which it consumes *before* the MAC is
// verified, making them unauthenticated attacker-controlled input.
func keystoreWithKDFParams(t *testing.T, kdf string, params map[string]interface{}) []byte {
	t.Helper()
	doc := map[string]interface{}{
		"address": "41a614f803b6fd780986a42c78ec9c7f77e6ded13c",
		"crypto": map[string]interface{}{
			"cipher":       "aes-128-ctr",
			"ciphertext":   "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"cipherparams": map[string]interface{}{"iv": "0123456789abcdef0123456789abcdef"},
			"kdf":          kdf,
			"kdfparams":    params,
			"mac":          "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		},
		"id":      "3198bc9c-6672-5ab3-d995-4942343ae5b6",
		"version": 3,
	}
	b, err := json.Marshal(doc)
	require.NoError(t, err)
	return b
}

func TestDecryptKey_MalformedKDFParams(t *testing.T) {
	validScrypt := func() map[string]interface{} {
		return map[string]interface{}{
			"dklen": 32, "n": 4096, "p": 1, "r": 8,
			"salt": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}
	}

	tests := map[string]struct {
		kdf    string
		params map[string]interface{}
	}{
		// Each of these previously hit an unchecked type assertion and panicked:
		// a missing key yields nil, and nil.(string) / nil.(float64) both panic.
		"missing salt": {"scrypt", map[string]interface{}{"dklen": 32, "n": 4096, "p": 1, "r": 8}},
		"salt not a string": {"scrypt", map[string]interface{}{
			"dklen": 32, "n": 4096, "p": 1, "r": 8, "salt": 12345,
		}},
		"salt not hex": {"scrypt", map[string]interface{}{
			"dklen": 32, "n": 4096, "p": 1, "r": 8, "salt": "nothex",
		}},
		"missing n":            {"scrypt", withoutKey(validScrypt(), "n")},
		"missing r":            {"scrypt", withoutKey(validScrypt(), "r")},
		"missing p":            {"scrypt", withoutKey(validScrypt(), "p")},
		"missing dklen":        {"scrypt", withoutKey(validScrypt(), "dklen")},
		"n not a number":       {"scrypt", withKey(validScrypt(), "n", "lots")},
		"n not a whole number": {"scrypt", withKey(validScrypt(), "n", 4096.5)},
		"pbkdf2 missing prf": {"pbkdf2", map[string]interface{}{
			"dklen": 32, "c": 262144,
			"salt": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}},
		"pbkdf2 prf not a string": {"pbkdf2", map[string]interface{}{
			"dklen": 32, "c": 262144, "prf": 7,
			"salt": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}},

		// Unbounded cost. These are rejected on the parameters alone, before any
		// derivation runs, so the test finishes immediately instead of trying to
		// allocate the requested working set.
		"n beyond limit":     {"scrypt", withKey(validScrypt(), "n", 1<<40)},
		"r beyond limit":     {"scrypt", withKey(validScrypt(), "r", 1<<20)},
		"p beyond limit":     {"scrypt", withKey(validScrypt(), "p", 1<<20)},
		"dklen beyond limit": {"scrypt", withKey(validScrypt(), "dklen", 1<<30)},
		// The decrypt paths read derivedKey[16:32]; anything below 32 is unusable
		// and only avoids a panic because the KDFs return spare capacity.
		"dklen below 32":      {"scrypt", withKey(validScrypt(), "dklen", 1)},
		"dklen just below 32": {"scrypt", withKey(validScrypt(), "dklen", 31)},
		"pbkdf2 dklen below 32": {"pbkdf2", map[string]interface{}{
			"dklen": 16, "c": 262144, "prf": "hmac-sha256",
			"salt": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}},
		"pbkdf2 c beyond limit": {"pbkdf2", map[string]interface{}{
			"dklen": 32, "c": 1 << 40, "prf": "hmac-sha256",
			"salt": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		}},
		// Within the individual limits, but 128*N*r would be ~34 GiB.
		"scrypt memory beyond limit": {"scrypt", withKey(withKey(validScrypt(), "n", 1<<22), "r", 64)},
		// Salt amplifies pre-MAC KDF cost and allocates on hex decode.
		"salt beyond limit": {"scrypt", withKey(validScrypt(), "salt", strings.Repeat("aa", 65))},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			doc := keystoreWithKDFParams(t, tt.kdf, tt.params)

			// The worker only collects a result. testify's require calls
			// t.FailNow, which must run on the test goroutine — calling it from
			// here would Goexit the wrong stack and could hang or misreport
			// instead of failing cleanly. So all assertions happen below.
			type outcome struct {
				err      error
				panicked any
			}
			results := make(chan outcome, 1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						results <- outcome{panicked: r}
					}
				}()
				_, err := keystore.DecryptKey(doc, "passphrase")
				results <- outcome{err: err}
			}()

			select {
			case got := <-results:
				require.Nil(t, got.panicked, "DecryptKey panicked instead of returning an error")
				require.Error(t, got.err)
			case <-time.After(20 * time.Second):
				t.Fatal("DecryptKey did not reject the parameters promptly — cost is still unbounded")
			}
		})
	}
}

func withoutKey(m map[string]interface{}, key string) map[string]interface{} {
	delete(m, key)
	return m
}

func withKey(m map[string]interface{}, key string, value interface{}) map[string]interface{} {
	m[key] = value
	return m
}
