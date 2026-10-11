package pqc

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/scrypt"
)

const ksTestPassword = "pqc keystore test password"

// mlSeedKey returns the ML-DSA-44 key of the nile-testnet fixture seed, with
// its seed retained.
func mlSeedKey(t *testing.T) PrivateKey {
	t.Helper()
	k, err := NewMLDSA44FromSeed(mustHex(t, mlFixtureSeedHex))
	require.NoError(t, err)
	return k
}

// mlKeyFromByte returns the ML-DSA-44 key whose seed is 32 copies of b.
func mlKeyFromByte(t *testing.T, b byte) PrivateKey {
	t.Helper()
	k, err := NewMLDSA44FromSeed(bytes.Repeat([]byte{b}, MLDSA44SeedSize))
	require.NoError(t, err)
	return k
}

// mlExpandedKey returns the fixture key imported from its expanded form, so
// that it has no seed.
func mlExpandedKey(t *testing.T) PrivateKey {
	t.Helper()
	k, err := ParsePrivateKey(core.PQScheme_ML_DSA_44, mlSeedKey(t).Bytes(), nil)
	require.NoError(t, err)
	require.Nil(t, k.Seed())
	return k
}

// fnKey returns the tron-grpc FN-DSA-512 test key pair.
func fnKey(t *testing.T) PrivateKey {
	t.Helper()
	k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, mustHex(t, fnPairPrivKeyHex), mustHex(t, fnPairPubKeyHex))
	require.NoError(t, err)
	return k
}

// rawKeystore builds a keystore through the byte-level API, bypassing the
// consistency checks EncryptKeystore applies.
func rawKeystore(t *testing.T, scheme, addr string, ext, seed []byte) []byte {
	t.Helper()
	out, err := keystore.EncryptPQKeyV3(scheme, addr, ext, seed, []byte(ksTestPassword),
		keystore.LightScryptN, keystore.LightScryptP)
	require.NoError(t, err)
	return out
}

func assertSameKey(t *testing.T, want, got PrivateKey) {
	t.Helper()
	require.NotNil(t, got)
	assert.Equal(t, want.Scheme(), got.Scheme())
	assert.True(t, want.Public().Equal(got.Public()))
	assert.Equal(t, want.Bytes(), got.Bytes())
	assert.Equal(t, want.Seed(), got.Seed())
}

func TestKeystore_RoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		key        func(*testing.T) PrivateKey
		scheme     string
		extHexLen  int
		seedHexLen int // 0: no seed segment
	}{
		{"ML-DSA-44 with seed", mlSeedKey, "ML_DSA_44", 2 * MLDSA44PrivateKeySize, 2 * MLDSA44SeedSize},
		{"ML-DSA-44 expanded only", mlExpandedKey, "ML_DSA_44", 2 * MLDSA44PrivateKeySize, 0},
		{"FN-DSA-512", fnKey, "FN_DSA_512", 2 * FNDSA512ExtendedPrivateKeySize, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := tt.key(t)
			out, err := EncryptKeystore(k, ksTestPassword, keystore.LightScryptN, keystore.LightScryptP)
			require.NoError(t, err)

			var m struct {
				Address string         `json:"address"`
				Scheme  string         `json:"scheme"`
				Version int            `json:"version"`
				Crypto  map[string]any `json:"crypto"`
			}
			require.NoError(t, json.Unmarshal(out, &m))
			assert.Equal(t, k.Public().Address().String(), m.Address)
			assert.Equal(t, tt.scheme, m.Scheme)
			assert.Equal(t, 3, m.Version)
			assert.Len(t, m.Crypto["ciphertext"], tt.extHexLen)
			if tt.seedHexLen == 0 {
				assert.NotContains(t, m.Crypto, "seedciphertext")
				assert.NotContains(t, m.Crypto, "seedcipherparams")
				assert.NotContains(t, m.Crypto, "seedmac")
			} else {
				assert.Len(t, m.Crypto["seedciphertext"], tt.seedHexLen)
			}
			assert.NotContains(t, string(out), hex.EncodeToString(k.Bytes()[:16]), "plaintext key leaked")

			got, err := DecryptKeystore(out, ksTestPassword)
			require.NoError(t, err)
			assertSameKey(t, k, got)
		})
	}
}

func TestKeystore_WrongPassword(t *testing.T) {
	out, err := EncryptKeystore(mlSeedKey(t), ksTestPassword, keystore.LightScryptN, keystore.LightScryptP)
	require.NoError(t, err)
	k, err := DecryptKeystore(out, "not the password")
	require.Error(t, err)
	assert.Nil(t, k)
	assert.True(t, errors.Is(err, keystore.ErrDecrypt), "got %v", err)
}

