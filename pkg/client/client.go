package client

import (
	"context"
	"fmt"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// GrpcClient controller structure
type GrpcClient struct {
	Address     string
	Conn        *grpc.ClientConn
	Client      api.WalletClient
	grpcTimeout time.Duration
	opts        []grpc.DialOption
	apiKey      string
	baseCtx     context.Context
}

// NewGrpcClient create grpc controller
func NewGrpcClient(address string) *GrpcClient {
	client := &GrpcClient{
		Address:     address,
		grpcTimeout: 5 * time.Second,
	}
	return client
}

// NewGrpcClientWithTimeout create grpc controller
func NewGrpcClientWithTimeout(address string, timeout time.Duration) *GrpcClient {
	client := &GrpcClient{
		Address:     address,
		grpcTimeout: timeout,
	}
	return client
}

// SetTimeout for Client connections
func (g *GrpcClient) SetTimeout(timeout time.Duration) {
	g.grpcTimeout = timeout
}

// Start initiate grpc  connection
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

// SetAPIKey enable API on connection
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

// Stop GRPC Connection
func (g *GrpcClient) Stop() {
	if g.Conn != nil {
		_ = g.Conn.Close()
	}
}

// Reconnect GRPC
func (g *GrpcClient) Reconnect(url string) error {
	g.Stop()
	if len(url) > 0 {
		g.Address = url
	}
	return g.Start(g.opts...)
}

// GetMessageBytes return grpc message from bytes
func GetMessageBytes(m []byte) *api.BytesMessage {
	message := new(api.BytesMessage)
	message.Value = m
	return message
}

// GetMessageNumber return grpc message number
func GetMessageNumber(n int64) *api.NumberMessage {
	message := new(api.NumberMessage)
	message.Num = n
	return message
}

// GetPaginatedMessage return grpc message number
func GetPaginatedMessage(offset int64, limit int64) *api.PaginatedMessage {
	return &api.PaginatedMessage{
		Offset: offset,
		Limit:  limit,
	}
}
