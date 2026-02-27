package keystore_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// constReader is an io.Reader that returns bytes filled with a constant value.
type constReader byte

func (c constReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(c)
	}
	return len(p), nil
}

// ---------- TextHash / TextAndHash ----------

func TestTextHash(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		useFixedLength []bool
		wantHash       []byte // expected keccak256 of the prefixed message
	}{
		{
			name: "simple message",
			data: []byte("hello"),
			// "\x19TRON Signed Message:\n5hello"
			wantHash: common.Keccak256([]byte("\x19TRON Signed Message:\n5hello")),
		},
		{
			name: "empty message",
			data: []byte{},
			// "\x19TRON Signed Message:\n0"
			wantHash: common.Keccak256([]byte("\x19TRON Signed Message:\n0")),
		},
		{
			name:           "fixed length flag true uses 32",
			data:           []byte("short"),
			useFixedLength: []bool{true},
			wantHash:       common.Keccak256([]byte("\x19TRON Signed Message:\n32short")),
		},
		{
			name:           "fixed length flag false uses actual length",
			data:           []byte("short"),
			useFixedLength: []bool{false},
			wantHash:       common.Keccak256([]byte("\x19TRON Signed Message:\n5short")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := keystore.TextHash(tt.data, tt.useFixedLength...)
			require.Len(t, hash, 32)
			assert.Equal(t, tt.wantHash, hash)
		})
	}
}

func TestTextHash_deterministic(t *testing.T) {
	data := []byte("deterministic check")
	h1 := keystore.TextHash(data)
	h2 := keystore.TextHash(data)
	assert.Equal(t, h1, h2, "same input must produce identical hash")
}

func TestTextHash_knownValue(t *testing.T) {
	// Manually compute the expected hash:
	// msg = "\x19TRON Signed Message:\n5hello"
	data := []byte("hello")
	msg := fmt.Sprintf("\x19TRON Signed Message:\n%d%s", len(data), string(data))
	expected := common.Keccak256([]byte(msg))

	got := keystore.TextHash(data)
	assert.Equal(t, expected, got)
}

func TestTextAndHash(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		useFixedLength []bool
		wantMsg        string
	}{
		{
			name:    "standard message",
			data:    []byte("hello"),
			wantMsg: "\x19TRON Signed Message:\n5hello",
		},
		{
			name:    "empty message",
			data:    []byte{},
			wantMsg: "\x19TRON Signed Message:\n0",
		},
		{
			name:           "fixed length uses 32",
			data:           []byte("abc"),
			useFixedLength: []bool{true},
			wantMsg:        "\x19TRON Signed Message:\n32abc",
		},
		{
			name:           "fixed length false uses actual length",
			data:           []byte("abc"),
			useFixedLength: []bool{false},
			wantMsg:        "\x19TRON Signed Message:\n3abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, msg := keystore.TextAndHash(tt.data, tt.useFixedLength...)
			assert.Equal(t, tt.wantMsg, msg)
			assert.Len(t, hash, 32)

			// Hash must match keccak256 of the returned message.
			expected := common.Keccak256([]byte(msg))
			assert.Equal(t, expected, hash)
		})
	}
}

// ---------- UnmarshalPublic ----------

func TestUnmarshalPublic(t *testing.T) {
	t.Run("wrong curve public key", func(t *testing.T) {
		// Build an uncompressed public key from the P256 curve.
		// go-ethereum's UnmarshalPubkey expects secp256k1, so this should fail.
		key, err := ecdsa.GenerateKey(elliptic.P256(), constReader(0xAB))
		require.NoError(t, err)
		uncompressed := make([]byte, 65)
		uncompressed[0] = 0x04
		xBytes := key.X.Bytes()
		yBytes := key.Y.Bytes()
		copy(uncompressed[1+32-len(xBytes):33], xBytes)
		copy(uncompressed[33+32-len(yBytes):65], yBytes)

		_, err = keystore.UnmarshalPublic(uncompressed)
		assert.Error(t, err)
	})

	t.Run("invalid bytes", func(t *testing.T) {
		_, err := keystore.UnmarshalPublic([]byte{0x00, 0x01})
		assert.Error(t, err)
	})

	t.Run("empty bytes", func(t *testing.T) {
		_, err := keystore.UnmarshalPublic([]byte{})
		assert.Error(t, err)
	})

	t.Run("nil bytes", func(t *testing.T) {
		_, err := keystore.UnmarshalPublic(nil)
		assert.Error(t, err)
	})

	t.Run("valid secp256k1 key round-trip", func(t *testing.T) {
		// Generate a real secp256k1 key via the keystore, then round-trip
		// through marshal/unmarshal to test UnmarshalPublic with valid input.
		ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
		acct, err := ks.NewAccount("test")
		require.NoError(t, err)

		// Export the key to get the raw ECDSA key
		exported, err := ks.Export(acct, "test", "test")
		require.NoError(t, err)

		decrypted, err := keystore.DecryptKey(exported, "test")
		require.NoError(t, err)

		// Marshal the public key to uncompressed form (04 || X || Y)
		pub := decrypted.PrivateKey.PublicKey
		uncompressed := make([]byte, 65)
		uncompressed[0] = 0x04
		xBytes := pub.X.Bytes()
		yBytes := pub.Y.Bytes()
		copy(uncompressed[1+32-len(xBytes):33], xBytes)
		copy(uncompressed[33+32-len(yBytes):65], yBytes)

		// UnmarshalPublic should succeed and return a matching key
		recovered, err := keystore.UnmarshalPublic(uncompressed)
		require.NoError(t, err)
		assert.Equal(t, pub.X, recovered.X)
		assert.Equal(t, pub.Y, recovered.Y)
	})
}

