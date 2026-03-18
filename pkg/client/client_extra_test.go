package client_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestNewGrpcClient(t *testing.T) {
	c := client.NewGrpcClient("localhost:50051")
	assert.Equal(t, "localhost:50051", c.Address)
}

func TestNewGrpcClientWithTimeout(t *testing.T) {
	// Verify the constructor-supplied timeout governs request deadlines.
	mock := &mockWalletServer{
		GetNextMaintenanceTimeFunc: func(ctx context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, mock)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.GracefulStop(); _ = lis.Close() })

	c := client.NewGrpcClientWithTimeout("passthrough:///bufconn", 50*time.Millisecond)
	err := c.Start(
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { c.Stop() })

	start := time.Now()
	_, err = c.GetNextMaintenanceTime()
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeadlineExceeded")
	assert.Less(t, elapsed, 1*time.Second)
}

func TestSetTimeout_ObservedViaDeadlineExceeded(t *testing.T) {
	// Use a mock that blocks until the context deadline fires to prove
	// SetTimeout actually governs the request context deadline.
	// We use GetNextMaintenanceTime because it propagates errors
	// (unlike ListNodes which swallows them).
	mock := &mockWalletServer{
		GetNextMaintenanceTimeFunc: func(ctx context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	c := newMockClient(t, mock)
	c.SetTimeout(50 * time.Millisecond)

	start := time.Now()
	_, err := c.GetNextMaintenanceTime()
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeadlineExceeded")
	// Elapsed time should be close to the timeout, not the default 5s.
	assert.Less(t, elapsed, 1*time.Second)
}

func TestSetAPIKey_ObservedInMetadata(t *testing.T) {
	// Verify the API key is attached as outgoing gRPC metadata on requests.
	var capturedKey string
	mock := &mockWalletServer{
		ListNodesFunc: func(ctx context.Context, _ *api.EmptyMessage) (*api.NodeList, error) {
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				vals := md.Get("TRON-PRO-API-KEY")
				if len(vals) > 0 {
					capturedKey = vals[0]
				}
			}
			return &api.NodeList{}, nil
		},
	}

	c := newMockClient(t, mock)
	_ = c.SetAPIKey("my-secret-api-key")

	_, err := c.ListNodes()
	require.NoError(t, err)
	assert.Equal(t, "my-secret-api-key", capturedKey)
}

func TestStart_RPC(t *testing.T) {
	// Prove Start creates a usable client by issuing a real RPC
	// through an in-memory bufconn server.
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	mock := &mockWalletServer{
		GetNowBlock2Func: func(_ context.Context, _ *api.EmptyMessage) (*api.BlockExtention, error) {
			return &api.BlockExtention{
				BlockHeader: &core.BlockHeader{
					RawData: &core.BlockHeaderRaw{Number: 42},
				},
			}, nil
		},
	}
	api.RegisterWalletServer(srv, mock)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.GracefulStop(); _ = lis.Close() })

	c := client.NewGrpcClient("passthrough:///bufconn")
	err := c.Start(
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { c.Stop() })

	block, err := c.GetNowBlock()
	require.NoError(t, err)
	assert.Equal(t, int64(42), block.BlockHeader.RawData.Number)
}

func TestStart_DefaultAddress(t *testing.T) {
	// Verify that Start sets the default address when empty.
	c := client.NewGrpcClient("")
	assert.Equal(t, "", c.Address)

	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	api.RegisterWalletServer(srv, &mockWalletServer{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(func() { srv.GracefulStop(); _ = lis.Close() })

	err := c.Start(grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}))
	require.NoError(t, err)
	assert.Equal(t, "grpc.trongrid.io:50051", c.Address)
	c.Stop()
}

func TestStop_NilConn(t *testing.T) {
	c := client.NewGrpcClient("localhost:50051")
	c.Stop() // should not panic with nil Conn
}