func TestEncryptKeystore_Rejects(t *testing.T) {
	_, err := EncryptKeystore(nil, ksTestPassword, keystore.LightScryptN, keystore.LightScryptP)
	assert.ErrorIs(t, err, ErrInvalidKey)
	_, err = EncryptKeystore(mlSeedKey(t), ksTestPassword, 3000, 1)
	assert.Error(t, err)
}

func TestDecryptKeystore_Rejects(t *testing.T) {
	ml := mlSeedKey(t)
	other := mlKeyFromByte(t, 0x42)
	fn := fnKey(t)
	mlAddr := ml.Public().Address().String()
	fnAddr := fn.Public().Address().String()
	fnExt := append(fn.Bytes(), fn.Public().Bytes()...)
	fnSeed := bytes.Repeat([]byte{7}, fnPersistedSeedSize)

	tests := []struct {
		name    string
		data    func(t *testing.T) []byte
		wantErr error
		msg     string // substring of the error, when set
	}{
		{name: "ML seed derives a different key than ext", data: func(t *testing.T) []byte {
			return rawKeystore(t, "ML_DSA_44", mlAddr, ml.Bytes(), other.Seed())
		}, wantErr: ErrKeyMismatch, msg: "seed and private key"},
		{"address of another key", func(t *testing.T) []byte {
			return rawKeystore(t, "ML_DSA_44", other.Public().Address().String(), ml.Bytes(), ml.Seed())
		}, ErrKeyMismatch, ""},
		{"address not base58check", func(t *testing.T) []byte {
			return rawKeystore(t, "ML_DSA_44", "not-an-address", ml.Bytes(), nil)
		}, ErrKeyMismatch, ""},
		{"address missing", func(t *testing.T) []byte {
			out := rawKeystore(t, "ML_DSA_44", mlAddr, ml.Bytes(), nil)
			m := map[string]any{}
			require.NoError(t, json.Unmarshal(out, &m))
			delete(m, "address")
			b, err := json.Marshal(m)
			require.NoError(t, err)
			return b
		}, ErrKeyMismatch, ""},
		{"FN seed only", func(t *testing.T) []byte {
			return rawKeystore(t, "FN_DSA_512", fnAddr, nil, fnSeed)
		}, ErrInvalidKey, ""},
		{"FN seed of wrong length", func(t *testing.T) []byte {
			return rawKeystore(t, "FN_DSA_512", fnAddr, fnExt, fnSeed[:32])
		}, ErrInvalidKey, ""},
		{"ML ext holding a seed", func(t *testing.T) []byte {
			return rawKeystore(t, "ML_DSA_44", mlAddr, ml.Seed(), nil)
		}, ErrInvalidKey, ""},
		{"ML seed of wrong length", func(t *testing.T) []byte {
			return rawKeystore(t, "ML_DSA_44", mlAddr, nil, bytes.Repeat([]byte{1}, 48))
		}, ErrInvalidKey, ""},
		{"FN ext of canonical 1280-byte form", func(t *testing.T) []byte {
			return rawKeystore(t, "FN_DSA_512", fnAddr, fn.Bytes(), nil)
		}, ErrInvalidKey, ""},
		{"unknown scheme", func(t *testing.T) []byte {
			return rawKeystore(t, "SLH_DSA_128S", mlAddr, ml.Bytes(), nil)
		}, ErrUnsupportedScheme, ""},
		{"UNKNOWN_PQ_SCHEME", func(t *testing.T) []byte {
			return rawKeystore(t, "UNKNOWN_PQ_SCHEME", mlAddr, ml.Bytes(), nil)
		}, ErrUnsupportedScheme, ""},
		{"scheme of the other algorithm", func(t *testing.T) []byte {
			return rawKeystore(t, "FN_DSA_512", mlAddr, ml.Bytes(), nil)
		}, ErrInvalidKey, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := DecryptKeystore(tt.data(t), ksTestPassword)
			require.Error(t, err)
			assert.Nil(t, k)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.False(t, errors.Is(err, keystore.ErrDecrypt), "reported as wrong password: %v", err)
			if tt.msg != "" {
				assert.Contains(t, err.Error(), tt.msg)
			}
		})
	}
}

// wallet-cli persists the FN-DSA-512 keygen seed next to the extended key
// (WalletApiWrapper.generate). A Falcon seed cannot be expanded portably, so
// its MAC is verified but the key comes from the extended segment alone.
func TestDecryptKeystore_FNWithSeedSegment(t *testing.T) {
	fn := fnKey(t)
	ext := append(fn.Bytes(), fn.Public().Bytes()...)
	data := rawKeystore(t, "FN_DSA_512", fn.Public().Address().String(), ext,
		bytes.Repeat([]byte{7}, fnPersistedSeedSize))
	got, err := DecryptKeystore(data, ksTestPassword)
	require.NoError(t, err)
	assertSameKey(t, fn, got)

	_, err = DecryptKeystore(data, "wrong")
	assert.ErrorIs(t, err, keystore.ErrDecrypt)
}

