package client

import (
	"context"
	"fmt"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// GrpcClient provides access to the TRON network via gRPC.
type GrpcClient struct {
	Address     string
	Conn        *grpc.ClientConn
	Client      api.WalletClient
	grpcTimeout time.Duration
	opts        []grpc.DialOption
	apiKey      string
	baseCtx     context.Context
}

// NewGrpcClient creates a new GrpcClient with a default timeout of 5 seconds.
func NewGrpcClient(address string) *GrpcClient {
	client := &GrpcClient{
		Address:     address,
		grpcTimeout: 5 * time.Second,
	}
	return client
}

// NewGrpcClientWithTimeout creates a new GrpcClient with the specified timeout.
func NewGrpcClientWithTimeout(address string, timeout time.Duration) *GrpcClient {
	client := &GrpcClient{
		Address:     address,
		grpcTimeout: timeout,
	}
	return client
}

// SetTimeout updates the timeout used for all subsequent RPC calls.
func (g *GrpcClient) SetTimeout(timeout time.Duration) {
	g.grpcTimeout = timeout
}

// Start establishes the gRPC connection. If no address was provided, it
// defaults to grpc.trongrid.io:50051.
func (g *GrpcClient) Start(opts ...grpc.DialOption) error {
	var err error
	if len(g.Address) == 0 {
		g.Address = "grpc.trongrid.io:50051"
	}
	g.opts = opts
	g.Conn, err = grpc.NewClient(g.Address, opts...)
	if err != nil {
		return fmt.Errorf("Connecting GRPC Client: %v", err)
	}
	g.Client = api.NewWalletClient(g.Conn)
	return nil
}

// SetAPIKey configures a TRON-PRO-API-KEY that is sent as gRPC metadata with every request.
func (g *GrpcClient) SetAPIKey(apiKey string) error {
	g.apiKey = apiKey
	return nil
}

// SetContext sets a base context for all RPC calls.
// This allows callers to propagate cancellation, deadlines, and tracing metadata.
// If not set, context.Background() is used.
//
// Note: cancelling the base context will cause all subsequent RPCs on this
// client to fail immediately. SetContext must not be called concurrently
// with RPC methods.
func (g *GrpcClient) SetContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("gotron: nil context")
	}
	g.baseCtx = ctx
	return nil
}

// newContext creates a derived context with the client's configured timeout.
// It does not inject API key metadata — use withAPIKey for that.
func (g *GrpcClient) newContext() (context.Context, context.CancelFunc) {
	base := g.baseCtx
	if base == nil {
		base = context.Background()
	}
	return context.WithTimeout(base, g.grpcTimeout)
}

// withAPIKey injects the API key as gRPC outgoing metadata if configured.
// It uses Set (not Append) so that chained Ctx calls don't produce duplicate headers.
func (g *GrpcClient) withAPIKey(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(g.apiKey) == 0 {
		return ctx
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	md.Set("TRON-PRO-API-KEY", g.apiKey)
	return metadata.NewOutgoingContext(ctx, md)
}

// Stop closes the underlying gRPC connection.
func (g *GrpcClient) Stop() {
	if g.Conn != nil {
		_ = g.Conn.Close()
	}
}

// Reconnect closes the current connection and re-establishes it.
// If url is non-empty, the client address is updated before reconnecting.
func (g *GrpcClient) Reconnect(url string) error {
	g.Stop()
	if len(url) > 0 {
		g.Address = url
	}
	return g.Start(g.opts...)
}

// GetMessageBytes wraps raw bytes into an api.BytesMessage for gRPC calls.
func GetMessageBytes(m []byte) *api.BytesMessage {
	message := new(api.BytesMessage)
	message.Value = m
	return message
}

// GetMessageNumber wraps an int64 into an api.NumberMessage for gRPC calls.
func GetMessageNumber(n int64) *api.NumberMessage {
	message := new(api.NumberMessage)
	message.Num = n
	return message
}

// GetPaginatedMessage creates an api.PaginatedMessage with the given offset and limit.
func GetPaginatedMessage(offset int64, limit int64) *api.PaginatedMessage {
	return &api.PaginatedMessage{
		Offset: offset,
		Limit:  limit,
	}
}
