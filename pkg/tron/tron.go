// Package tron provides convenience constructors that bind a GrpcClient to
// the builder packages (txbuilder, contract, trc20). This avoids boilerplate
// and gives developers a single entry point to the SDK.
//
//	conn := client.NewGrpcClient("grpc.trongrid.io:50051")
//	conn.Start(grpc.WithTransportCredentials(insecure.NewCredentials()))
//
//	t := tron.New(conn)
//	t.TxBuilder().Transfer(from, to, amount).Send(ctx, signer)
//	t.Contract(addr).Method("transfer").From(from).Params(json).Send(ctx, signer)
//	t.TRC20(addr).Transfer(from, to, amount).Send(ctx, signer)
package tron

import (
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/standards/trc20"
	"github.com/fbsobreira/gotron-sdk/pkg/txbuilder"
)

// Compile-time interface satisfaction checks.
var (
	_ txbuilder.Client = (*client.GrpcClient)(nil)
	_ contract.Client  = (*client.GrpcClient)(nil)
)

// SDK wraps a GrpcClient and provides builder constructors.
type SDK struct {
	conn *client.GrpcClient
}

// New creates an SDK instance bound to the given GrpcClient.
func New(conn *client.GrpcClient) *SDK {
	return &SDK{conn: conn}
}

// TxBuilder returns a transaction builder for native TRON operations.
// Options set shared defaults for all transactions (e.g. WithPermissionID).
func (s *SDK) TxBuilder(opts ...txbuilder.Option) *txbuilder.Builder {
	return txbuilder.New(s.conn, opts...)
}

// Contract returns a contract call builder for the given address.
func (s *SDK) Contract(contractAddress string) *contract.ContractCall {
	return contract.New(s.conn, contractAddress)
}

// TRC20 returns a typed TRC20 token handle for the given contract address.
func (s *SDK) TRC20(contractAddress string) *trc20.Token {
	return trc20.New(s.conn, contractAddress)
}

// Client returns the underlying GrpcClient.
func (s *SDK) Client() *client.GrpcClient {
	return s.conn
}
