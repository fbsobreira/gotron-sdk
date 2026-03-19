//go:build integration

package client_test

import (
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// Nile testnet endpoint — fixtures below are Nile-specific.
	nileEndpoint       = "grpc.nile.trongrid.io:50051"
	integrationTimeout = 30 * time.Second

	// Well-known Nile testnet addresses used by the test fixtures.
	nileTestAddress        = "TUoHaVjx7n5xz8LwPRDckgFrDWhMhuSuJM"
	nileTestAddress2       = "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g"
	nileTestWitnessAddress = "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH"
	nileTestAccountAddress = "TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b"
	nileUSDTContract       = "TXLAQ63Xg1NAzckPwKHvzw7CSEmLMEqcdj"
)

// getEndpoint returns the Nile testnet gRPC endpoint.
func getEndpoint() string {
	return nileEndpoint
}

// newIntegrationClient creates a GrpcClient connected to the configured TRON node.
// It fails the test if the connection cannot be established — a green integration
// run must mean the node was actually reachable and the tests ran.
func newIntegrationClient(t *testing.T) *client.GrpcClient {
	t.Helper()

	endpoint := getEndpoint()
	c := client.NewGrpcClientWithTimeout(endpoint, integrationTimeout)
	err := c.Start(grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "cannot connect to TRON node at %s", endpoint)
	t.Cleanup(c.Stop)
	return c
}
