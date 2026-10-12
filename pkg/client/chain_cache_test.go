package client_test

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

// countingChainParamsMock answers GetChainParameters with whatever next
// returns and counts the RPCs that reach the server.
func countingChainParamsMock(calls *atomic.Int32, next func(n int32) (*core.ChainParameters, error)) *mockWalletServer {
	return &mockWalletServer{
		GetChainParametersFunc: func(_ context.Context, _ *api.EmptyMessage) (*core.ChainParameters, error) {
			return next(calls.Add(1))
		},
	}
}

func TestGetPQCapabilities_CachesWithinTTL(t *testing.T) {
	var calls atomic.Int32
	c := newMockClient(t, countingChainParamsMock(&calls, func(int32) (*core.ChainParameters, error) {
		return chainParams("getAllowMlDsa44", int64(1)), nil
	}))

	first, err := c.GetPQCapabilities()
	require.NoError(t, err)
	second, err := c.GetPQCapabilitiesCtx(context.Background())
	require.NoError(t, err)
	allowed, err := c.PQSchemeAllowed(core.PQScheme_ML_DSA_44)
	require.NoError(t, err)

	assert.Equal(t, client.PQCapabilities{MLDSA44: true}, first)
	assert.Equal(t, first, second)
	assert.True(t, allowed)
	assert.Equal(t, int32(1), calls.Load(), "calls within the default TTL must reuse the first probe")
}

// TestGetPQCapabilities_CancelledContextOnCacheHit checks that a cached
// answer does not hide a cancelled context from the caller.
func TestGetPQCapabilities_CancelledContextOnCacheHit(t *testing.T) {
	var calls atomic.Int32
	c := newMockClient(t, countingChainParamsMock(&calls, func(int32) (*core.ChainParameters, error) {
		return chainParams("getAllowMlDsa44", int64(1)), nil
	}))
	_, err := c.GetPQCapabilities()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = c.GetPQCapabilitiesCtx(ctx)
	require.ErrorIs(t, err, context.Canceled)
	_, err = c.PQSchemeAllowedCtx(ctx, core.PQScheme_ML_DSA_44)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(1), calls.Load())
}

// TestGetPQCapabilities_NilContext checks that a nil ctx is treated as
// context.Background, as the other client RPCs do, on a hit and on a miss.
func TestGetPQCapabilities_NilContext(t *testing.T) {
	var calls atomic.Int32
	c := newMockClient(t, countingChainParamsMock(&calls, func(int32) (*core.ChainParameters, error) {
		return chainParams("getAllowMlDsa44", int64(1)), nil
	}))

	require.NotPanics(t, func() {
		//nolint:staticcheck // SA1012: a nil ctx is the case under test
		caps, err := c.GetPQCapabilitiesCtx(nil)
		require.NoError(t, err)
		assert.True(t, caps.MLDSA44)
		//nolint:staticcheck // SA1012: a nil ctx is the case under test
		allowed, err := c.PQSchemeAllowedCtx(nil, core.PQScheme_ML_DSA_44)
		require.NoError(t, err)
		assert.True(t, allowed)
	})
	assert.Equal(t, int32(1), calls.Load())
}

func TestGetPQCapabilities_ErrorNotCached(t *testing.T) {
	var calls atomic.Int32
	c := newMockClient(t, countingChainParamsMock(&calls, func(n int32) (*core.ChainParameters, error) {
		if n == 1 {
			return nil, status.Error(codes.Unavailable, "node down")
		}
		return chainParams("getAllowFnDsa512", int64(1)), nil
	}))

	_, err := c.PQSchemeAllowed(core.PQScheme_FN_DSA_512)
	require.Error(t, err)

	allowed, err := c.PQSchemeAllowed(core.PQScheme_FN_DSA_512)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int32(2), calls.Load(), "a failed probe must not be cached")

	// The successful probe is cached.
	_, err = c.GetPQCapabilities()
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

func TestInvalidatePQCapabilities_ForcesRefetch(t *testing.T) {
	var calls atomic.Int32
	c := newMockClient(t, countingChainParamsMock(&calls, func(n int32) (*core.ChainParameters, error) {
		// The node enables ML-DSA-44 between the first and second probe.
		return chainParams("getAllowMlDsa44", int64(n-1)), nil
	}))

	allowed, err := c.PQSchemeAllowed(core.PQScheme_ML_DSA_44)
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = c.PQSchemeAllowed(core.PQScheme_ML_DSA_44)
	require.NoError(t, err)
	assert.False(t, allowed, "cached answer expected before invalidation")
	assert.Equal(t, int32(1), calls.Load())

	c.InvalidatePQCapabilities()

	allowed, err = c.PQSchemeAllowed(core.PQScheme_ML_DSA_44)
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, int32(2), calls.Load())
}

