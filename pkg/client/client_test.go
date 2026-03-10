package client_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestTRC20(t *testing.T) {
	usdtAddr, _ := address.Base58ToAddress("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")

	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			var result []byte
			if bytes.Equal(in.ContractAddress, usdtAddr) {
				// 6 decimals
				result, _ = hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000006")
			} else {
				// 0 decimals
				result, _ = hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000000")
			}

			return &api.TransactionExtention{
				Result: &api.Return{
					Result: true,
					Code:   api.Return_SUCCESS,
				},
				ConstantResult: [][]byte{result},
			}, nil
		},
	}

	c := newMockClient(t, mock)

	value, err := c.TRC20GetDecimals("TN7EWmuVWrdehLwKGnU2rk42GWodbAXGUM")
	require.NoError(t, err)
	require.Equal(t, int64(0), value.Int64())

	value, err = c.TRC20GetDecimals("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	require.Equal(t, int64(6), value.Int64())
}

func TestSend(t *testing.T) {
	t.Skip()
	fromAddress := ""
	toAddress := ""

	privateKeyBytes, _ := hex.DecodeString("ABCD")

	c := client.NewGrpcClient("")
	err := c.Start(client.GRPCInsecure())
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

	// Convert btcec key to go-ethereum's curve to ensure compatibility
	// with non-CGO builds (e.g., Windows)
	ecdsaKey, err := crypto.ToECDSA(crypto.FromECDSA(sk.ToECDSA()))
	require.Nil(t, err)

	signature, err := crypto.Sign(hash, ecdsaKey)
	require.Nil(t, err)
	tx.Transaction.Signature = append(tx.Transaction.Signature, signature)

	result, err := c.Broadcast(tx.Transaction)
	require.Nil(t, err)
	require.NotNil(t, result)
}
