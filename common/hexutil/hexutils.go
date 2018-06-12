package hexutil

import "encoding/hex"

var (
	EmptyString = &hexError{"empty hex string"}
)

type hexError struct {
	msg string
}

func (h *hexError) Error() string {
	return h.msg
}

// Encode encodes bytes as a hex string.
func Encode(bytes []byte) string {
	encode := make([]byte, len(bytes)*2)
	hex.Encode(encode, bytes)

	return string(encode)
}

// Decode hex string as bytes
func Decode(input string) ([]byte, error) {
	if len(input) == 0 {
		return nil, EmptyString
	}

	return hex.DecodeString(input[:])
}
