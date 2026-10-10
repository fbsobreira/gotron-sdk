//go:build falcon

package pqc

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// detachedFromSM converts a NIST signed message into the TRON detached
// signature 0x39 || nonce || compressed s2.
func detachedFromSM(t *testing.T, e falconKAT) []byte {
	t.Helper()
	sm := e.sm
	require.GreaterOrEqual(t, len(sm), 2+40+e.mlen+1)
	siglen := int(sm[0])<<8 | int(sm[1])
	require.Equal(t, 2+40+e.mlen+siglen, len(sm), "count %d: smlen", e.count)
	require.Equal(t, e.msg, sm[42:42+e.mlen], "count %d: embedded message", e.count)
	require.Equal(t, byte(0x29), sm[42+e.mlen], "count %d: compressed-signature header", e.count)
	sig := make([]byte, 0, 40+siglen)
	sig = append(sig, FNDSA512SignatureHeader)
	sig = append(sig, sm[2:42]...)
	sig = append(sig, sm[42+e.mlen+1:]...)
	require.Len(t, sig, 40+siglen)
	return sig
}

func randomDigest(t *testing.T) []byte {
	t.Helper()
	d := make([]byte, DigestSize)
	_, err := rand.Read(d)
	require.NoError(t, err)
	return d
}

func TestFalconLiboqs_Available(t *testing.T) {
	require.NoError(t, falcon.availErr(), "liboqs Falcon-512 backend must be available with -tags falcon")
}

// TestFalconLiboqs_BouncyCastleKAT checks the backend against all 100 entries
// of BouncyCastle 1.84's Falcon-512 KAT: every reference signature verifies,
// a one-byte change does not, and the BC-format key pair signs a fresh digest
// that verifies (proving the 0x59 / 0x09 header handling).
func TestFalconLiboqs_BouncyCastleKAT(t *testing.T) {
	kats := loadFalconKAT(t)
	require.Len(t, kats, 100)

	verified, rejected, signed := 0, 0, 0
	for i, e := range kats {
		require.Equal(t, i, e.count)
		require.Equal(t, 7, e.parsed, "count %d: fields after count", e.count)
		require.Len(t, e.msg, e.mlen, "count %d", e.count)
		require.Len(t, e.sm, e.smlen, "count %d", e.count)
		require.Len(t, e.pk, FNDSA512PublicKeySize+1, "count %d", e.count)
		require.Equal(t, byte(fnPublicKeyHeader), e.pk[0], "count %d", e.count)
		require.Len(t, e.sk, FNDSA512PrivateKeySize+1, "count %d", e.count)
		require.Equal(t, byte(fnPrivateKeyHeader), e.sk[0], "count %d", e.count)

		sig := detachedFromSM(t, e)
		require.True(t, fnSignatureWellFormed(sig), "count %d: %d-byte KAT signature outside the TRON bounds", e.count, len(sig))

		pk := e.pk[1:]
		if falcon.verify(bytes.Clone(pk), bytes.Clone(e.msg), bytes.Clone(sig)) {
			verified++
		} else {
			t.Errorf("count %d: KAT signature did not verify", e.count)
		}

		bad := bytes.Clone(sig)
		bad[1+(e.count*7)%(len(bad)-1)] ^= 0x01
		if !falcon.verify(bytes.Clone(pk), bytes.Clone(e.msg), bad) {
			rejected++
		} else {
			t.Errorf("count %d: mutated KAT signature verified", e.count)
		}

		digest := sha256.Sum256(e.sm)
		mine, err := falcon.sign(bytes.Clone(e.sk[1:]), digest[:])
		require.NoError(t, err, "count %d", e.count)
		require.True(t, fnSignatureWellFormed(mine), "count %d: %d-byte signature", e.count, len(mine))
		if falcon.verify(bytes.Clone(pk), digest[:], mine) {
			signed++
		} else {
			t.Errorf("count %d: signature from the BC private key did not verify", e.count)
		}
	}
	assert.Equal(t, 100, verified, "KAT signatures verified")
	assert.Equal(t, 100, rejected, "mutated KAT signatures rejected")
	assert.Equal(t, 100, signed, "BC-key signatures verified")
	t.Logf("BouncyCastle KAT: %d/100 verified, %d/100 mutations rejected, %d/100 BC-key signatures verified",
		verified, rejected, signed)
}

