package keystore

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pqTestScheme   = "ML_DSA_44"
	pqTestAddress  = "TTestAddressNotCheckedByThisLayer"
	pqTestPassword = "pq-test-password"
)

var (
	pqTestExt  = bytes.Repeat([]byte{0xa5}, 2560)
	pqTestSeed = bytes.Repeat([]byte{0x5a}, 32)
)

func pqTestKeystore(t *testing.T, ext, seed []byte) []byte {
	t.Helper()
	out, err := EncryptPQKeyV3(pqTestScheme, pqTestAddress, ext, seed, []byte(pqTestPassword), LightScryptN, LightScryptP)
	require.NoError(t, err)
	return out
}

func pqDecodeMap(t *testing.T, b []byte) map[string]any {
	t.Helper()
	m := map[string]any{}
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

func pqKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// pqMutate decodes a keystore, lets f edit the top-level map and its "crypto"
// map, and re-encodes it.
func pqMutate(t *testing.T, b []byte, f func(top, crypto map[string]any)) []byte {
	t.Helper()
	top := pqDecodeMap(t, b)
	c, _ := top["crypto"].(map[string]any)
	f(top, c)
	out, err := json.Marshal(top)
	require.NoError(t, err)
	return out
}

// flipHex changes the first hex digit of s.
func flipHex(s string) string {
	if s[0] == '0' {
		return "1" + s[1:]
	}
	return "0" + s[1:]
}

func TestPQKeyV3_RoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		ext, seed  []byte
		cryptoKeys []string
	}{
		{"ext and seed", pqTestExt, pqTestSeed, []string{"cipher", "cipherparams", "ciphertext", "kdf", "kdfparams", "mac", "seedcipherparams", "seedciphertext", "seedmac"}},
		{"ext only", pqTestExt, nil, []string{"cipher", "cipherparams", "ciphertext", "kdf", "kdfparams", "mac"}},
		{"seed only", nil, pqTestSeed, []string{"cipher", "kdf", "kdfparams", "seedcipherparams", "seedciphertext", "seedmac"}},
		{"plaintext at the 4096-byte cap", bytes.Repeat([]byte{1}, maxPQPlaintextLen), nil, []string{"cipher", "cipherparams", "ciphertext", "kdf", "kdfparams", "mac"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := pqTestKeystore(t, tt.ext, tt.seed)

			top := pqDecodeMap(t, out)
			assert.Equal(t, []string{"address", "crypto", "id", "scheme", "version"}, pqKeys(top))
			assert.Equal(t, pqTestAddress, top["address"])
			assert.Equal(t, pqTestScheme, top["scheme"])
			assert.Equal(t, float64(3), top["version"])
			assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, top["id"])

			c, ok := top["crypto"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, tt.cryptoKeys, pqKeys(c))
			assert.Equal(t, "aes-128-ctr", c["cipher"])
			assert.Equal(t, "scrypt", c["kdf"])
			kp, ok := c["kdfparams"].(map[string]any)
			require.True(t, ok)
			assert.Equal(t, []string{"dklen", "n", "p", "r", "salt"}, pqKeys(kp))
			assert.Equal(t, float64(32), kp["dklen"])
			assert.Equal(t, float64(LightScryptN), kp["n"])
			assert.Equal(t, float64(LightScryptP), kp["p"])
			assert.Equal(t, float64(8), kp["r"])
			assert.Regexp(t, `^[0-9a-f]{64}$`, kp["salt"])

			var ivs []string
			for _, seg := range []struct{ ct, params, mac string }{
				{"ciphertext", "cipherparams", "mac"},
				{"seedciphertext", "seedcipherparams", "seedmac"},
			} {
				if _, ok := c[seg.ct]; !ok {
					continue
				}
				p, ok := c[seg.params].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, []string{"iv"}, pqKeys(p))
				assert.Regexp(t, `^[0-9a-f]{32}$`, p["iv"])
				assert.Regexp(t, `^[0-9a-f]{64}$`, c[seg.mac])
				assert.Regexp(t, `^[0-9a-f]+$`, c[seg.ct])
				ivs = append(ivs, p["iv"].(string))
			}
			if len(ivs) == 2 {
				assert.NotEqual(t, ivs[0], ivs[1], "segment IVs must differ")
			}

			scheme, addr, ext, seed, err := DecryptPQKeyV3(out, pqTestPassword)
			require.NoError(t, err)
			assert.Equal(t, pqTestScheme, scheme)
			assert.Equal(t, pqTestAddress, addr)
			assert.Equal(t, tt.ext, ext)
			assert.Equal(t, tt.seed, seed)
		})
	}
}

