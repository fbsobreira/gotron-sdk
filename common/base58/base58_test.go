package base58

import (
	"strings"
	"testing"

	"github.com/fbsobreira/gotron/common/hexutil"
)

func TestEncode(t *testing.T) {
}

func TestDecodeCheck(t *testing.T) {
	decodeBytes, _ := DecodeCheck("27ZESitosJfKouTBrGg6Nk5yEjnJHXMbkZp")

	decode := hexutil.Encode(decodeBytes)

	if strings.EqualFold(decode, "a06f61d05912402335744c288d4b72a735ede35604") {
		t.Log("success")
	} else {
		t.Fatalf("failure: %s", decode)
	}
}