// TestFalconLiboqs_NileFixture verifies the nile-testnet PR #81 FN-DSA-512
// conformance signature through the public API.
func TestFalconLiboqs_NileFixture(t *testing.T) {
	msg := mustHex(t, fnFixtureMessageHex)
	sig := mustHex(t, fnFixtureSigHex)
	require.Len(t, sig, 657)

	for name, pkHex := range map[string]string{"bare": fnFixturePubKeyHex, "reference": fnFixtureRefPubKeyHex} {
		t.Run(name, func(t *testing.T) {
			pub, err := ParsePublicKey(core.PQScheme_FN_DSA_512, mustHex(t, pkHex))
			require.NoError(t, err)
			assert.Equal(t, fnFixtureAddress, pub.Address().String())

			ok, err := pub.Verify(msg, sig)
			require.NoError(t, err)
			assert.True(t, ok)

			for _, i := range []int{1, 40, 41, 300, len(sig) - 1} {
				bad := bytes.Clone(sig)
				bad[i] ^= 0x80
				ok, err := pub.Verify(msg, bad)
				require.NoError(t, err)
				assert.False(t, ok, "mutated byte %d", i)
			}

			other := bytes.Clone(msg)
			other[0] ^= 0x01
			ok, err = pub.Verify(other, sig)
			require.NoError(t, err)
			assert.False(t, ok, "different digest")
		})
	}
}

// TestFalconLiboqs_TronGrpcPair runs the real pair probe on the tron-grpc key
// pair and signs 200 random digests with it.
func TestFalconLiboqs_TronGrpcPair(t *testing.T) {
	skBytes := mustHex(t, fnPairPrivKeyHex)
	pkBytes := mustHex(t, fnPairPubKeyHex)

	k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, skBytes, pkBytes)
	require.NoError(t, err)
	assert.Equal(t, fnPairAddress, k.Public().Address().String())

	withHeader := append([]byte{fnPrivateKeyHeader}, skBytes...)
	_, err = ParsePrivateKey(core.PQScheme_FN_DSA_512, withHeader, pkBytes)
	require.NoError(t, err, "0x59-header private key")
	_, err = ParsePrivateKey(core.PQScheme_FN_DSA_512, append(bytes.Clone(skBytes), pkBytes...), nil)
	require.NoError(t, err, "extended f||g||F||h private key")

	// fnFixturePubKeyHex is the same key as fnPairPubKeyHex, so the mismatch
	// cases use a freshly generated key and a BouncyCastle KAT key.
	other, err := GenerateKey(core.PQScheme_FN_DSA_512, nil)
	require.NoError(t, err)
	_, err = ParsePrivateKey(core.PQScheme_FN_DSA_512, skBytes, other.Public().Bytes())
	require.ErrorIs(t, err, ErrKeyMismatch)
	kat := loadFalconKAT(t)[0]
	_, err = ParsePrivateKey(core.PQScheme_FN_DSA_512, skBytes, kat.pk)
	require.ErrorIs(t, err, ErrKeyMismatch)
	_, err = ParsePrivateKey(core.PQScheme_FN_DSA_512, append(bytes.Clone(skBytes), kat.pk[1:]...), nil)
	require.ErrorIs(t, err, ErrKeyMismatch)

	const n = 200
	lengths := make([]int, 0, n)
	for i := 0; i < n; i++ {
		d := randomDigest(t)
		sig, err := k.Sign(d)
		require.NoError(t, err)
		require.Equal(t, byte(FNDSA512SignatureHeader), sig[0])
		require.GreaterOrEqual(t, len(sig), FNDSA512SignatureMinSize)
		require.LessOrEqual(t, len(sig), FNDSA512SignatureMaxSize)
		lengths = append(lengths, len(sig))

		ok, err := k.Public().Verify(d, sig)
		require.NoError(t, err)
		require.True(t, ok, "signature %d", i)

		ok, err = other.Public().Verify(d, sig)
		require.NoError(t, err)
		require.False(t, ok, "cross-key verify %d", i)
	}
	t.Logf("FN-DSA-512 signature lengths over %d signs: %s", n, lengthSummary(lengths))
}

