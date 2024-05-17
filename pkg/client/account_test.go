package client_test

import (
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

	require.Nil(t, err)
	require.NotNil(t, acc.Allowance)
	require.NotNil(t, acc.Rewards)

	require.NotNil(t, acc.MaxCanDelegateBandwidth)
	require.NotNil(t, acc.MaxCanDelegateEnergy)

}

func TestFreezeV2(t *testing.T) {
	t.Skip() // Only in testnet nile
	freezeTx, err := conn.FreezeBalanceV2(testnetNileAddressExample, core.ResourceCode_BANDWIDTH, 1000000)

	require.Nil(t, err)
	require.NotNil(t, freezeTx.GetTxid())

}

func TestUnfreezeV2(t *testing.T) {
	t.Skip() // Only in testnet nile
	unfreezeTx, err := conn.UnfreezeBalanceV2(testnetNileAddressExample, core.ResourceCode_BANDWIDTH, 1000000)

	require.Nil(t, err)
	require.NotNil(t, unfreezeTx.GetTxid())

}

func TestDelegate(t *testing.T) {
	t.Skip() // Only in testnet nile
	tx, err := conn.DelegateResource(testnetNileAddressExample, testnetNileAddressDelegateExample, core.ResourceCode_BANDWIDTH, 1000000, false, 10000)

	require.Nil(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestUndelegate(t *testing.T) {
	t.Skip() // Only in testnet nile
	tx, err := conn.UnDelegateResource(testnetNileAddressExample, testnetNileAddressDelegateExample, core.ResourceCode_BANDWIDTH, 1000000, false)

	require.Nil(t, err)
	require.NotNil(t, tx.GetTxid())
}

func TestDelegateMaxSize(t *testing.T) {
	t.Skip() // Only in testnet nile
	tx, err := conn.GetCanDelegatedMaxSize(testnetNileAddressExample, int32(core.ResourceCode_BANDWIDTH.Number()))

	require.Nil(t, err)
	require.GreaterOrEqual(t, tx.GetMaxSize(), int64(0))
}

func TestUnfreezeLeftCount(t *testing.T) {
	t.Skip() // Only in testnet nile
	tx, err := conn.GetAvailableUnfreezeCount(testnetNileAddressExample)

	require.Nil(t, err)
	require.GreaterOrEqual(t, tx.GetCount(), int64(0))
}
