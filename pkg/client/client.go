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
	g.Conn, err = grpc.Dial(g.Address, opts...)

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

func (g *GrpcClient) getContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), g.grpcTimeout)
	if len(g.apiKey) > 0 {
		ctx = metadata.AppendToOutgoingContext(ctx, "TRON-PRO-API-KEY", g.apiKey)
	}
	return ctx, cancel
}

// Stop GRPC Connection
func (g *GrpcClient) Stop() {
	if g.Conn != nil {
		g.Conn.Close()
	}
}

// Reconnect GRPC
func (g *GrpcClient) Reconnect(url string) error {
	g.Stop()
	if len(url) > 0 {
		g.Address = url
	}
	g.Start(g.opts...)
	return nil
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