// TestFalconLiboqs_MalformedKeysReturnPromptly runs the malformed pairs,
// including the shapes that made liboqs signing loop forever in the pair
// probe (all-0xff, F = f), through ParsePrivateKey with the real backend:
// each must be rejected promptly. The 10 s limit (and -timeout 120s in
// make test-falcon) turns a regression into a failure instead of a hang.
func TestFalconLiboqs_MalformedKeysReturnPromptly(t *testing.T) {
	for _, tt := range fnMalformedPairs(t) {
		t.Run(tt.name, func(t *testing.T) {
			done := make(chan error, 1)
			go func() {
				_, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, bytes.Clone(tt.sk), bytes.Clone(tt.pk))
				done <- err
			}()
			select {
			case err := <-done:
				require.Error(t, err)
				if tt.want != nil {
					require.ErrorIs(t, err, tt.want)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("ParsePrivateKey did not return within 10 s")
			}
		})
	}
}

// lengthSummary renders min/median/max/mean and a histogram of lengths.
func lengthSummary(lengths []int) string {
	s := append([]int(nil), lengths...)
	sort.Ints(s)
	total := 0
	hist := map[int]int{}
	for _, l := range s {
		total += l
		hist[l]++
	}
	keys := make([]int, 0, len(hist))
	for l := range hist {
		keys = append(keys, l)
	}
	sort.Ints(keys)
	var b strings.Builder
	for _, l := range keys {
		fmt.Fprintf(&b, " %d:%d", l, hist[l])
	}
	return fmt.Sprintf("min=%d median=%d max=%d mean=%.1f hist=[%s ]",
		s[0], s[len(s)/2], s[len(s)-1], float64(total)/float64(len(s)), b.String())
}

// TestFalconLiboqs_GenerateKey checks a generated key end to end, including
// the toolkit JSON and encrypted keystore round trips, which now run the real
// pair probe.
func TestFalconLiboqs_GenerateKey(t *testing.T) {
	k, err := GenerateKey(core.PQScheme_FN_DSA_512, nil)
	require.NoError(t, err)
	require.Equal(t, core.PQScheme_FN_DSA_512, k.Scheme())
	require.Len(t, k.Bytes(), FNDSA512PrivateKeySize)
	require.Len(t, k.Public().Bytes(), FNDSA512PublicKeySize)
	assert.Nil(t, k.Seed())

	pk := k.Public().Bytes()
	want := address.BytesToAddress(common.Keccak256(pk)[12:])
	assert.Equal(t, want.String(), k.Public().Address().String())
	reparsed, err := ParsePublicKey(core.PQScheme_FN_DSA_512, pk)
	require.NoError(t, err)
	assert.True(t, reparsed.Equal(k.Public()))

	d := randomDigest(t)
	sig, err := k.Sign(d)
	require.NoError(t, err)
	ok, err := k.Public().Verify(d, sig)
	require.NoError(t, err)
	assert.True(t, ok)

	k2, err := GenerateKey(core.PQScheme_FN_DSA_512, nil)
	require.NoError(t, err)
	assert.False(t, k2.Public().Equal(k.Public()), "two generated keys must differ")
	ok, err = k2.Public().Verify(d, sig)
	require.NoError(t, err)
	assert.False(t, ok)

	check := func(t *testing.T, got PrivateKey) {
		t.Helper()
		assert.Equal(t, k.Bytes(), got.Bytes())
		assert.True(t, got.Public().Equal(k.Public()))
		assert.Equal(t, k.Public().Address().String(), got.Public().Address().String())
		d := randomDigest(t)
		sig, err := got.Sign(d)
		require.NoError(t, err)
		ok, err := k.Public().Verify(d, sig)
		require.NoError(t, err)
		assert.True(t, ok)
	}

	t.Run("toolkit JSON", func(t *testing.T) {
		js, err := MarshalToolkitJSON(k)
		require.NoError(t, err)
		got, err := ParseToolkitJSON(js)
		require.NoError(t, err)
		check(t, got)
	})
	t.Run("keystore", func(t *testing.T) {
		data, err := EncryptKeystore(k, "falcon-test-password", keystore.LightScryptN, keystore.LightScryptP)
		require.NoError(t, err)
		got, err := DecryptKeystore(data, "falcon-test-password")
		require.NoError(t, err)
		check(t, got)
	})
}