func TestEncryptPQKeyV3_Rejects(t *testing.T) {
	big := bytes.Repeat([]byte{1}, maxPQPlaintextLen+1)
	tests := []struct {
		name            string
		scheme, address string
		ext, seed       []byte
		n, p            int
	}{
		{"empty scheme", "", pqTestAddress, pqTestExt, nil, LightScryptN, LightScryptP},
		{"empty address", pqTestScheme, "", pqTestExt, nil, LightScryptN, LightScryptP},
		{"no segment", pqTestScheme, pqTestAddress, nil, nil, LightScryptN, LightScryptP},
		{"ext over cap", pqTestScheme, pqTestAddress, big, nil, LightScryptN, LightScryptP},
		{"seed over cap", pqTestScheme, pqTestAddress, nil, big, LightScryptN, LightScryptP},
		{"scrypt N above reader limit", pqTestScheme, pqTestAddress, pqTestExt, nil, maxScryptN * 2, 1},
		{"scrypt P zero", pqTestScheme, pqTestAddress, pqTestExt, nil, LightScryptN, 0},
		{"scrypt P above reader limit", pqTestScheme, pqTestAddress, pqTestExt, nil, LightScryptN, maxScryptP + 1},
		{"scrypt N not a power of two", pqTestScheme, pqTestAddress, pqTestExt, nil, 3000, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := EncryptPQKeyV3(tt.scheme, tt.address, tt.ext, tt.seed, []byte(pqTestPassword), tt.n, tt.p)
			require.Error(t, err)
			assert.Nil(t, out)
		})
	}
}

