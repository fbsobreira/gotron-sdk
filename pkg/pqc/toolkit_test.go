package pqc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// toolkitJSON renders a Toolkit key file from its fields; empty fields are
// left out.
func toolkitJSON(t *testing.T, fields map[string]string) []byte {
	t.Helper()
	b, err := json.Marshal(fields)
	require.NoError(t, err)
	return b
}

// mlFields returns the Toolkit fields of the ML-DSA-44 fixture key.
func mlFields(t *testing.T) map[string]string {
	t.Helper()
	k := mlSeedKey(t)
	return map[string]string{
		"scheme":     "ML_DSA_44",
		"seed":       mlFixtureSeedHex,
		"privateKey": hex.EncodeToString(k.Bytes()),
		"publicKey":  mlFixturePubKeyHex,
		"address":    mlFixtureAddress,
	}
}

// fnFields returns the Toolkit fields of the tron-grpc FN-DSA-512 key pair,
// with a 48-byte seed as Toolkit.jar writes it.
func fnFields() map[string]string {
	return map[string]string{
		"scheme":     "FN_DSA_512",
		"seed":       strings.Repeat("07", fnPersistedSeedSize),
		"privateKey": fnPairPrivKeyHex,
		"publicKey":  fnPairPubKeyHex,
		"address":    fnPairAddress,
	}
}

func with(base map[string]string, kv ...string) map[string]string {
	out := make(map[string]string, len(base))
	for k, v := range base {
		out[k] = v
	}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] == "" {
			delete(out, kv[i])
			continue
		}
		out[kv[i]] = kv[i+1]
	}
	return out
}

func TestMarshalToolkitJSON_Golden(t *testing.T) {
	seed := make([]byte, MLDSA44SeedSize)
	k, err := NewMLDSA44FromSeed(seed)
	require.NoError(t, err)

	// Pin the key bytes independently of this package's encoders, using the
	// Nile all-zero-seed KAT digests.
	sk, pk := k.Bytes(), k.Public().Bytes()
	skSum, pkSum := sha256.Sum256(sk), sha256.Sum256(pk)
	require.Equal(t, "0f9086044d77b6d610c7e92418d9f70a398c69febc7e99f8254aaea98dcfbe77", hex.EncodeToString(skSum[:]))
	require.Equal(t, "eb4e7302842153b0fa19e8620739ad258af4929c26dd89079a7ec7d4282208e1", hex.EncodeToString(pkSum[:]))
	addr := common.EncodeCheck(append([]byte{0x41}, common.Keccak256(pk)[12:]...))

	// Layout of PqKeyNew.buildJson: this field order, two-space indent,
	// lowercase hex without 0x, trailing newline.
	want := fmt.Sprintf("{\n  \"scheme\": \"ML_DSA_44\",\n  \"seed\": \"%s\",\n  \"privateKey\": \"%x\",\n"+
		"  \"publicKey\": \"%x\",\n  \"address\": \"%s\"\n}\n",
		strings.Repeat("0", 64), sk, pk, addr)
	got, err := MarshalToolkitJSON(k)
	require.NoError(t, err)
	assert.Equal(t, want, string(got))
}

func TestMarshalToolkitJSON_NoSeed(t *testing.T) {
	ml := mlExpandedKey(t)
	tests := []struct {
		name string
		key  PrivateKey
		want string
	}{
		{"FN-DSA-512", fnKey(t), "{\n  \"scheme\": \"FN_DSA_512\",\n  \"privateKey\": \"" + fnPairPrivKeyHex +
			"\",\n  \"publicKey\": \"" + fnPairPubKeyHex + "\",\n  \"address\": \"" + fnPairAddress + "\"\n}\n"},
		{"ML-DSA-44 expanded", ml, "{\n  \"scheme\": \"ML_DSA_44\",\n  \"privateKey\": \"" + hex.EncodeToString(ml.Bytes()) +
			"\",\n  \"publicKey\": \"" + mlFixturePubKeyHex + "\",\n  \"address\": \"" + mlFixtureAddress + "\"\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MarshalToolkitJSON(tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}

	_, err := MarshalToolkitJSON(nil)
	assert.ErrorIs(t, err, ErrInvalidKey)
}

func TestToolkitJSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		key  func(*testing.T) PrivateKey
	}{
		{"ML-DSA-44 with seed", mlSeedKey},
		{"ML-DSA-44 expanded only", mlExpandedKey},
		{"FN-DSA-512", fnKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := tt.key(t)
			b, err := MarshalToolkitJSON(k)
			require.NoError(t, err)
			got, err := ParseToolkitJSON(b)
			require.NoError(t, err)
			assertSameKey(t, k, got)
		})
	}
}