func TestStopAndStartNewClient(t *testing.T) {
	// Verify we can stop one client and start a fresh one pointing
	// at a different server, proving the stop/start lifecycle works.
	lis1 := bufconn.Listen(bufSize)
	srv1 := grpc.NewServer()
	mock1 := &mockWalletServer{
		GetNowBlock2Func: func(_ context.Context, _ *api.EmptyMessage) (*api.BlockExtention, error) {
			return &api.BlockExtention{
				BlockHeader: &core.BlockHeader{RawData: &core.BlockHeaderRaw{Number: 1}},
			}, nil
		},
	}
	api.RegisterWalletServer(srv1, mock1)
	go func() { _ = srv1.Serve(lis1) }()
	t.Cleanup(func() { srv1.GracefulStop(); _ = lis1.Close() })

	dialerFor := func(l *bufconn.Listener) grpc.DialOption {
		return grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return l.DialContext(ctx)
		})
	}

	c := client.NewGrpcClient("passthrough:///bufconn1")
	err := c.Start(dialerFor(lis1), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	block, err := c.GetNowBlock()
	require.NoError(t, err)
	assert.Equal(t, int64(1), block.BlockHeader.RawData.Number)

	// Set up a second server returning a different block number.
	lis2 := bufconn.Listen(bufSize)
	srv2 := grpc.NewServer()
	mock2 := &mockWalletServer{
		GetNowBlock2Func: func(_ context.Context, _ *api.EmptyMessage) (*api.BlockExtention, error) {
			return &api.BlockExtention{
				BlockHeader: &core.BlockHeader{RawData: &core.BlockHeaderRaw{Number: 2}},
			}, nil
		},
	}
	api.RegisterWalletServer(srv2, mock2)
	go func() { _ = srv2.Serve(lis2) }()
	t.Cleanup(func() { srv2.GracefulStop(); _ = lis2.Close() })

	// Reconnect re-calls Start with the saved opts, so we need to
	// update the dialer. Since Start stores opts, we need to inject
	// the new dialer. The simplest approach: stop, reconfigure, start.
	c.Stop()
	c2 := client.NewGrpcClient("passthrough:///bufconn2")
	err = c2.Start(dialerFor(lis2), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { c2.Stop() })

	block, err = c2.GetNowBlock()
	require.NoError(t, err)
	assert.Equal(t, int64(2), block.BlockHeader.RawData.Number)
}

func TestGetMessageBytes(t *testing.T) {
	msg := client.GetMessageBytes([]byte{0x01, 0x02})
	assert.Equal(t, []byte{0x01, 0x02}, msg.Value)
}

func TestGetMessageNumber(t *testing.T) {
	msg := client.GetMessageNumber(12345)
	assert.Equal(t, int64(12345), msg.Num)
}

func TestGetPaginatedMessage(t *testing.T) {
	msg := client.GetPaginatedMessage(10, 20)
	assert.Equal(t, int64(10), msg.Offset)
	assert.Equal(t, int64(20), msg.Limit)
}

func TestGRPCInsecure(t *testing.T) {
	opt := client.GRPCInsecure()
	assert.NotNil(t, opt)
}

