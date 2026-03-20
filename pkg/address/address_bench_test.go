package address

import (
	"math/big"
	"testing"
)

func BenchmarkAddressString(b *testing.B) {
	addr, err := Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = addr.String()
	}
}

func BenchmarkBase58ToAddress(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	}
}

func BenchmarkHexToAddress(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = HexToAddress("41b9f4a69c5bae7cb8190e345d5de734779976a79c")
	}
}

func BenchmarkBigToAddress(b *testing.B) {
	bi := big.NewInt(12345678)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = BigToAddress(bi)
	}
}

func BenchmarkBytesToAddress(b *testing.B) {
	raw := []byte{0x41, 0xb9, 0xf4, 0xa6, 0x9c, 0x5b, 0xae, 0x7c, 0xb8, 0x19,
		0x0e, 0x34, 0x5d, 0x5d, 0xe7, 0x34, 0x77, 0x99, 0x76, 0xa7, 0x9c}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BytesToAddress(raw)
	}
}

func BenchmarkAddressHex(b *testing.B) {
	addr, err := Base58ToAddress("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = addr.Hex()
	}
}
