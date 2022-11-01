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
	conn    *client.GrpcClient
	apiKey  = "622ec85e-7406-431d-9caf-0a19501469a4"
	address = "grpc.trongrid.io:50051"
)

func TestMain(m *testing.M) {
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithInsecure())

	conn = client.NewGrpcClient(address)

	if err := conn.Start(opts...); err != nil {
		_ = fmt.Errorf("Error connecting GRPC Client: %v", err)
	}

	conn.SetAPIKey(apiKey)

	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestTRXTRC20Rewards(t *testing.T) {
	acc, err := conn.GetAccount("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b")
	require.Nil(t, err)

	fmt.Println("=========== ", acc)

	acc2, _ := conn.GetAccount("TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U")
	fmt.Println("=========== ", acc2)
}