func TestGetBlockByNum(t *testing.T) {
	mock := &mockWalletServer{
		GetBlockByNum2Func: func(_ context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
			assert.Equal(t, int64(999), in.Num)
			return &api.BlockExtention{
				BlockHeader: &core.BlockHeader{
					RawData: &core.BlockHeaderRaw{Number: 999},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	block, err := c.GetBlockByNum(999)
	require.NoError(t, err)
	assert.Equal(t, int64(999), block.BlockHeader.RawData.Number)
}

func TestSetContext_CancellationPropagated(t *testing.T) {
	mock := &mockWalletServer{
		GetNextMaintenanceTimeFunc: func(ctx context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	c := newMockClient(t, mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel
	require.NoError(t, c.SetContext(ctx))

	_, err := c.GetNextMaintenanceTime()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Canceled")
}

func TestSetContext_DeadlinePropagated(t *testing.T) {
	mock := &mockWalletServer{
		GetNextMaintenanceTimeFunc: func(ctx context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	c := newMockClient(t, mock)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	require.NoError(t, c.SetContext(ctx))

	start := time.Now()
	_, err := c.GetNextMaintenanceTime()
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeadlineExceeded")
	assert.Less(t, elapsed, 1*time.Second)
}

func TestSetContext_NilReturnsError(t *testing.T) {
	c := client.NewGrpcClient("localhost:50051")
	//nolint:staticcheck // SA1012: intentionally passing nil to test error guard
	err := c.SetContext(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil context")
}

// --- Ctx variant tests ---

func TestGetNowBlockCtx(t *testing.T) {
	mock := &mockWalletServer{
		GetNowBlock2Func: func(_ context.Context, _ *api.EmptyMessage) (*api.BlockExtention, error) {
			return &api.BlockExtention{
				BlockHeader: &core.BlockHeader{
					RawData: &core.BlockHeaderRaw{Number: 100},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	block, err := c.GetNowBlockCtx(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(100), block.BlockHeader.RawData.Number)
}

func TestGetBlockByNumCtx(t *testing.T) {
	mock := &mockWalletServer{
		GetBlockByNum2Func: func(_ context.Context, in *api.NumberMessage) (*api.BlockExtention, error) {
			return &api.BlockExtention{
				BlockHeader: &core.BlockHeader{
					RawData: &core.BlockHeaderRaw{Number: in.Num},
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	block, err := c.GetBlockByNumCtx(context.Background(), 555)
	require.NoError(t, err)
	assert.Equal(t, int64(555), block.BlockHeader.RawData.Number)
}

func TestGetNextMaintenanceTimeCtx_Cancellation(t *testing.T) {
	mock := &mockWalletServer{
		GetNextMaintenanceTimeFunc: func(ctx context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	c := newMockClient(t, mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	_, err := c.GetNextMaintenanceTimeCtx(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Canceled")
}

func TestGetNextMaintenanceTimeCtx_Deadline(t *testing.T) {
	mock := &mockWalletServer{
		GetNextMaintenanceTimeFunc: func(ctx context.Context, _ *api.EmptyMessage) (*api.NumberMessage, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	c := newMockClient(t, mock)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.GetNextMaintenanceTimeCtx(ctx)
	elapsed := time.Since(start)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DeadlineExceeded")
	assert.Less(t, elapsed, 1*time.Second)
}

func TestBroadcastCtx(t *testing.T) {
	mock := &mockWalletServer{
		BroadcastTransactionFunc: func(_ context.Context, _ *core.Transaction) (*api.Return, error) {
			return &api.Return{Result: true, Code: api.Return_SUCCESS}, nil
		},
	}

	c := newMockClient(t, mock)
	result, err := c.BroadcastCtx(context.Background(), &core.Transaction{})
	require.NoError(t, err)
	assert.True(t, result.GetResult())
}

func TestListNodesCtx_APIKeyPropagated(t *testing.T) {
	var capturedKey string
	mock := &mockWalletServer{
		ListNodesFunc: func(ctx context.Context, _ *api.EmptyMessage) (*api.NodeList, error) {
			md, ok := metadata.FromIncomingContext(ctx)
			if ok {
				vals := md.Get("TRON-PRO-API-KEY")
				if len(vals) > 0 {
					capturedKey = vals[0]
				}
			}
			return &api.NodeList{}, nil
		},
	}

	c := newMockClient(t, mock)
	_ = c.SetAPIKey("ctx-test-key")

	_, err := c.ListNodesCtx(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ctx-test-key", capturedKey)
}

func TestGetAccountResourceCtx(t *testing.T) {
	mock := &mockWalletServer{
		GetAccountResourceFunc: func(_ context.Context, _ *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{
				FreeNetLimit: 5000,
				EnergyLimit:  50000,
			}, nil
		},
	}

	c := newMockClient(t, mock)
	res, err := c.GetAccountResourceCtx(context.Background(), "TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b")
	require.NoError(t, err)
	assert.Equal(t, int64(5000), res.FreeNetLimit)
	assert.Equal(t, int64(50000), res.EnergyLimit)
}

func TestGetAccountResourceCtx_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetAccountResourceCtx(context.Background(), "invalid")
	require.Error(t, err)
}

func TestGetAccountResource(t *testing.T) {
	mock := &mockWalletServer{
		GetAccountResourceFunc: func(_ context.Context, _ *core.Account) (*api.AccountResourceMessage, error) {
			return &api.AccountResourceMessage{
				FreeNetLimit: 5000,
				EnergyLimit:  50000,
			}, nil
		},
	}

	c := newMockClient(t, mock)
	res, err := c.GetAccountResource("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b")
	require.NoError(t, err)
	assert.Equal(t, int64(5000), res.FreeNetLimit)
	assert.Equal(t, int64(50000), res.EnergyLimit)
}

func TestGetAccountResource_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.GetAccountResource("invalid")
	require.Error(t, err)
}
