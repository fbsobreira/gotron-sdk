package client_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func TestTRC20(t *testing.T) {
	c := client.NewGrpcClient("")
	err := c.Start(grpc.WithInsecure())
	require.Nil(t, err)

	value, err := c.TRC20GetDecimals("TN7EWmuVWrdehLwKGnU2rk42GWodbAXGUM")
	require.Nil(t, err)
	require.Equal(t, value.Int64(), int64(0))

	value, err = c.TRC20GetDecimals("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.Nil(t, err)
	require.Equal(t, value.Int64(), int64(6))
}

func TestSend(t *testing.T) {
	t.Skip()
	fromAddress := ""
	toAddress := ""

	privateKeyBytes, _ := hex.DecodeString("ABCD")

	c := client.NewGrpcClient("")
	err := c.Start(grpc.WithInsecure())
	require.Nil(t, err)
	tx, err := c.Transfer(fromAddress, toAddress, 1000)
	require.Nil(t, err)

	rawData, err := proto.Marshal(tx.Transaction.GetRawData())
	require.Nil(t, err)
	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)

	// btcec.PrivKeyFromBytes only returns a secret key and public key
	sk, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

	signature, err := crypto.Sign(hash, sk.ToECDSA())
	require.Nil(t, err)
	tx.Transaction.Signature = append(tx.Transaction.Signature, signature)

	result, err := c.Broadcast(tx.Transaction)
	require.Nil(t, err)
	require.NotNil(t, result)
}
