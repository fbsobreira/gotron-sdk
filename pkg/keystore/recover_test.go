package keystore

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Recover(t *testing.T) {
	owner := "TA49kZ26Ky44NXEA6zFJ46XjzmZXGtvEdw"
	hash, _ := hex.DecodeString("bb5ca6345a3d199e66a2c63fec8abdc3fad602d57aefdc4621fd5f5b33658491")
	signature, _ := hex.DecodeString("c1d046315acb9334bce1183f5f40b7e356e07451191f82c7ab182898d4e81752359304ad42dbb737ab7f7b9540ec6170052f274dcb49d3e42f8d8c1e05727b4500")
	pubKey, err := RecoverPubkey(hash, signature)
	assert.Nil(t, err)
	assert.Equal(t, owner, pubKey.String())
}
