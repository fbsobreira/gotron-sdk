package client_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kima-finance/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestGrpcClient_GetTransactionListFromPending(t *testing.T) {
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithInsecure())
	conn := client.NewGrpcClient(tronAddress)
	if err := conn.Start(opts...); err != nil {
		fmt.Println("Error starting grpc client")
	}
	err := conn.SetAPIKey(apiKey)
	if err != nil {
		fmt.Println("Error setting api key")
	}
	txIds, err := conn.GetTransactionListFromPending()
	if err != nil {
		fmt.Println("Error getting pending transaction list")
	}
	fmt.Println(strings.Join(txIds, "\n"))
}

func TestGrpcClient_GetTransactionFromPending(t *testing.T) {
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithInsecure())
	conn := client.NewGrpcClient(tronAddress)
	if err := conn.Start(opts...); err != nil {
		fmt.Println("Error starting grpc client")
	}
	err := conn.SetAPIKey(apiKey)
	if err != nil {
		fmt.Println("Error setting api key")
	}
	txIds, err := conn.GetTransactionListFromPending()
	if err != nil {
		fmt.Println("Error getting pending transaction list")
	}
	if len(txIds) == 0 {
		fmt.Println("No transactions found in pending list")
	} else {
		txId := txIds[0]
		tx, err := conn.GetTransactionFromPending(txId)
		require.Nil(t, err)
		if tx != nil {
			fmt.Println(tx.String())
		} else {
			fmt.Println("Transaction already confirmed")
		}
	}
}