func TestParseToolkitJSON_Accepts(t *testing.T) {
	ml := mlSeedKey(t)
	mlSKHex := hex.EncodeToString(ml.Bytes())

	tests := []struct {
		name     string
		data     func(t *testing.T) []byte
		want     func(t *testing.T) PrivateKey
		wantSeed bool
	}{
		{"ML all fields", func(t *testing.T) []byte { return toolkitJSON(t, mlFields(t)) }, mlSeedKey, true},
		{"ML 0x prefixes and uppercase", func(t *testing.T) []byte {
			return toolkitJSON(t, with(mlFields(t),
				"seed", "0x"+strings.ToUpper(mlFixtureSeedHex),
				"privateKey", "0X"+strings.ToUpper(mlSKHex),
				"publicKey", "0x"+strings.ToUpper(mlFixturePubKeyHex)))
		}, mlSeedKey, true},
		{"ML seed only", func(t *testing.T) []byte {
			return toolkitJSON(t, map[string]string{"scheme": "ML_DSA_44", "seed": mlFixtureSeedHex})
		}, mlSeedKey, true},
		{"ML privateKey only", func(t *testing.T) []byte {
			return toolkitJSON(t, map[string]string{"scheme": "ML_DSA_44", "privateKey": mlSKHex})
		}, mlExpandedKey, false},
		{"ML blank seed treated as absent", func(t *testing.T) []byte {
			return toolkitJSON(t, with(mlFields(t), "seed", "   "))
		}, mlExpandedKey, false},
		{"unknown fields ignored", func(t *testing.T) []byte {
			return []byte(`{"scheme":"ML_DSA_44","seed":"` + mlFixtureSeedHex + `","comment":"x","n":1,"nested":{"a":[1]}}`)
		}, mlSeedKey, true},
		{"FN all fields, seed ignored", func(t *testing.T) []byte { return toolkitJSON(t, fnFields()) }, fnKey, false},
		{"FN malformed seed ignored", func(t *testing.T) []byte {
			return toolkitJSON(t, with(fnFields(), "seed", "not hex"))
		}, fnKey, false},
		{"FN 0x prefixes and uppercase, no address", func(t *testing.T) []byte {
			return toolkitJSON(t, with(fnFields(),
				"privateKey", "0x"+strings.ToUpper(fnPairPrivKeyHex),
				"publicKey", "0X"+strings.ToUpper(fnPairPubKeyHex),
				"address", ""))
		}, fnKey, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToolkitJSON(tt.data(t))
			require.NoError(t, err)
			want := tt.want(t)
			assertSameKey(t, want, got)
			assert.Equal(t, tt.wantSeed, got.Seed() != nil)
		})
	}
}

