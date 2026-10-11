package pqc

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// falconKATSHA256 is the SHA-256 of the uncompressed BouncyCastle
// falcon512-KAT.rsp (see testdata/README.md).
const falconKATSHA256 = "dd75c946fdedef4ec46a2bee7e10c65c9126f1a839b9ced6921fd45f7354b5cd"

// falconKAT is one entry of the BouncyCastle Falcon-512 KAT file. pk is the
// 897-byte reference public key (0x09 || h), sk the 1281-byte reference
// private key (0x59 || f || g || F) and sm the NIST signed message
// siglen(2, big-endian) || nonce(40) || msg || 0x29 || compressed s2.
type falconKAT struct {
	count  int
	mlen   int
	smlen  int
	msg    []byte
	pk     []byte
	sk     []byte
	sm     []byte
	parsed int // number of fields seen, for sanity
}

// loadFalconKAT reads testdata/falcon512-KAT.rsp.gz and checks the SHA-256 of
// the uncompressed file before parsing any of it.
func loadFalconKAT(t *testing.T) []falconKAT {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "falcon512-KAT.rsp.gz"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, f.Close()) })
	zr, err := gzip.NewReader(f)
	require.NoError(t, err)
	raw, err := io.ReadAll(zr)
	require.NoError(t, err)
	require.NoError(t, zr.Close())
	sum := sha256.Sum256(raw)
	require.Equal(t, falconKATSHA256, hex.EncodeToString(sum[:]), "KAT file does not match the pinned SHA-256")

	var out []falconKAT
	var cur *falconKAT
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		key, val, ok := strings.Cut(line, " = ")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		if key == "count" {
			n, err := strconv.Atoi(val)
			require.NoError(t, err)
			out = append(out, falconKAT{count: n})
			cur = &out[len(out)-1]
			continue
		}
		require.NotNil(t, cur, "field %q before the first count", key)
		switch key {
		case "mlen", "smlen":
			n, err := strconv.Atoi(val)
			require.NoError(t, err)
			if key == "mlen" {
				cur.mlen = n
			} else {
				cur.smlen = n
			}
		case "msg", "pk", "sk", "sm":
			b, err := hex.DecodeString(val)
			require.NoError(t, err)
			switch key {
			case "msg":
				cur.msg = b
			case "pk":
				cur.pk = b
			case "sk":
				cur.sk = b
			case "sm":
				cur.sm = b
			}
		case "seed":
		default:
			t.Fatalf("unexpected KAT field %q", key)
		}
		cur.parsed++
	}
	return out
}