// ---------- URL ----------

func TestURL_String(t *testing.T) {
	tests := []struct {
		name   string
		url    keystore.URL
		expect string
	}{
		{
			name:   "scheme and path",
			url:    keystore.URL{Scheme: "keystore", Path: "/path/to/key"},
			expect: "keystore:///path/to/key",
		},
		{
			name:   "empty scheme returns path only",
			url:    keystore.URL{Scheme: "", Path: "/some/path"},
			expect: "/some/path",
		},
		{
			name:   "both empty",
			url:    keystore.URL{Scheme: "", Path: ""},
			expect: "",
		},
		{
			name:   "scheme with empty path",
			url:    keystore.URL{Scheme: "https", Path: ""},
			expect: "https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, tt.url.String())
		})
	}
}

func TestURL_Cmp(t *testing.T) {
	tests := []struct {
		name string
		a    keystore.URL
		b    keystore.URL
		want int
	}{
		{
			name: "equal URLs",
			a:    keystore.URL{Scheme: "keystore", Path: "/a"},
			b:    keystore.URL{Scheme: "keystore", Path: "/a"},
			want: 0,
		},
		{
			name: "same scheme, path a < b",
			a:    keystore.URL{Scheme: "keystore", Path: "/a"},
			b:    keystore.URL{Scheme: "keystore", Path: "/b"},
			want: -1,
		},
		{
			name: "same scheme, path a > b",
			a:    keystore.URL{Scheme: "keystore", Path: "/b"},
			b:    keystore.URL{Scheme: "keystore", Path: "/a"},
			want: 1,
		},
		{
			name: "different schemes, compare by scheme",
			a:    keystore.URL{Scheme: "aaa", Path: "/z"},
			b:    keystore.URL{Scheme: "zzz", Path: "/a"},
			want: -1,
		},
		{
			name: "different schemes reversed",
			a:    keystore.URL{Scheme: "zzz", Path: "/a"},
			b:    keystore.URL{Scheme: "aaa", Path: "/z"},
			want: 1,
		},
		{
			name: "both empty",
			a:    keystore.URL{},
			b:    keystore.URL{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Cmp(tt.b))
		})
	}
}

func TestURL_MarshalJSON_UnmarshalJSON_roundTrip(t *testing.T) {
	tests := []struct {
		name string
		url  keystore.URL
	}{
		{
			name: "typical URL",
			url:  keystore.URL{Scheme: "keystore", Path: "/home/user/.tronctl/keystore/key"},
		},
		{
			name: "https URL",
			url:  keystore.URL{Scheme: "https", Path: "example.com/key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.url)
			require.NoError(t, err)

			var decoded keystore.URL
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.url.Scheme, decoded.Scheme)
			assert.Equal(t, tt.url.Path, decoded.Path)
		})
	}
}

func TestURL_MarshalJSON_format(t *testing.T) {
	u := keystore.URL{Scheme: "keystore", Path: "/tmp/key"}
	data, err := json.Marshal(u)
	require.NoError(t, err)

	// Should produce a JSON string like "keystore:///tmp/key".
	assert.Equal(t, `"keystore:///tmp/key"`, string(data))
}

func TestURL_UnmarshalJSON_errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "missing scheme",
			input: `"/just/a/path"`,
		},
		{
			name:  "no separator",
			input: `"noscheme"`,
		},
		{
			name:  "invalid JSON",
			input: `not-json`,
		},
		{
			name:  "empty scheme with separator",
			input: `"://path"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var u keystore.URL
			err := json.Unmarshal([]byte(tt.input), &u)
			assert.Error(t, err)
		})
	}
}

func TestURL_TerminalString(t *testing.T) {
	t.Run("short URL returned as-is", func(t *testing.T) {
		u := keystore.URL{Scheme: "ks", Path: "/a"}
		assert.Equal(t, "ks:///a", u.TerminalString())
	})

	t.Run("long URL is truncated", func(t *testing.T) {
		// Build a URL longer than 32 characters.
		longPath := "/this/is/a/very/long/path/that/exceeds/32/characters"
		u := keystore.URL{Scheme: "keystore", Path: longPath}
		ts := u.TerminalString()
		// TerminalString truncates to 31 chars + ellipsis.
		assert.LessOrEqual(t, len([]rune(ts)), 32)
		assert.Equal(t, "…", string([]rune(ts)[len([]rune(ts))-1]))
		assert.Equal(t, u.String()[:31], string([]rune(ts)[:31]))
	})
}