// TestFalconLiboqs_Concurrent signs and verifies from 8 goroutines sharing
// the one OQS_SIG object; run with -race.
func TestFalconLiboqs_Concurrent(t *testing.T) {
	k, err := ParsePrivateKey(core.PQScheme_FN_DSA_512, mustHex(t, fnPairPrivKeyHex), mustHex(t, fnPairPubKeyHex))
	require.NoError(t, err)

	const workers, rounds = 8, 20
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
		good int
	)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				d := sha256.Sum256([]byte(fmt.Sprintf("worker %d round %d", w, r)))
				sig, err := k.Sign(d[:])
				var ok bool
				if err == nil {
					ok, err = k.Public().Verify(d[:], sig)
				}
				mu.Lock()
				switch {
				case err != nil:
					errs = append(errs, fmt.Errorf("worker %d round %d: %w", w, r, err))
				case !ok:
					errs = append(errs, fmt.Errorf("worker %d round %d: signature did not verify", w, r))
				default:
					good++
				}
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()
	require.Empty(t, errs)
	assert.Equal(t, workers*rounds, good)
}

// TestFalconLiboqs_InputValidation feeds the backend malformed inputs: it
// must return errors or false, never panic or crash.
func TestFalconLiboqs_InputValidation(t *testing.T) {
	sk := mustHex(t, fnPairPrivKeyHex)
	pk := mustHex(t, fnPairPubKeyHex)
	msg := mustHex(t, fnFixtureMessageHex)
	sig := mustHex(t, fnFixtureSigHex)
	require.True(t, falcon.verify(bytes.Clone(pk), msg, sig))

	for _, n := range []int{0, 1, FNDSA512PrivateKeySize - 1, FNDSA512PrivateKeySize + 1, FNDSA512ExtendedPrivateKeySize} {
		got, err := falcon.sign(make([]byte, n), msg)
		assert.Error(t, err, "sk of %d bytes", n)
		assert.Nil(t, got)
	}
	for _, n := range []int{0, 1, FNDSA512PublicKeySize - 1, FNDSA512PublicKeySize + 1} {
		assert.False(t, falcon.verify(make([]byte, n), msg, sig), "pk of %d bytes", n)
	}
	refPK := append([]byte{fnPublicKeyHeader}, pk...)
	assert.False(t, falcon.verify(refPK, msg, sig), "897-byte pk must be rejected by the backend")

	for name, s := range map[string][]byte{
		"nil":          nil,
		"empty":        {},
		"header only":  {FNDSA512SignatureHeader},
		"nonce only":   sig[:41],
		"truncated":    sig[:len(sig)-1],
		"zero-padded":  append(bytes.Clone(sig), 0),
		"huge":         append(bytes.Clone(sig), make([]byte, 4096)...),
		"wrong header": append([]byte{0x3a}, sig[1:]...),
		"padded form":  append([]byte{0x29}, sig[1:]...),
	} {
		assert.False(t, falcon.verify(bytes.Clone(pk), msg, s), name)
	}

	// Messages of any length, including empty, are passed through.
	for _, m := range [][]byte{nil, {}, {0x01}, bytes.Repeat([]byte{0xab}, 1000)} {
		s, err := falcon.sign(bytes.Clone(sk), m)
		require.NoError(t, err, "message of %d bytes", len(m))
		assert.True(t, falcon.verify(bytes.Clone(pk), m, s), "message of %d bytes", len(m))
	}
}

