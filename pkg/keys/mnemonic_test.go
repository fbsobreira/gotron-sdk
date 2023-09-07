package keys_test

import (
	"encoding/hex"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/keys"
	"github.com/stretchr/testify/assert"
)

func Test_mnemonic_to_pk(t *testing.T) {
	// Hardcoded index of 0 for brandnew account.
	private, _ := keys.FromMnemonicSeedAndPassphrase("abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about", "", 0)
	pk_bytes := private.Serialize()

	println("Privatekey: ", hex.EncodeToString(pk_bytes))
	assert.Equal(t, hex.EncodeToString(pk_bytes), "b5a4cea271ff424d7c31dc12a3e43e401df7a40d7412a15750f3f0b6b5449a28")
}
