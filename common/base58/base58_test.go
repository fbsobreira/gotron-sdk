package base58

import (
	"github.com/sasaxie/go-client-api/common/hexutil"
	"strings"
	"testing"
)

func TestEncode(t *testing.T) {
}

func TestDecodeCheck(t *testing.T) {
	decodeBytes := DecodeCheck("27ZESitosJfKouTBrGg6Nk5yEjnJHXMbkZp")

	decode := hexutil.Encode(decodeBytes)

	if strings.EqualFold(decode, "a06f61d05912402335744c288d4b72a735ede35604") {
		t.Log("success")
	} else {
		t.Fatalf("failure: %s", decode)
	}
}
