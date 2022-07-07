package keys

import (
	secp256k1 "github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Dump struct {
	PrivateKey, PublicKeyCompressed, PublicKey string
}

func EncodeHex(sk *secp256k1.PrivateKey, pk *secp256k1.PublicKey) *Dump {
	p0 := sk.Serialize()
	p1 := pk.SerializeCompressed()
	p2 := pk.SerializeUncompressed()
	return &Dump{hexutil.Encode(p0), hexutil.Encode(p1), hexutil.Encode(p2)}
}
