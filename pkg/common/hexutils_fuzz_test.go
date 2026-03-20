package common_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

func FuzzFromHex(f *testing.F) {
	f.Add("0x48656c6c6f")
	f.Add("48656c6c6f")
	f.Add("")
	f.Add("0x")
	f.Add("0xZZ")
	f.Add("G")
	f.Add("0x1")

	f.Fuzz(func(t *testing.T, s string) {
		// Just verify no panic.
		_, _ = common.FromHex(s)
	})
}

func FuzzHex2Bytes(f *testing.F) {
	f.Add("48656c6c6f")
	f.Add("")
	f.Add("ZZ")
	f.Add("0")
	f.Add("ff")
	f.Add("ABCDEF")

	f.Fuzz(func(t *testing.T, s string) {
		// Just verify no panic.
		_, _ = common.Hex2Bytes(s)
	})
}
