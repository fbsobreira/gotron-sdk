package client

import (
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"go.uber.org/zap"
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
		zap.L().Error("Connecting GRPC Client", zap.Error(err))
		return err
	}
	g.Client = api.NewWalletClient(g.Conn)
	return nil
}