func TestParseToolkitJSON_Rejects(t *testing.T) {
	ml := mlSeedKey(t)
	other := mlKeyFromByte(t, 0x42)
	otherSKHex := hex.EncodeToString(other.Bytes())
	secrets := []string{mlFixtureSeedHex[2:18], hex.EncodeToString(ml.Bytes())[2:18], fnPairPrivKeyHex[2:18],
		otherSKHex[2:18], hex.EncodeToString(other.Seed())[2:18]}

	tests := []struct {
		name    string
		data    func(t *testing.T) []byte
		wantErr error
	}{
		{"not JSON", func(*testing.T) []byte { return []byte("{" + mlFixtureSeedHex) }, nil},
		{"unknown scheme", func(t *testing.T) []byte { return toolkitJSON(t, with(mlFields(t), "scheme", "SLH_DSA")) }, ErrUnsupportedScheme},
		{"missing scheme", func(t *testing.T) []byte { return toolkitJSON(t, with(mlFields(t), "scheme", "")) }, ErrUnsupportedScheme},
		{"lowercase scheme", func(t *testing.T) []byte { return toolkitJSON(t, with(mlFields(t), "scheme", "ml_dsa_44")) }, ErrUnsupportedScheme},
		{"missing seed and privateKey", func(t *testing.T) []byte {
			return toolkitJSON(t, with(mlFields(t), "seed", "", "privateKey", ""))
		}, ErrInvalidKey},
		{"ML seed and privateKey disagree", func(t *testing.T) []byte {
			return toolkitJSON(t, with(mlFields(t), "privateKey", otherSKHex, "publicKey", "", "address", ""))
		}, ErrKeyMismatch},
		{"ML publicKey mismatch", func(t *testing.T) []byte {
			return toolkitJSON(t, with(mlFields(t), "publicKey", hex.EncodeToString(other.Public().Bytes())))
		}, ErrKeyMismatch},
		{"ML seed-only publicKey mismatch", func(t *testing.T) []byte {
			return toolkitJSON(t, with(mlFields(t), "privateKey", "", "publicKey", hex.EncodeToString(other.Public().Bytes())))
		}, ErrKeyMismatch},
		{"ML address mismatch", func(t *testing.T) []byte {
			return toolkitJSON(t, with(mlFields(t), "address", other.Public().Address().String()))
		}, ErrKeyMismatch},
		{"ML address not base58check", func(t *testing.T) []byte {
			return toolkitJSON(t, with(mlFields(t), "address", "Tnot-an-address"))
		}, ErrKeyMismatch},
		{"ML privateKey holding a seed", func(t *testing.T) []byte {
			return toolkitJSON(t, map[string]string{"scheme": "ML_DSA_44", "privateKey": mlFixtureSeedHex})
		}, ErrInvalidKey},
		{"ML seed wrong length", func(t *testing.T) []byte {
			return toolkitJSON(t, map[string]string{"scheme": "ML_DSA_44", "seed": mlFixtureSeedHex + "00"})
		}, ErrInvalidKey},
		{"ML seed invalid hex", func(t *testing.T) []byte {
			return toolkitJSON(t, map[string]string{"scheme": "ML_DSA_44", "seed": "zz" + mlFixtureSeedHex[2:]})
		}, ErrInvalidKey},
		{"ML privateKey invalid hex", func(t *testing.T) []byte {
			return toolkitJSON(t, with(mlFields(t), "privateKey", "g"+hex.EncodeToString(ml.Bytes())[1:]))
		}, ErrInvalidKey},
		{"FN seed only", func(t *testing.T) []byte {
			return toolkitJSON(t, with(fnFields(), "privateKey", "", "publicKey", ""))
		}, ErrInvalidKey},
		{"FN seed and privateKey without publicKey", func(t *testing.T) []byte {
			return toolkitJSON(t, with(fnFields(), "publicKey", ""))
		}, ErrInvalidKey},
		{"FN extended privateKey form", func(t *testing.T) []byte {
			return toolkitJSON(t, with(fnFields(), "privateKey", fnPairPrivKeyHex+fnPairPubKeyHex))
		}, ErrInvalidKey},
		{"FN publicKey with 0x09 header", func(t *testing.T) []byte {
			return toolkitJSON(t, with(fnFields(), "publicKey", "09"+fnPairPubKeyHex))
		}, ErrInvalidKey},
		{"FN address mismatch", func(t *testing.T) []byte {
			return toolkitJSON(t, with(fnFields(), "address", mlFixtureAddress))
		}, ErrKeyMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := ParseToolkitJSON(tt.data(t))
			require.Error(t, err)
			assert.Nil(t, k)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			}
			msg := strings.ToLower(err.Error())
			for _, s := range secrets {
				assert.NotContains(t, msg, s, "error message leaks key material")
			}
		})
	}
}

// nile/plugins/README.md documents the Toolkit file layout with placeholders
// only ("<64 hex chars>"), so there is no complete published sample to parse.
// This checks that a key file in that exact layout, as PqKeyNew.buildJson
// writes it for FN-DSA-512 (seed included), loads.
func TestParseToolkitJSON_ToolkitLayout(t *testing.T) {
	data := "{\n  \"scheme\": \"FN_DSA_512\",\n  \"seed\": \"" + strings.Repeat("ab", fnPersistedSeedSize) +
		"\",\n  \"privateKey\": \"" + fnPairPrivKeyHex + "\",\n  \"publicKey\": \"" + fnPairPubKeyHex +
		"\",\n  \"address\": \"" + fnPairAddress + "\"\n}\n"
	k, err := ParseToolkitJSON([]byte(data))
	require.NoError(t, err)
	assert.Equal(t, core.PQScheme_FN_DSA_512, k.Scheme())
	assert.Equal(t, fnPairAddress, k.Public().Address().String())
	assert.True(t, bytes.Equal(mustHex(t, fnPairPrivKeyHex), k.Bytes()))
}

// The node's Jackson loader matches field names case-sensitively; so must we.
func TestParseToolkitJSON_FieldNamesCaseSensitive(t *testing.T) {
	ml := mlFields(t)

	t.Run("case variant alone is ignored", func(t *testing.T) {
		data := toolkitJSON(t, map[string]string{"scheme": "ML_DSA_44", "SEED": ml["seed"]})
		_, err := ParseToolkitJSON(data)
		assert.ErrorIs(t, err, ErrInvalidKey)
	})

	t.Run("case variant does not override the exact field", func(t *testing.T) {
		other := strings.Repeat("ff", MLDSA44SeedSize)
		data := toolkitJSON(t, with(ml, "privateKey", "", "publicKey", "", "address", "", "Seed", other))
		k, err := ParseToolkitJSON(data)
		require.NoError(t, err)
		assert.Equal(t, ml["seed"], hex.EncodeToString(k.Seed()))
	})
}
