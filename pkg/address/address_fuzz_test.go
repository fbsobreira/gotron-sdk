package address

import (
	"math/big"
	"testing"
)

func FuzzBigToAddress(f *testing.F) {
	f.Add([]byte{0})
	f.Add([]byte{0x41})
	f.Add(make([]byte, 21))
	f.Add(make([]byte, 22))

	f.Fuzz(func(t *testing.T, data []byte) {
		b := new(big.Int).SetBytes(data)
		addr, err := BigToAddress(b)
		if err != nil {
			return
		}
		if len(addr) != AddressLength {
			t.Errorf("expected address length %d, got %d", AddressLength, len(addr))
		}
	})
}

func FuzzHexToAddress(f *testing.F) {
	f.Add("41b9f4a69c5bae7cb8190e345d5de734779976a79c")
	f.Add("0x41b9f4a69c5bae7cb8190e345d5de734779976a79c")
	f.Add("")
	f.Add("ZZZZ")
	f.Add("0x1")

	f.Fuzz(func(t *testing.T, s string) {
		addr, err := HexToAddress(s)
		if err != nil {
			return
		}
		_ = addr.Hex()
	})
}

func FuzzBase58ToAddress(f *testing.F) {
	f.Add("TSvT6Bg3siokv3dbdtt9o4oM1CTXmymGn1")
	f.Add("TN3W4H6rK2ce4vX9YnFQHwKENnHjoxb3m9")
	f.Add("")
	f.Add("not_a_valid_address")

	f.Fuzz(func(t *testing.T, s string) {
		addr, err := Base58ToAddress(s)
		if err != nil {
			return
		}
		_ = addr.String()
	})
}

func FuzzBase64ToAddress(f *testing.F) {
	f.Add("QUFB")
	f.Add("")
	f.Add("!!!not-base64!!!")

	f.Fuzz(func(t *testing.T, s string) {
		addr, err := Base64ToAddress(s)
		if err != nil {
			return
		}
		_ = addr.Bytes()
	})
}

func FuzzBytesToAddress(f *testing.F) {
	f.Add([]byte{0x41})
	f.Add(make([]byte, 20))
	f.Add(make([]byte, 21))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		addr := BytesToAddress(data)
		_ = addr.Bytes()
	})
}
