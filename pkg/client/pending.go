package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ErrPendingTxNotFound is returned when a transaction is not found in the pending pool.
var ErrPendingTxNotFound = errors.New("transaction not found in pending pool")

// GetTransactionFromPending returns a transaction from the pending pool by ID.
func (g *GrpcClient) GetTransactionFromPending(id string) (*core.Transaction, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetTransactionFromPendingCtx(ctx, id)
}

// GetTransactionFromPendingCtx is the context-aware version of GetTransactionFromPending.
func (g *GrpcClient) GetTransactionFromPendingCtx(ctx context.Context, id string) (*core.Transaction, error) {
	ctx = g.withAPIKey(ctx)
	msg := new(api.BytesMessage)
	var err error

	msg.Value, err = common.FromHex(id)
	if err != nil {
		return nil, fmt.Errorf("get transaction from pending error: %w", err)
	}

	tx, err := g.Client.GetTransactionFromPending(ctx, msg)
	if err != nil {
		return nil, err
	}
	if size := proto.Size(tx); size > 0 {
		return tx, nil
	}
	return nil, ErrPendingTxNotFound
}

// GetTransactionListFromPending returns the list of transaction IDs in the pending pool.
func (g *GrpcClient) GetTransactionListFromPending() (*api.TransactionIdList, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetTransactionListFromPendingCtx(ctx)
}

// GetTransactionListFromPendingCtx is the context-aware version of GetTransactionListFromPending.
func (g *GrpcClient) GetTransactionListFromPendingCtx(ctx context.Context) (*api.TransactionIdList, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.GetTransactionListFromPending(ctx, new(api.EmptyMessage))
}

// GetPendingSize returns the number of transactions in the pending pool.
func (g *GrpcClient) GetPendingSize() (*api.NumberMessage, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetPendingSizeCtx(ctx)
}

// GetPendingSizeCtx is the context-aware version of GetPendingSize.
func (g *GrpcClient) GetPendingSizeCtx(ctx context.Context) (*api.NumberMessage, error) {
	ctx = g.withAPIKey(ctx)
	return g.Client.GetPendingSize(ctx, new(api.EmptyMessage))
}

// IsTransactionPending checks whether a transaction is in the pending pool.
func (g *GrpcClient) IsTransactionPending(id string) (bool, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.IsTransactionPendingCtx(ctx, id)
}

// IsTransactionPendingCtx is the context-aware version of IsTransactionPending.
func (g *GrpcClient) IsTransactionPendingCtx(ctx context.Context, id string) (bool, error) {
	tx, err := g.GetTransactionFromPendingCtx(ctx, id)
	if err != nil {
		if errors.Is(err, ErrPendingTxNotFound) {
			return false, nil
		}
		return false, err
	}
	return tx != nil, nil
}

// GetPendingTransactionsByAddress returns pending transactions where the given
// address is the owner (sender) of the contract. It fetches all pending tx IDs
// and then retrieves each transaction to check the owner address.
func (g *GrpcClient) GetPendingTransactionsByAddress(address string) ([]*core.Transaction, error) {
	ctx, cancel := g.newContext()
	defer cancel()
	return g.GetPendingTransactionsByAddressCtx(ctx, address)
}

// GetPendingTransactionsByAddressCtx is the context-aware version of GetPendingTransactionsByAddress.
func (g *GrpcClient) GetPendingTransactionsByAddressCtx(ctx context.Context, address string) ([]*core.Transaction, error) {
	addrBytes, err := common.DecodeCheck(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %v", err)
	}

	list, err := g.GetTransactionListFromPendingCtx(ctx)
	if err != nil {
		return nil, err
	}

	var result []*core.Transaction
	for _, txID := range list.GetTxId() {
		tx, err := g.GetTransactionFromPendingCtx(ctx, txID)
		if err != nil {
			if errors.Is(err, ErrPendingTxNotFound) {
				continue // tx may have been confirmed between list and fetch
			}
			return nil, err
		}
		if ownerAddr := extractOwnerAddress(tx); ownerAddr != nil && bytes.Equal(ownerAddr, addrBytes) {
			result = append(result, tx)
		}
	}
	return result, nil
}

// extractOwnerAddress extracts the owner_address field from the first contract
// in a transaction using proto reflection, avoiding a type switch over all
// contract types.
func extractOwnerAddress(tx *core.Transaction) []byte {
	contracts := tx.GetRawData().GetContract()
	if len(contracts) == 0 {
		return nil
	}

	param := contracts[0].GetParameter()
	if param == nil {
		return nil
	}

	msg, err := param.UnmarshalNew()
	if err != nil {
		return nil
	}

	fd := msg.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name("owner_address"))
	if fd == nil || fd.Kind() != protoreflect.BytesKind {
		return nil
	}
	return msg.ProtoReflect().Get(fd).Bytes()
}
