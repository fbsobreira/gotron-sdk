package client

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

// ListNodes provides list of network nodes
func (g *GrpcClient) ListNodes() (*api.NodeList, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	nodeList, err := g.Client.ListNodes(ctx, new(api.EmptyMessage))
	if err != nil {
		zap.L().Error("List nodes", zap.Error(err))
	}
	return nodeList, nil
}

// GetNextMaintenanceTime get next epoch timestamp
func (g *GrpcClient) GetNextMaintenanceTime() (*api.NumberMessage, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.GetNextMaintenanceTime(ctx,
		new(api.EmptyMessage))
}

// TotalTransaction return total transciton in network
func (g *GrpcClient) TotalTransaction() (*api.NumberMessage, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.TotalTransaction(ctx,
		new(api.EmptyMessage))
}

// GetTransactionByID returns transaction details by ID
func (g *GrpcClient) GetTransactionByID(id string) (*core.Transaction, error) {
	transactionID := new(api.BytesMessage)
	var err error

	transactionID.Value, err = common.FromHex(id)
	if err != nil {
		return nil, fmt.Errorf("get transaction by id error: %v", err)
	}

	ctx, cancel := g.getContext()
	defer cancel()

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
	transactionID := new(api.BytesMessage)
	var err error

	transactionID.Value, err = common.FromHex(id)
	if err != nil {
		return nil, fmt.Errorf("get transaction by id error: %v", err)
	}

	ctx, cancel := g.getContext()
	defer cancel()

	txi, err := g.Client.GetTransactionInfoById(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(txi.Id, transactionID.Value) {
		return txi, nil
	}
	return nil, fmt.Errorf("transaction info not found")
}

// Broadcast broadcast TX
func (g *GrpcClient) Broadcast(tx *core.Transaction) (*api.Return, error) {
	ctx, cancel := g.getContext()
	defer cancel()
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

// GetNodeInfo current connection
func (g *GrpcClient) GetNodeInfo() (*core.NodeInfo, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.GetNodeInfo(ctx, new(api.EmptyMessage))
}

// GetEnergyPrices returns energy prices
func (g *GrpcClient) GetEnergyPrices() (*api.PricesResponseMessage, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.GetEnergyPrices(ctx, new(api.EmptyMessage))
}

// GetBandwidthPrices returns bandwidth prices
func (g *GrpcClient) GetBandwidthPrices() (*api.PricesResponseMessage, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.GetBandwidthPrices(ctx, new(api.EmptyMessage))
}

// GetMemoFee returns memo fee
func (g *GrpcClient) GetMemoFee() (*api.PricesResponseMessage, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	return g.Client.GetMemoFee(ctx, new(api.EmptyMessage))
}

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
	resp, err := g.GetEnergyPrices()
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
	resp, err := g.GetBandwidthPrices()
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
	resp, err := g.GetMemoFee()
	if err != nil {
		return nil, err
	}
	entries, err := ParsePrices(resp.GetPrices())
	if err != nil {
		return nil, fmt.Errorf("parse memo fee prices: %w", err)
	}
	return entries, nil
}