func TestInvalidatePQCapabilities_DuringInflightProbe(t *testing.T) {
	var calls atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	c := newMockClient(t, countingChainParamsMock(&calls, func(n int32) (*core.ChainParameters, error) {
		if n == 1 {
			close(entered)
			<-release
			return chainParams("getAllowMlDsa44", int64(0)), nil
		}
		return chainParams("getAllowMlDsa44", int64(1)), nil
	}))

	done := make(chan error, 1)
	go func() {
		_, err := c.GetPQCapabilities()
		done <- err
	}()
	<-entered
	c.InvalidatePQCapabilities()
	close(release)
	require.NoError(t, <-done)

	allowed, err := c.PQSchemeAllowed(core.PQScheme_ML_DSA_44)
	require.NoError(t, err)
	assert.True(t, allowed, "a probe in flight during invalidation must not repopulate the cache")
	assert.Equal(t, int32(2), calls.Load())
}

func TestSetPQCapabilitiesTTL(t *testing.T) {
	tests := []struct {
		name      string
		ttl       time.Duration
		wantCalls int32
	}{
		{"zero disables caching", 0, 3},
		{"negative disables caching", -time.Second, 3},
		{"positive caches", time.Hour, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			c := newMockClient(t, countingChainParamsMock(&calls, func(int32) (*core.ChainParameters, error) {
				return chainParams("getAllowMlDsa44", int64(1)), nil
			}))
			c.SetPQCapabilitiesTTL(tt.ttl)
			for range 3 {
				allowed, err := c.PQSchemeAllowed(core.PQScheme_ML_DSA_44)
				require.NoError(t, err)
				assert.True(t, allowed)
			}
			assert.Equal(t, tt.wantCalls, calls.Load())
		})
	}
}

func TestSetPQCapabilitiesTTL_DropsCachedResult(t *testing.T) {
	var calls atomic.Int32
	c := newMockClient(t, countingChainParamsMock(&calls, func(int32) (*core.ChainParameters, error) {
		return chainParams("getAllowMlDsa44", int64(1)), nil
	}))
	_, err := c.GetPQCapabilities()
	require.NoError(t, err)
	c.SetPQCapabilitiesTTL(client.DefaultPQCapabilitiesTTL)
	_, err = c.GetPQCapabilities()
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

func TestGetPQCapabilities_ExpiresAfterTTL(t *testing.T) {
	var calls atomic.Int32
	c := newMockClient(t, countingChainParamsMock(&calls, func(int32) (*core.ChainParameters, error) {
		return chainParams("getAllowMlDsa44", int64(1)), nil
	}))
	const ttl = 20 * time.Millisecond
	c.SetPQCapabilitiesTTL(ttl)

	_, err := c.GetPQCapabilities()
	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())

	// Sleeping well past the TTL makes expiry certain; no assertion here
	// depends on a call landing inside the TTL.
	time.Sleep(3 * ttl)

	_, err = c.GetPQCapabilities()
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load(), "an expired entry must be refetched")
}

func TestGetPQCapabilities_ConcurrentAccess(t *testing.T) {
	var calls atomic.Int32
	c := newMockClient(t, countingChainParamsMock(&calls, func(int32) (*core.ChainParameters, error) {
		return chainParams("getAllowFnDsa512", int64(1), "getAllowMlDsa44", int64(1)), nil
	}))

	const workers = 8
	const iterations = 25
	var wg sync.WaitGroup
	errs := make(chan error, workers*iterations)
	for w := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range iterations {
				switch (w + i) % 4 {
				case 0:
					c.InvalidatePQCapabilities()
				case 1:
					c.SetPQCapabilitiesTTL(client.DefaultPQCapabilitiesTTL)
				default:
					allowed, err := c.PQSchemeAllowed(core.PQScheme_ML_DSA_44)
					if err == nil && !allowed {
						err = assert.AnError
					}
					if err != nil {
						errs <- err
					}
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	assert.Positive(t, calls.Load())
}

func TestReconnect_InvalidatesPQCapabilities(t *testing.T) {
	var calls atomic.Int32
	mock := countingChainParamsMock(&calls, func(int32) (*core.ChainParameters, error) {
		return chainParams("getAllowMlDsa44", int64(1)), nil
	})
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, mock)
	go func() {
		_ = srv.Serve(lis)
	}()
	t.Cleanup(func() {
		srv.GracefulStop()
		_ = lis.Close()
	})

	opts := []grpc.DialOption{
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	c := client.NewGrpcClient("passthrough:///bufconn")
	require.NoError(t, c.Start(opts...))
	t.Cleanup(c.Stop)

	_, err := c.GetPQCapabilities()
	require.NoError(t, err)
	_, err = c.GetPQCapabilities()
	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())

	// Reconnect may reach a different node, so it must not reuse the probe.
	require.NoError(t, c.Reconnect(""))
	_, err = c.GetPQCapabilities()
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())

	// A stopped client must not answer from the cache: the probe now fails
	// on the closed connection instead of reporting the scheme allowed.
	c.Stop()
	_, err = c.GetPQCapabilities()
	require.Error(t, err)
	assert.Equal(t, int32(2), calls.Load())

	// So may Stop followed by Start, for example after changing Address.
	require.NoError(t, c.Start(opts...))
	_, err = c.GetPQCapabilities()
	require.NoError(t, err)
	assert.Equal(t, int32(3), calls.Load())
}
