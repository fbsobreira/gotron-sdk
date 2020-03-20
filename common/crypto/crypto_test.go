package crypto

import (
	"fmt"
	"github.com/fbsobreira/gotron/common/hexutil"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	k, err := GenerateKey()

	if err != nil {
		t.Error(err.Error())
	}

	priv := k.D.Text(16)
	fmt.Println(priv)

	k2, err := GetPrivateKeyByHexString(priv)
	if err != nil {
		t.Error(err.Error())
	}
	priv2 := k2.D.Text(16)
	fmt.Println(priv2)
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
