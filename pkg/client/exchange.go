package client

import (
	"encoding/binary"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
)

// ExchangeList of bancor TRC10, use page -1 to list all
func (g *GrpcClient) ExchangeList(page int64, limit ...int) (*api.ExchangeList, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	if page == -1 {
		return g.Client.ListExchanges(ctx, new(api.EmptyMessage))
	}

	useLimit := int64(10)
	if len(limit) == 1 {
		useLimit = int64(limit[0])
	}
	return g.Client.GetPaginatedExchangeList(ctx, GetPaginatedMessage(page*useLimit, useLimit))
}

// ExchangeByID returns exchangeDetails
func (g *GrpcClient) ExchangeByID(id int64) (*core.Exchange, error) {
	ctx, cancel := g.getContext()
	defer cancel()
	bID := make([]byte, 8)
	binary.BigEndian.PutUint64(bID, uint64(id))

	result, err := g.Client.GetExchangeById(ctx, GetMessageBytes(bID))
	if err != nil {
		return nil, err
	}
	if result.ExchangeId != id {
		return nil, fmt.Errorf("Exchange does not exists")
	}
	return result, nil
}

// ExchangeCreate from two tokens (TRC10/TRX) only
func (g *GrpcClient) ExchangeCreate(
	from string,
	tokenID1 string,
	amountToken1 int64,
	tokenID2 string,
	amountToken2 int64,
) (*api.TransactionExtention, error) {
	var err error

	contract := &core.ExchangeCreateContract{
		FirstTokenId:       []byte(tokenID1),
		FirstTokenBalance:  amountToken1,
		SecondTokenId:      []byte(tokenID2),
		SecondTokenBalance: amountToken2,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.ExchangeCreate(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}

// ExchangeInject both tokens into banco pair (the second token is taken info transaction process)
func (g *GrpcClient) ExchangeInject(
	from string,
	exchangeID int64,
	tokenID string,
	amountToken int64,
) (*api.TransactionExtention, error) {
	var err error

	contract := &core.ExchangeInjectContract{
		ExchangeId: exchangeID,
		TokenId:    []byte(tokenID),
		Quant:      amountToken,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.ExchangeInject(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}

// ExchangeWithdraw both tokens into banco pair (the second token is taken info transaction process)
func (g *GrpcClient) ExchangeWithdraw(
	from string,
	exchangeID int64,
	tokenID string,
	amountToken int64,
) (*api.TransactionExtention, error) {
	var err error

	contract := &core.ExchangeWithdrawContract{
		ExchangeId: exchangeID,
		TokenId:    []byte(tokenID),
		Quant:      amountToken,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.ExchangeWithdraw(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}

// ExchangeTrade on bancor TRC10
func (g *GrpcClient) ExchangeTrade(
	from string,
	exchangeID int64,
	tokenID string,
	amountToken int64,
	amountExpected int64,
) (*api.TransactionExtention, error) {
	var err error

	contract := &core.ExchangeTransactionContract{
		ExchangeId: exchangeID,
		TokenId:    []byte(tokenID),
		Quant:      amountToken,
		Expected:   amountExpected,
	}
	if contract.OwnerAddress, err = common.DecodeCheck(from); err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	tx, err := g.Client.ExchangeTransaction(ctx, contract)
	if err != nil {
		return nil, err
	}
	if proto.Size(tx) == 0 {
		return nil, fmt.Errorf("bad transaction")
	}
	if tx.GetResult().GetCode() != 0 {
		return nil, fmt.Errorf("%s", tx.GetResult().GetMessage())
	}
	return tx, nil
}
