package common_test

import (
	"bytes"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

func FuzzBase58EncodeCheck(f *testing.F) {
	// Seed with a valid 21-byte TRON address (0x41 prefix + 20 bytes).
	f.Add(append([]byte{0x41}, make([]byte, 20)...))
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0x41, 0x01, 0x02, 0x03})

	f.Fuzz(func(t *testing.T, data []byte) {
		encoded := common.EncodeCheck(data)
		if len(encoded) == 0 {
			t.Fatalf("EncodeCheck returned empty string for input of length %d", len(data))
		}

		// DecodeCheck enforces TRON-specific constraints (prefix, length,
		// checksum) so it only round-trips for valid 21-byte inputs with
		// the 0x41 prefix. For other inputs we just verify no panic.
		decoded, err := common.DecodeCheck(encoded)
		if err != nil {
			return
		}
		if !bytes.Equal(decoded, data) {
			t.Errorf("round-trip mismatch: got %x, want %x", decoded, data)
		}
	})
}

func FuzzBase58DecodeCheck(f *testing.F) {
	f.Add("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	f.Add("")
	f.Add("1")
	f.Add("!!!invalid!!!")
	f.Add("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")

	f.Fuzz(func(t *testing.T, s string) {
		// Just verify no panic.
		_, _ = common.DecodeCheck(s)
	})
}