func TestECDSAPlaintextCapUnchanged(t *testing.T) {
	_, err := EncryptDataV3(make([]byte, maxCiphertextLen+1), []byte("x"), LightScryptN, LightScryptP)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds limit 1024")

	mac := strings.Repeat("00", macLen)
	iv := strings.Repeat("00", aesIVLen)
	_, _, _, err = decodeCipherFields(mac, iv, strings.Repeat("00", maxCiphertextLen+1), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds limit 2048")
	_, _, ct, err := decodeCipherFields(mac, iv, strings.Repeat("00", maxCiphertextLen), false)
	require.NoError(t, err)
	assert.Len(t, ct, maxCiphertextLen)
}

func TestDecryptPQKeyV3_Rejects(t *testing.T) {
	valid := pqTestKeystore(t, pqTestExt, pqTestSeed)
	tooLong := hex.EncodeToString(bytes.Repeat([]byte{1}, maxPQPlaintextLen+1))

	tests := []struct {
		name     string
		mutate   func(top, c map[string]any)
		password string
		wantErr  error  // checked with errors.Is when set
		contains string // checked against the message when set
		raw      []byte // replaces the keystore when set
	}{
		{name: "wrong password", password: "wrong", wantErr: ErrDecrypt},
		{name: "tampered ciphertext", mutate: func(_, c map[string]any) { c["ciphertext"] = flipHex(c["ciphertext"].(string)) }, wantErr: ErrDecrypt},
		{name: "tampered seedciphertext", mutate: func(_, c map[string]any) { c["seedciphertext"] = flipHex(c["seedciphertext"].(string)) }, wantErr: ErrDecrypt},
		{name: "tampered mac", mutate: func(_, c map[string]any) { c["mac"] = flipHex(c["mac"].(string)) }, wantErr: ErrDecrypt},
		{name: "tampered seedmac", mutate: func(_, c map[string]any) { c["seedmac"] = flipHex(c["seedmac"].(string)) }, wantErr: ErrDecrypt},
		{name: "equal IVs", mutate: func(_, c map[string]any) { c["seedcipherparams"] = c["cipherparams"] }, contains: "IV"},
		{name: "seed segment missing seedmac", mutate: func(_, c map[string]any) { delete(c, "seedmac") }, contains: "incomplete"},
		{name: "seed segment missing iv", mutate: func(_, c map[string]any) { c["seedcipherparams"] = map[string]any{} }, contains: "incomplete"},
		{name: "ext segment missing cipherparams", mutate: func(_, c map[string]any) { delete(c, "cipherparams") }, contains: "incomplete"},
		{name: "ext segment missing ciphertext", mutate: func(_, c map[string]any) { delete(c, "ciphertext") }, contains: "incomplete"},
		{name: "no segment", mutate: func(_, c map[string]any) {
			for _, k := range []string{"ciphertext", "cipherparams", "mac", "seedciphertext", "seedcipherparams", "seedmac"} {
				delete(c, k)
			}
		}, contains: "no encrypted segment"},
		{name: "empty ciphertext", mutate: func(_, c map[string]any) { c["ciphertext"] = "" }, contains: "empty ciphertext"},
		{name: "ciphertext over cap", mutate: func(_, c map[string]any) { c["ciphertext"] = tooLong }, contains: "exceeds limit"},
		{name: "missing scheme", mutate: func(top, _ map[string]any) { delete(top, "scheme") }, contains: "scheme"},
		{name: "version 2", mutate: func(top, _ map[string]any) { top["version"] = 2 }, contains: "version"},
		{name: "missing version", mutate: func(top, _ map[string]any) { delete(top, "version") }, contains: "version"},
		{name: "missing crypto", mutate: func(top, _ map[string]any) { delete(top, "crypto") }, contains: "crypto"},
		{name: "cipher aes-128-cbc", mutate: func(_, c map[string]any) { c["cipher"] = "aes-128-cbc" }, contains: "cipher"},
		{name: "kdf pbkdf2", mutate: func(_, c map[string]any) {
			c["kdf"] = "pbkdf2"
			kp := c["kdfparams"].(map[string]any)
			kp["c"] = 262144
			kp["prf"] = "hmac-sha256"
		}, contains: "kdf"},
		{name: "scrypt N above limit", mutate: func(_, c map[string]any) { c["kdfparams"].(map[string]any)["n"] = maxScryptN * 2 }, contains: "kdf params"},
		{name: "not JSON", raw: []byte("{not json")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid
			if tt.mutate != nil {
				in = pqMutate(t, valid, tt.mutate)
			}
			if tt.raw != nil {
				in = tt.raw
			}
			pw := pqTestPassword
			if tt.password != "" {
				pw = tt.password
			}
			scheme, addr, ext, seed, err := DecryptPQKeyV3(in, pw)
			require.Error(t, err)
			if tt.wantErr != nil {
				assert.True(t, errors.Is(err, tt.wantErr), "got %v", err)
			} else {
				assert.False(t, errors.Is(err, ErrDecrypt), "structural error reported as wrong password: %v", err)
			}
			if tt.contains != "" {
				assert.Contains(t, err.Error(), tt.contains)
			}
			assert.Empty(t, scheme)
			assert.Empty(t, addr)
			assert.Nil(t, ext, "no plaintext may be returned on error")
			assert.Nil(t, seed, "no plaintext may be returned on error")
		})
	}
}

func TestDecryptPQKeyV3_Accepts(t *testing.T) {
	valid := pqTestKeystore(t, pqTestExt, pqTestSeed)
	tests := []struct {
		name     string
		mutate   func(top, c map[string]any)
		wantExt  []byte
		wantSeed []byte
	}{
		{"Crypto alias", func(top, c map[string]any) {
			delete(top, "crypto")
			top["Crypto"] = c
		}, pqTestExt, pqTestSeed},
		// Jackson serialises absent ext fields as explicit nulls.
		{"explicit null ext fields", func(_, c map[string]any) {
			c["ciphertext"], c["cipherparams"], c["mac"] = nil, nil, nil
		}, nil, pqTestSeed},
		{"unknown fields ignored", func(top, c map[string]any) {
			top["name"] = nil
			c["extra"] = "x"
		}, pqTestExt, pqTestSeed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ext, seed, err := DecryptPQKeyV3(pqMutate(t, valid, tt.mutate), pqTestPassword)
			require.NoError(t, err)
			assert.Equal(t, tt.wantExt, ext)
			assert.Equal(t, tt.wantSeed, seed)
		})
	}
}
