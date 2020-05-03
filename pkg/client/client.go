package client

import (
	"fmt"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"google.golang.org/grpc"
)

const grpcTimeout = 5 * time.Second

// GrpcClient controller structure
type GrpcClient struct {
	Address string
	Conn    *grpc.ClientConn
	Client  api.WalletClient
}

// NewGrpcClient create grpc controller
func NewGrpcClient(address string) *GrpcClient {
	client := new(GrpcClient)
	client.Address = address
	return client
}

// Start initiate grpc  connection
func (g *GrpcClient) Start() error {
	var err error
	g.Conn, err = grpc.Dial(g.Address, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("Connecting GRPC Client: %v", err)
	}
	g.Client = api.NewWalletClient(g.Conn)
	return nil
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
	g.Start()
	return nil
}
