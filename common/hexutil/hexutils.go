package hexutil

import "encoding/hex"

// Encode encodes bytes as a hex string.
func Encode(bytes []byte) string {
	encode := make([]byte, len(bytes) * 2)
	hex.Encode(encode, bytes)

	return string(encode)
}