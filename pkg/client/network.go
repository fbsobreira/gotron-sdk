package client

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// ListNodes returns the list of nodes connected to the network.
func (g *GrpcClient) ListNodes() (*api.NodeList, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.ListNodesCtx(ctx)
}

// ListNodesCtx is the context-aware version of ListNodes.
func (g *GrpcClient) ListNodesCtx(ctx context.Context) (*api.NodeList, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.ListNodes(ctx, new(api.EmptyMessage))
}

// GetNextMaintenanceTime returns the timestamp of the next SR maintenance epoch.
func (g *GrpcClient) GetNextMaintenanceTime() (*api.NumberMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetNextMaintenanceTimeCtx(ctx)
}

// GetNextMaintenanceTimeCtx is the context-aware version of GetNextMaintenanceTime.
func (g *GrpcClient) GetNextMaintenanceTimeCtx(ctx context.Context) (*api.NumberMessage, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.GetNextMaintenanceTime(ctx,
		new(api.EmptyMessage))
}

// TotalTransaction returns the total number of transactions on the network.
func (g *GrpcClient) TotalTransaction() (*api.NumberMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.TotalTransactionCtx(ctx)
}

// TotalTransactionCtx is the context-aware version of TotalTransaction.
func (g *GrpcClient) TotalTransactionCtx(ctx context.Context) (*api.NumberMessage, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.TotalTransaction(ctx,
		new(api.EmptyMessage))
}

// GetTransactionByID returns transaction details by ID
func (g *GrpcClient) GetTransactionByID(id string) (*core.Transaction, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetTransactionByIDCtx(ctx, id)
}

// GetTransactionByIDCtx is the context-aware version of GetTransactionByID.
func (g *GrpcClient) GetTransactionByIDCtx(ctx context.Context, id string) (*core.Transaction, error) {
	ctx = g.withAPIKey(ctx)
	transactionID := new(api.BytesMessage)
	var err error

	transactionID.Value, err = common.FromHex(id)
	if err != nil {
		return nil, fmt.Errorf("get transaction by id error: %v", err)
	}

	tx, err := g.Client.GetTransactionById(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if size := proto.Size(tx); size > 0 {
		return tx, nil
	}
	return nil, fmt.Errorf("transaction info not found")
}

// GetTransactionInfoByID returns transaction receipt by ID
func (g *GrpcClient) GetTransactionInfoByID(id string) (*core.TransactionInfo, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetTransactionInfoByIDCtx(ctx, id)
}

// GetTransactionInfoByIDCtx is the context-aware version of GetTransactionInfoByID.
func (g *GrpcClient) GetTransactionInfoByIDCtx(ctx context.Context, id string) (*core.TransactionInfo, error) {
	ctx = g.withAPIKey(ctx)
	transactionID := new(api.BytesMessage)
	var err error

	transactionID.Value, err = common.FromHex(id)
	if err != nil {
		return nil, fmt.Errorf("get transaction by id error: %v", err)
	}

	txi, err := g.Client.GetTransactionInfoById(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(txi.Id, transactionID.Value) {
		return txi, nil
	}
	return nil, fmt.Errorf("transaction info not found")
}

// Broadcast submits a signed transaction to the network.
func (g *GrpcClient) Broadcast(tx *core.Transaction) (*api.Return, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.BroadcastCtx(ctx, tx)
}

// BroadcastCtx is the context-aware version of Broadcast.
func (g *GrpcClient) BroadcastCtx(ctx context.Context, tx *core.Transaction) (*api.Return, error) {
	ctx = g.withAPIKey(ctx)
	result, err := g.Client.BroadcastTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}
	if !result.GetResult() {
		return result, fmt.Errorf("result error: %s", result.GetMessage())
	}
	if result.GetCode() != api.Return_SUCCESS {
		return result, fmt.Errorf("result error(%s): %s", result.GetCode(), result.GetMessage())
	}
	return result, nil
}

// GetNodeInfo returns information about the connected TRON node.
func (g *GrpcClient) GetNodeInfo() (*core.NodeInfo, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetNodeInfoCtx(ctx)
}

// GetNodeInfoCtx is the context-aware version of GetNodeInfo.
func (g *GrpcClient) GetNodeInfoCtx(ctx context.Context) (*core.NodeInfo, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.GetNodeInfo(ctx, new(api.EmptyMessage))
}

// GetEnergyPrices returns energy prices
func (g *GrpcClient) GetEnergyPrices() (*api.PricesResponseMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetEnergyPricesCtx(ctx)
}

// GetEnergyPricesCtx is the context-aware version of GetEnergyPrices.
func (g *GrpcClient) GetEnergyPricesCtx(ctx context.Context) (*api.PricesResponseMessage, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.GetEnergyPrices(ctx, new(api.EmptyMessage))
}

// GetBandwidthPrices returns bandwidth prices
func (g *GrpcClient) GetBandwidthPrices() (*api.PricesResponseMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetBandwidthPricesCtx(ctx)
}

// GetBandwidthPricesCtx is the context-aware version of GetBandwidthPrices.
func (g *GrpcClient) GetBandwidthPricesCtx(ctx context.Context) (*api.PricesResponseMessage, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.GetBandwidthPrices(ctx, new(api.EmptyMessage))
}

// GetMemoFee returns memo fee
func (g *GrpcClient) GetMemoFee() (*api.PricesResponseMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetMemoFeeCtx(ctx)
}

// GetMemoFeeCtx is the context-aware version of GetMemoFee.
func (g *GrpcClient) GetMemoFeeCtx(ctx context.Context) (*api.PricesResponseMessage, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.GetMemoFee(ctx, new(api.EmptyMessage))
}

// GRPCInsecure returns a grpc.DialOption that disables transport security.
func GRPCInsecure() grpc.DialOption {
	return grpc.WithTransportCredentials(insecure.NewCredentials())
}

// PriceEntry represents a single timestamp:price pair from the TRON price history.
type PriceEntry struct {
	Timestamp int64
	Price     int64
}

// ParsePrices parses the comma-separated "timestamp:price" format returned
// by GetEnergyPrices, GetBandwidthPrices, and GetMemoFee into structured entries.
// It does not validate sign — callers should check for negative values if needed.
func ParsePrices(raw string) ([]PriceEntry, error) {
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	entries := make([]PriceEntry, 0, len(parts))

	for _, p := range parts {
		ts, price, ok := strings.Cut(p, ":")
		if !ok {
			return nil, fmt.Errorf("invalid price entry %q: missing colon separator", p)
		}

		timestamp, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp in %q: %w", p, err)
		}

		priceVal, err := strconv.ParseInt(price, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price in %q: %w", p, err)
		}

		entries = append(entries, PriceEntry{Timestamp: timestamp, Price: priceVal})
	}

	return entries, nil
}

