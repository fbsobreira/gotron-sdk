package client_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

var (
	conn                  *client.GrpcClient
	apiKey                = "622ec85e-7406-431d-9caf-0a19501469a4"
	tronAddress           = "grpc.trongrid.io:50051"
	accountAddress        = "TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b"
	accountAddressWitness = "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"
)

func TestMain(m *testing.M) {
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithInsecure())

	conn = client.NewGrpcClient(tronAddress)

	if err := conn.Start(opts...); err != nil {
		_ = fmt.Errorf("Error connecting GRPC Client: %v", err)
	}

	conn.SetAPIKey(apiKey)

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestGetAccountDetailed(t *testing.T) {
	acc, err := conn.GetAccountDetailed(accountAddress)
	require.Nil(t, err)
	require.NotNil(t, acc.Allowance)
	require.NotNil(t, acc.Rewards)

	acc2, err := conn.GetAccountDetailed(accountAddressWitness)
	require.Nil(t, err)
	require.NotNil(t, acc2.Allowance)
	require.NotNil(t, acc2.Rewards)

}
