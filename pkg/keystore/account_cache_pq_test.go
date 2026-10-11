package keystore

// This is an internal test: it drives the per-file keystore reader directly,
// which the scan calls in random order. It cannot import pkg/pqc (which
// imports this package), so PQ keystores come from EncryptPQKeyV3.

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// fn wrote to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	orig := os.Stdout
	os.Stdout = w
	done := make(chan []byte)
	go func() {
		b, readErr := io.ReadAll(r)
		if readErr != nil {
			b = append(b, []byte("read error: "+readErr.Error())...)
		}
		done <- b
	}()
	defer func() {
		os.Stdout = orig
	}()
	fn()
	os.Stdout = orig
	require.NoError(t, w.Close())
	out := <-done
	require.NoError(t, r.Close())
	return string(out)
}

// newECDSAKeyFiles creates n ECDSA V3 keystore files in dir and returns
// their accounts.
func newECDSAKeyFiles(t *testing.T, dir string, n int) []Account {
	t.Helper()
	ks := NewKeyStore(dir, LightScryptN, LightScryptP)
	defer func() {
		ks.Close()
		runtime.Gosched()
	}()
	accts := make([]Account, 0, n)
	for range n {
		a, err := ks.NewAccount("pw")
		require.NoError(t, err)
		accts = append(accts, a)
	}
	return accts
}

// setSchemeField rewrites the keystore file at path with a "scheme" field
// set to value (nil writes JSON null).
func setSchemeField(t *testing.T, path string, value any) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(data, &m))
	m["scheme"] = value
	out, err := json.Marshal(m)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, out, 0600))
}

// A post-quantum keystore sharing a directory with ECDSA keystores is
// skipped without printing, and the ECDSA account is still listed. The scan
// order is random, so this does not prove a PQ file's scheme cannot leak into
// the next file; TestAccountCache_ReaderResetsSchemeBetweenFiles does.
func TestAccountCache_SkipsPQKeystoreQuietly(t *testing.T) {
	dir := t.TempDir()
	ecdsaAcct := newECDSAKeyFiles(t, dir, 1)[0]
	require.NoError(t, os.WriteFile(filepath.Join(dir, "UTC--pq.json"),
		pqTestKeystore(t, pqTestExt, pqTestSeed), 0600))

	var accts []Account
	out := captureStdout(t, func() {
		ks := NewKeyStore(dir, LightScryptN, LightScryptP)
		accts = ks.Accounts()
		ks.Close()
		runtime.Gosched()
	})

	assert.Empty(t, out, "scanning a PQ keystore must not print to stdout")
	require.Len(t, accts, 1)
	assert.Equal(t, ecdsaAcct.Address.String(), accts[0].Address.String())
}

// One reader reads a PQ file and then ECDSA files, in that fixed order. The
// PQ file's scheme must not carry over and make the reader skip them.
func TestAccountCache_ReaderResetsSchemeBetweenFiles(t *testing.T) {
	dir := t.TempDir()
	ecdsa := newECDSAKeyFiles(t, dir, 2)
	pqPath := filepath.Join(dir, "UTC--pq.json")
	require.NoError(t, os.WriteFile(pqPath, pqTestKeystore(t, pqTestExt, pqTestSeed), 0600))
	// The second ECDSA file has "scheme":null, which leaves the decode
	// target untouched, so only an explicit reset clears it.
	setSchemeField(t, ecdsa[1].URL.Path, nil)

	r := newKeyFileReader()
	out := captureStdout(t, func() {
		for _, want := range ecdsa {
			assert.Nil(t, r.read(pqPath), "a PQ keystore is skipped")
			got := r.read(want.URL.Path)
			if assert.NotNil(t, got, "ECDSA file read after a PQ file must be accepted: %s", want.URL.Path) {
				assert.Equal(t, want.Address.String(), got.Address.String())
				assert.Equal(t, want.URL.Path, got.URL.Path)
			}
		}
	})
	assert.Empty(t, out)
}

// ECDSA V3 files with an empty or null "scheme" field are ordinary ECDSA
// keystores: they are listed and nothing is printed.
func TestAccountCache_EmptyOrNullSchemeIsECDSA(t *testing.T) {
	dir := t.TempDir()
	ecdsa := newECDSAKeyFiles(t, dir, 2)
	setSchemeField(t, ecdsa[0].URL.Path, "")
	setSchemeField(t, ecdsa[1].URL.Path, nil)

	var accts []Account
	out := captureStdout(t, func() {
		ks := NewKeyStore(dir, LightScryptN, LightScryptP)
		accts = ks.Accounts()
		ks.Close()
		runtime.Gosched()
	})

	assert.Empty(t, out, "an empty or null scheme must not print to stdout")
	got := make([]string, 0, len(accts))
	for _, a := range accts {
		got = append(got, a.Address.String())
	}
	assert.ElementsMatch(t, []string{ecdsa[0].Address.String(), ecdsa[1].Address.String()}, got)
}