// GetEnergyPriceHistory returns parsed energy price history.
func (g *GrpcClient) GetEnergyPriceHistory() ([]PriceEntry, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetEnergyPriceHistoryCtx(ctx)
}

// GetEnergyPriceHistoryCtx is the context-aware version of GetEnergyPriceHistory.
func (g *GrpcClient) GetEnergyPriceHistoryCtx(ctx context.Context) ([]PriceEntry, error) {
	resp, err := g.GetEnergyPricesCtx(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := ParsePrices(resp.GetPrices())
	if err != nil {
		return nil, fmt.Errorf("parse energy prices: %w", err)
	}
	return entries, nil
}

// GetBandwidthPriceHistory returns parsed bandwidth price history.
func (g *GrpcClient) GetBandwidthPriceHistory() ([]PriceEntry, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetBandwidthPriceHistoryCtx(ctx)
}

// GetBandwidthPriceHistoryCtx is the context-aware version of GetBandwidthPriceHistory.
func (g *GrpcClient) GetBandwidthPriceHistoryCtx(ctx context.Context) ([]PriceEntry, error) {
	resp, err := g.GetBandwidthPricesCtx(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := ParsePrices(resp.GetPrices())
	if err != nil {
		return nil, fmt.Errorf("parse bandwidth prices: %w", err)
	}
	return entries, nil
}

// GetMemoFeeHistory returns parsed memo fee history.
func (g *GrpcClient) GetMemoFeeHistory() ([]PriceEntry, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetMemoFeeHistoryCtx(ctx)
}

// GetMemoFeeHistoryCtx is the context-aware version of GetMemoFeeHistory.
func (g *GrpcClient) GetMemoFeeHistoryCtx(ctx context.Context) ([]PriceEntry, error) {
	resp, err := g.GetMemoFeeCtx(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := ParsePrices(resp.GetPrices())
	if err != nil {
		return nil, fmt.Errorf("parse memo fee prices: %w", err)
	}
	return entries, nil
}
