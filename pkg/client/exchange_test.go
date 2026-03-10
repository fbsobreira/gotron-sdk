package client_test

import (
	"context"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExchangeList_All(t *testing.T) {
	expected := &api.ExchangeList{
		Exchanges: []*core.Exchange{
			{ExchangeId: 1},
			{ExchangeId: 2},
		},
	}
	mock := &mockWalletServer{
		ListExchangesFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.ExchangeList, error) {
			return expected, nil
		},
	}
	c := newMockClient(t, mock)

	result, err := c.ExchangeList(-1)
	require.NoError(t, err)
	assert.Len(t, result.Exchanges, 2)
	assert.Equal(t, int64(1), result.Exchanges[0].ExchangeId)
	assert.Equal(t, int64(2), result.Exchanges[1].ExchangeId)
}

func TestExchangeList_Paginated(t *testing.T) {
	var capturedOffset, capturedLimit int64
	expected := &api.ExchangeList{
		Exchanges: []*core.Exchange{
			{ExchangeId: 10},
		},
	}
	mock := &mockWalletServer{
		GetPaginatedExchangeListFunc: func(_ context.Context, in *api.PaginatedMessage) (*api.ExchangeList, error) {
			capturedOffset = in.Offset
			capturedLimit = in.Limit
			return expected, nil
		},
	}
	c := newMockClient(t, mock)

	result, err := c.ExchangeList(2, 5)
	require.NoError(t, err)
	assert.Len(t, result.Exchanges, 1)
	assert.Equal(t, int64(10), result.Exchanges[0].ExchangeId)
	assert.Equal(t, int64(10), capturedOffset) // page(2) * limit(5)
	assert.Equal(t, int64(5), capturedLimit)
}

func TestExchangeByID(t *testing.T) {
	mock := &mockWalletServer{
		GetExchangeByIdFunc: func(_ context.Context, _ *api.BytesMessage) (*core.Exchange, error) {
			return &core.Exchange{
				ExchangeId:         42,
				FirstTokenId:       []byte("TRX"),
				SecondTokenId:      []byte("WIN"),
				FirstTokenBalance:  1000,
				SecondTokenBalance: 2000,
			}, nil
		},
	}
	c := newMockClient(t, mock)

	result, err := c.ExchangeByID(42)
	require.NoError(t, err)
	assert.Equal(t, int64(42), result.ExchangeId)
	assert.Equal(t, []byte("TRX"), result.FirstTokenId)
	assert.Equal(t, []byte("WIN"), result.SecondTokenId)
}

func TestExchangeByID_NotFound(t *testing.T) {
	mock := &mockWalletServer{
		GetExchangeByIdFunc: func(_ context.Context, _ *api.BytesMessage) (*core.Exchange, error) {
			return &core.Exchange{
				ExchangeId: 99,
			}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.ExchangeByID(42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Exchange does not exists")
}

func TestExchangeCreate(t *testing.T) {
	var captured *core.ExchangeCreateContract
	mock := &mockWalletServer{
		ExchangeCreateFunc: func(_ context.Context, in *core.ExchangeCreateContract) (*api.TransactionExtention, error) {
			captured = in
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	tx, err := c.ExchangeCreate(accountAddress, "TRX", 1000, "WIN", 2000)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, captured)
	assert.Equal(t, []byte("TRX"), captured.FirstTokenId)
	assert.Equal(t, int64(1000), captured.FirstTokenBalance)
	assert.Equal(t, []byte("WIN"), captured.SecondTokenId)
	assert.Equal(t, int64(2000), captured.SecondTokenBalance)
}

func TestExchangeCreate_InvalidAddress(t *testing.T) {
	mock := &mockWalletServer{}
	c := newMockClient(t, mock)

	_, err := c.ExchangeCreate("INVALID", "TRX", 1000, "WIN", 2000)
	require.Error(t, err)
}

func TestExchangeCreate_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		ExchangeCreateFunc: func(_ context.Context, _ *core.ExchangeCreateContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.ExchangeCreate(accountAddress, "TRX", 1000, "WIN", 2000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestExchangeInject(t *testing.T) {
	var captured *core.ExchangeInjectContract
	mock := &mockWalletServer{
		ExchangeInjectFunc: func(_ context.Context, in *core.ExchangeInjectContract) (*api.TransactionExtention, error) {
			captured = in
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	tx, err := c.ExchangeInject(accountAddress, 42, "TRX", 5000)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, captured)
	assert.Equal(t, int64(42), captured.ExchangeId)
	assert.Equal(t, []byte("TRX"), captured.TokenId)
	assert.Equal(t, int64(5000), captured.Quant)
}

func TestExchangeInject_InvalidAddress(t *testing.T) {
	mock := &mockWalletServer{}
	c := newMockClient(t, mock)

	_, err := c.ExchangeInject("INVALID", 42, "TRX", 5000)
	require.Error(t, err)
}

func TestExchangeWithdraw(t *testing.T) {
	var captured *core.ExchangeWithdrawContract
	mock := &mockWalletServer{
		ExchangeWithdrawFunc: func(_ context.Context, in *core.ExchangeWithdrawContract) (*api.TransactionExtention, error) {
			captured = in
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	tx, err := c.ExchangeWithdraw(accountAddress, 42, "WIN", 3000)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, captured)
	assert.Equal(t, int64(42), captured.ExchangeId)
	assert.Equal(t, []byte("WIN"), captured.TokenId)
	assert.Equal(t, int64(3000), captured.Quant)
}

func TestExchangeTrade(t *testing.T) {
	var captured *core.ExchangeTransactionContract
	mock := &mockWalletServer{
		ExchangeTransactionFunc: func(_ context.Context, in *core.ExchangeTransactionContract) (*api.TransactionExtention, error) {
			captured = in
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	tx, err := c.ExchangeTrade(accountAddress, 42, "TRX", 5000, 4500)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.NotNil(t, captured)
	assert.Equal(t, int64(42), captured.ExchangeId)
	assert.Equal(t, []byte("TRX"), captured.TokenId)
	assert.Equal(t, int64(5000), captured.Quant)
	assert.Equal(t, int64(4500), captured.Expected)
}

func TestExchangeTrade_InvalidAddress(t *testing.T) {
	mock := &mockWalletServer{}
	c := newMockClient(t, mock)

	_, err := c.ExchangeTrade("INVALID", 42, "TRX", 5000, 4500)
	require.Error(t, err)
}
