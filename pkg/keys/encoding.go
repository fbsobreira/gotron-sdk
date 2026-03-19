package keys

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

// Dump holds hex-encoded representations of a key pair.
type Dump struct {
	PrivateKey, PublicKeyCompressed, PublicKey string
}

// EncodeHex serializes a private/public key pair into hex-encoded strings.
func EncodeHex(sk *btcec.PrivateKey, pk *btcec.PublicKey) *Dump {
	p0 := sk.Serialize()
	defer common.ZeroBytes(p0)
	p1 := pk.SerializeCompressed()
	p2 := pk.SerializeUncompressed()
	return &Dump{hexutil.Encode(p0), hexutil.Encode(p1), hexutil.Encode(p2)}
}