// TestFalconLiboqs_SignClearsSecretArg checks that the backend wipes the
// private-key slice it is given once it has copied it into C memory.
func TestFalconLiboqs_SignClearsSecretArg(t *testing.T) {
	sk := mustHex(t, fnPairPrivKeyHex)
	_, err := falcon.sign(sk, mustHex(t, fnFixtureMessageHex))
	require.NoError(t, err)
	assert.Equal(t, make([]byte, FNDSA512PrivateKeySize), sk)
}

// TestFalconLiboqs_SignRetry drives the resampling loop with scripted
// attempts: only a 0x39-headed signature of 617..667 bytes is accepted, and
// after 16 non-conforming attempts sign fails.
func TestFalconLiboqs_SignRetry(t *testing.T) {
	script := func(sigs ...[]byte) (func() ([]byte, error), *int) {
		calls := 0
		return func() ([]byte, error) {
			s := sigs[min(calls, len(sigs)-1)]
			calls++
			return bytes.Clone(s), nil
		}, &calls
	}
	tests := []struct {
		name      string
		sigs      [][]byte
		wantLen   int // 0 means an error is expected
		wantCalls int
	}{
		{"first 650", [][]byte{fnSig(FNDSA512SignatureHeader, 650)}, 650, 1},
		{"min 617", [][]byte{fnSig(FNDSA512SignatureHeader, 617)}, 617, 1},
		{"max 667", [][]byte{fnSig(FNDSA512SignatureHeader, 667)}, 667, 1},
		{"668 then 650", [][]byte{fnSig(FNDSA512SignatureHeader, 668), fnSig(FNDSA512SignatureHeader, 650)}, 650, 2},
		{"616 then 667", [][]byte{fnSig(FNDSA512SignatureHeader, 616), fnSig(FNDSA512SignatureHeader, 667)}, 667, 2},
		{"bad header then ok", [][]byte{fnSig(0x29, 650), fnSig(FNDSA512SignatureHeader, 650)}, 650, 2},
		{"always 668", [][]byte{fnSig(FNDSA512SignatureHeader, 668)}, 0, fnMaxSignAttempts},
		{"always 752", [][]byte{fnSig(FNDSA512SignatureHeader, 752)}, 0, fnMaxSignAttempts},
		{"always 616", [][]byte{fnSig(FNDSA512SignatureHeader, 616)}, 0, fnMaxSignAttempts},
		{"always empty", [][]byte{nil}, 0, fnMaxSignAttempts},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			try, calls := script(tt.sigs...)
			got, err := signUntilWellFormed(try)
			assert.Equal(t, tt.wantCalls, *calls)
			if tt.wantLen == 0 {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "617..667")
				assert.Contains(t, err.Error(), "16 attempts")
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Len(t, got, tt.wantLen)
		})
	}

	t.Run("attempt error stops", func(t *testing.T) {
		boom := errors.New("boom")
		calls := 0
		got, err := signUntilWellFormed(func() ([]byte, error) { calls++; return nil, boom })
		require.ErrorIs(t, err, boom)
		assert.Nil(t, got)
		assert.Equal(t, 1, calls)
	})
	assert.Equal(t, 16, fnMaxSignAttempts)
}
