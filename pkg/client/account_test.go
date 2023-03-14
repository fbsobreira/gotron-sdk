package client_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

var (
	conn                              *client.GrpcClient
	apiKey                            = "622ec85e-7406-431d-9caf-0a19501469a4"
	tronAddress                       = "grpc.nile.trongrid.io:50051"
	accountAddress                    = "TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b"
	accountAddressWitness             = "TGj1Ej1qRzL9feLTLhjwgxXF4Ct6GTWg2U"
	testnetNileAddressExample         = "TUoHaVjx7n5xz8LwPRDckgFrDWhMhuSuJM"
	testnetNileAddressDelegateExample = "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g"
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

func TestGetAccountDetailedV2(t *testing.T) {
	acc, err := conn.GetAccountDetailed(testnetNileAddressExample)
	data, err := json.Marshal(acc)
	println(string(data))

	require.Nil(t, err)
	require.NotNil(t, acc.Allowance)
	require.NotNil(t, acc.Rewards)

}

func TestFreezeV2(t *testing.T) {
	freezeTx, err := conn.FreezeBalanceV2(testnetNileAddressExample, core.ResourceCode_BANDWIDTH, 1000000)
	require.Nil(t, err)

	data, err := json.Marshal(freezeTx)
	println(string(data))

	require.Nil(t, err)
	require.NotNil(t, freezeTx.GetTxid())

}

func TestUnfreezeV2(t *testing.T) {
	unfreezeTx, err := conn.UnfreezeBalanceV2(testnetNileAddressExample, core.ResourceCode_BANDWIDTH, 1000000)
	require.Nil(t, err)

	data, err := json.Marshal(unfreezeTx)
	println(string(data))

	require.Nil(t, err)
	require.NotNil(t, unfreezeTx.GetTxid())

}

func TestDelegate(t *testing.T) {
	tx, err := conn.DelegateResource(testnetNileAddressExample, testnetNileAddressDelegateExample, core.ResourceCode_BANDWIDTH, 1000000, false)
	require.Nil(t, err)

	data, err := json.Marshal(tx)
	println(string(data))

	require.Nil(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestUndelegate(t *testing.T) {
	tx, err := conn.UnDelegateResource(testnetNileAddressExample, testnetNileAddressDelegateExample, core.ResourceCode_BANDWIDTH, 1000000, false)
	require.Nil(t, err)

	data, err := json.Marshal(tx)
	println(string(data))

	require.Nil(t, err)
	require.NotNil(t, tx.GetTxid())
}