// TestDecryptKeystore_IndependentKAT builds wallet-cli PQ keystores by hand,
// mirroring Wallet.createPQ / createPQWalletFile in wallet-cli
// feature/post-quantum, with fixed salt and IVs and the scrypt, AES-128-CTR
// and Keccak-256 primitives called directly. It guards against the encoder
// and decoder sharing a bug that a round trip would not reveal. No keystore
// produced by a real wallet-cli build is available yet; replace or extend
// this with one when it is.
func TestDecryptKeystore_IndependentKAT(t *testing.T) {
	const password = "wallet-cli KAT"
	salt := bytes.Repeat([]byte{0x11}, 32)
	extIV := bytes.Repeat([]byte{0x22}, 16)
	seedIV := bytes.Repeat([]byte{0x33}, 16)
	dk, err := scrypt.Key([]byte(password), salt, keystore.LightScryptN, 8, keystore.LightScryptP, 32)
	require.NoError(t, err)

	ctr := func(iv, plain []byte) []byte {
		block, err := aes.NewCipher(dk[:16])
		require.NoError(t, err)
		out := make([]byte, len(plain))
		cipher.NewCTR(block, iv).XORKeyStream(out, plain)
		return out
	}
	mac := func(ct []byte) []byte {
		return common.Keccak256(append(bytes.Clone(dk[16:32]), ct...))
	}
	extFields := func(ext []byte) string {
		ct := ctr(extIV, ext)
		return fmt.Sprintf(`"ciphertext":"%x","cipherparams":{"iv":"%x"},"mac":"%x"`, ct, extIV, mac(ct))
	}
	seedFields := func(seed []byte) string {
		ct := ctr(seedIV, seed)
		return fmt.Sprintf(`"seedciphertext":"%x","seedcipherparams":{"iv":"%x"},"seedmac":"%x"`, ct, seedIV, mac(ct))
	}
	build := func(addr, scheme, segments string) []byte {
		return []byte(fmt.Sprintf(`{"address":"%s","id":"0b9a3c7e-5f43-4c55-9a8e-7f0f6d0c1e2a","version":3,"scheme":"%s",`+
			`"crypto":{"cipher":"aes-128-ctr",%s,"kdf":"scrypt",`+
			`"kdfparams":{"dklen":32,"n":%d,"p":%d,"r":8,"salt":"%x"}}}`,
			addr, scheme, segments, keystore.LightScryptN, keystore.LightScryptP, salt))
	}

	mlSeed := mustHex(t, mlFixtureSeedHex)
	ml := mlSeedKey(t)
	require.Equal(t, mlFixturePubKeyHex, hex.EncodeToString(ml.Public().Bytes()))
	fnExt := append(mustHex(t, fnPairPrivKeyHex), mustHex(t, fnPairPubKeyHex)...)

	tests := []struct {
		name       string
		data       []byte
		wantScheme core.PQScheme
		wantAddr   string
		wantPub    string
		wantSeed   []byte
	}{
		{"ML-DSA-44 ext and seed", build(mlFixtureAddress, "ML_DSA_44", extFields(ml.Bytes())+","+seedFields(mlSeed)),
			core.PQScheme_ML_DSA_44, mlFixtureAddress, mlFixturePubKeyHex, mlSeed},
		{"ML-DSA-44 ext only", build(mlFixtureAddress, "ML_DSA_44", extFields(ml.Bytes())),
			core.PQScheme_ML_DSA_44, mlFixtureAddress, mlFixturePubKeyHex, nil},
		// Jackson writes the absent ext fields as explicit nulls.
		{"ML-DSA-44 seed only", build(mlFixtureAddress, "ML_DSA_44", `"ciphertext":null,"cipherparams":null,"mac":null,`+seedFields(mlSeed)),
			core.PQScheme_ML_DSA_44, mlFixtureAddress, mlFixturePubKeyHex, mlSeed},
		{"FN-DSA-512 ext only", build(fnPairAddress, "FN_DSA_512", extFields(fnExt)),
			core.PQScheme_FN_DSA_512, fnPairAddress, fnPairPubKeyHex, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, err := DecryptKeystore(tt.data, password)
			require.NoError(t, err)
			assert.Equal(t, tt.wantScheme, k.Scheme())
			assert.Equal(t, tt.wantAddr, k.Public().Address().String())
			assert.Equal(t, tt.wantPub, hex.EncodeToString(k.Public().Bytes()))
			assert.Equal(t, tt.wantSeed, k.Seed())

			_, err = DecryptKeystore(tt.data, password+"!")
			assert.ErrorIs(t, err, keystore.ErrDecrypt)
		})
	}
}
