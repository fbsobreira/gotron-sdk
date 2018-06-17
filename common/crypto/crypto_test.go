package crypto

import (
	"github.com/sasaxie/go-client-api/common/hexutil"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	_, err := GenerateKey()

	if err != nil {
		t.Error(err.Error())
	}
}

func TestGetPrivateKeyByHexString(t *testing.T) {
	_, err := GetPrivateKeyByHexString("f5b1c865e615d584eb2a9234b95cd749cfbf0dc69d6e75584052812ba5b71418")

	if err != nil {
		t.Error(err.Error())
	}
}

func TestPubkeyToAddress(t *testing.T) {
	privateKey, err := GetPrivateKeyByHexString(
		"f5b1c865e615d584eb2a9234b95cd749cfbf0dc69d6e75584052812ba5b71418")

	if err != nil {
		t.Error(err.Error())
	}

	address := PubkeyToAddress(privateKey.PublicKey)

	t.Log(hexutil.Encode(address.Bytes()))
}
