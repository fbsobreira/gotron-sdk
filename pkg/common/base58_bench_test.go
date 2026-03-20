package common_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

func BenchmarkEncodeCheck(b *testing.B) {
	// 21-byte TRON address (0x41 prefix + 20-byte payload)
	input := []byte{0x41, 0xb9, 0xf4, 0xa6, 0x9c, 0x5b, 0xae, 0x7c, 0xb8, 0x19,
		0x0e, 0x34, 0x5d, 0x5d, 0xe7, 0x34, 0x77, 0x99, 0x76, 0xa7, 0x9c}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = common.EncodeCheck(input)
	}
}

func BenchmarkDecodeCheck(b *testing.B) {
	const addr = "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = common.DecodeCheck(addr)
	}
}

func BenchmarkEncode(b *testing.B) {
	input := []byte{0x41, 0xb9, 0xf4, 0xa6, 0x9c, 0x5b, 0xae, 0x7c, 0xb8, 0x19,
		0x0e, 0x34, 0x5d, 0x5d, 0xe7, 0x34, 0x77, 0x99, 0x76, 0xa7, 0x9c,
		0xaa, 0xbb, 0xcc, 0xdd}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = common.Encode(input)
	}
}

func BenchmarkDecode(b *testing.B) {
	const addr = "TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = common.Decode(addr)
	}
}
