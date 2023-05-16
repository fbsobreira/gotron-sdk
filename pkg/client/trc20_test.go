package client_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestTRC20_Balance(t *testing.T) {
	trc20Contract := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // USDT
	address := "TMuA6YqfCeX8EhbfYEg5y7S4DqzSJireY9"

	conn := client.NewGrpcClient("grpc.trongrid.io:50051")
	err := conn.Start(grpc.WithInsecure())
	require.Nil(t, err)

	balance, err := conn.TRC20ContractBalance(address, trc20Contract)
	assert.Nil(t, err)
	assert.Greater(t, balance.Int64(), int64(0))
}
