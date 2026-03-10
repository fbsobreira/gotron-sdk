package client_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// GetAssetIssueByAccount
// ---------------------------------------------------------------------------

func TestGetAssetIssueByAccount_Success(t *testing.T) {
	expectedAddr, _ := common.DecodeCheck(accountAddress)
	mock := &mockWalletServer{
		GetAssetIssueByAccountFunc: func(_ context.Context, in *core.Account) (*api.AssetIssueList, error) {
			assert.Equal(t, expectedAddr, in.Address)
			return &api.AssetIssueList{
				AssetIssue: []*core.AssetIssueContract{
					{Name: []byte("TestToken")},
				},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	result, err := c.GetAssetIssueByAccount(accountAddress)
	require.NoError(t, err)
	require.Len(t, result.AssetIssue, 1)
	assert.Equal(t, "TestToken", string(result.AssetIssue[0].Name))
}

func TestGetAssetIssueByAccount_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	_, err := c.GetAssetIssueByAccount("INVALID")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetAssetIssueByName
// ---------------------------------------------------------------------------

func TestGetAssetIssueByName_Success(t *testing.T) {
	mock := &mockWalletServer{
		GetAssetIssueByNameFunc: func(_ context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error) {
			assert.Equal(t, []byte("TRX"), in.Value)
			return &core.AssetIssueContract{
				Name:        []byte("TRX"),
				TotalSupply: 1000000,
			}, nil
		},
	}
	c := newMockClient(t, mock)

	result, err := c.GetAssetIssueByName("TRX")
	require.NoError(t, err)
	assert.Equal(t, "TRX", string(result.Name))
	assert.Equal(t, int64(1000000), result.TotalSupply)
}

// ---------------------------------------------------------------------------
// GetAssetIssueByID
// ---------------------------------------------------------------------------

func TestGetAssetIssueByID_Success(t *testing.T) {
	mock := &mockWalletServer{
		GetAssetIssueByIdFunc: func(_ context.Context, in *api.BytesMessage) (*core.AssetIssueContract, error) {
			return &core.AssetIssueContract{
				Name:        []byte("TestToken"),
				TotalSupply: 5000000,
			}, nil
		},
	}
	c := newMockClient(t, mock)

	result, err := c.GetAssetIssueByID("1000001")
	require.NoError(t, err)
	assert.Equal(t, "TestToken", string(result.Name))
	assert.Equal(t, int64(5000000), result.TotalSupply)
}

// ---------------------------------------------------------------------------
// GetAssetIssueList
// ---------------------------------------------------------------------------

func TestGetAssetIssueList_AllPages(t *testing.T) {
	mock := &mockWalletServer{
		GetAssetIssueListFunc: func(_ context.Context, _ *api.EmptyMessage) (*api.AssetIssueList, error) {
			return &api.AssetIssueList{
				AssetIssue: []*core.AssetIssueContract{
					{Name: []byte("Token1")},
					{Name: []byte("Token2")},
				},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	result, err := c.GetAssetIssueList(-1)
	require.NoError(t, err)
	require.Len(t, result.AssetIssue, 2)
}

func TestGetAssetIssueList_Paginated(t *testing.T) {
	mock := &mockWalletServer{
		GetPaginatedAssetIssueListFunc: func(_ context.Context, in *api.PaginatedMessage) (*api.AssetIssueList, error) {
			// page=0, default limit=10: offset=0, limit=10
			assert.Equal(t, int64(0), in.Offset)
			assert.Equal(t, int64(10), in.Limit)
			return &api.AssetIssueList{
				AssetIssue: []*core.AssetIssueContract{
					{Name: []byte("Token1")},
				},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	result, err := c.GetAssetIssueList(0)
	require.NoError(t, err)
	require.Len(t, result.AssetIssue, 1)
}

func TestGetAssetIssueList_PaginatedWithCustomLimit(t *testing.T) {
	mock := &mockWalletServer{
		GetPaginatedAssetIssueListFunc: func(_ context.Context, in *api.PaginatedMessage) (*api.AssetIssueList, error) {
			// page=2, limit=5: offset=10, limit=5
			assert.Equal(t, int64(10), in.Offset)
			assert.Equal(t, int64(5), in.Limit)
			return &api.AssetIssueList{}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.GetAssetIssueList(2, 5)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// AssetIssue
// ---------------------------------------------------------------------------

func validAssetIssueArgs() (
	from, name, description, abbr, urlStr string,
	precision int32,
	totalSupply, startTime, endTime, freeAssetNetLimit, publicFreeAssetNetLimit int64,
	trxNum, icoNum, voteScore int32,
	frozenSupply map[string]string,
) {
	now := time.Now().UnixNano()/1000000 + 86400000 // +1 day
	return accountAddress, "TestToken", "A test token", "TT", "https://test.com",
		6,
		1000000, now, now + 86400000, 1000, 500,
		1, 1, 0,
		map[string]string{"1": "100"}
}

func TestAssetIssue_Success(t *testing.T) {
	mock := &mockWalletServer{
		CreateAssetIssue2Func: func(_ context.Context, in *core.AssetIssueContract) (*api.TransactionExtention, error) {
			assert.Equal(t, []byte("TestToken"), in.Name)
			assert.Equal(t, []byte("TT"), in.Abbr)
			assert.Equal(t, int32(6), in.Precision)
			assert.Equal(t, int64(1000000), in.TotalSupply)
			assert.Equal(t, int32(1), in.TrxNum)
			assert.Equal(t, int32(1), in.Num)
			assert.Equal(t, int64(1000), in.FreeAssetNetLimit)
			assert.Equal(t, int64(500), in.PublicFreeAssetNetLimit)
			assert.Equal(t, []byte("A test token"), in.Description)
			assert.Equal(t, []byte("https://test.com"), in.Url)
			require.Len(t, in.FrozenSupply, 1)
			assert.Equal(t, int64(100), in.FrozenSupply[0].FrozenAmount)
			assert.Equal(t, int64(1), in.FrozenSupply[0].FrozenDays)
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	tx, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestAssetIssue_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	_, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue("INVALID", name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen)
	require.Error(t, err)
}

func TestAssetIssue_PrecisionNegative(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, _, total, start, end, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, -1, total, start, end, free, pub, trx, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "precision")
}

func TestAssetIssue_PrecisionTooHigh(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, _, total, start, end, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, 7, total, start, end, free, pub, trx, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "precision")
}

func TestAssetIssue_TotalSupplyZero(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, prec, _, start, end, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, 0, start, end, free, pub, trx, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "total supply")
}

func TestAssetIssue_TrxNumZero(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, prec, total, start, end, free, pub, _, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, free, pub, 0, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trxNum")
}

func TestAssetIssue_IcoNumZero(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, _, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, 0, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "num <= 0")
}

func TestAssetIssue_StartTimeInPast(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, prec, total, _, _, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	pastTime := time.Now().UnixNano()/1000000 - 1000
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, pastTime, pastTime+86400000, free, pub, trx, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start time")
}

func TestAssetIssue_EndTimeBeforeStartTime(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, prec, total, start, _, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, start-1, free, pub, trx, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "end time")
}

func TestAssetIssue_NegativeFreeAssetNetLimit(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, prec, total, start, end, _, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, -1, pub, trx, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "free asset net limit")
}

func TestAssetIssue_NegativePublicFreeAssetNetLimit(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, prec, total, start, end, free, _, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, free, -1, trx, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "public free asset net limit")
}

func TestAssetIssue_InvalidFrozenSupplyAmount(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, _ := validAssetIssueArgs()
	badFrozen := map[string]string{"1": "notanumber"}
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, badFrozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "convert error")
}

func TestAssetIssue_InvalidFrozenSupplyDays(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, _ := validAssetIssueArgs()
	badFrozen := map[string]string{"notaday": "100"}
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, badFrozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "convert error")
}

func TestAssetIssue_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		CreateAssetIssue2Func: func(_ context.Context, _ *core.AssetIssueContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)

	from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestAssetIssue_ResultCodeError(t *testing.T) {
	mock := &mockWalletServer{
		CreateAssetIssue2Func: func(_ context.Context, _ *core.AssetIssueContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Txid: []byte{0x01},
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{},
				},
				Result: &api.Return{
					Result:  false,
					Code:    api.Return_BANDWITH_ERROR,
					Message: []byte("bandwidth error"),
				},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bandwidth error")
}

// ---------------------------------------------------------------------------
// UpdateAssetIssue
// ---------------------------------------------------------------------------

func TestUpdateAssetIssue_Success(t *testing.T) {
	expectedAddr, _ := common.DecodeCheck(accountAddress)
	mock := &mockWalletServer{
		UpdateAsset2Func: func(_ context.Context, in *core.UpdateAssetContract) (*api.TransactionExtention, error) {
			assert.Equal(t, expectedAddr, in.OwnerAddress)
			assert.Equal(t, []byte("updated desc"), in.Description)
			assert.Equal(t, []byte("https://updated.com"), in.Url)
			assert.Equal(t, int64(2000), in.NewLimit)
			assert.Equal(t, int64(1000), in.NewPublicLimit)
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	tx, err := c.UpdateAssetIssue(accountAddress, "updated desc", "https://updated.com", 2000, 1000)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestUpdateAssetIssue_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	_, err := c.UpdateAssetIssue("INVALID", "desc", "https://test.com", 100, 100)
	require.Error(t, err)
}

func TestUpdateAssetIssue_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		UpdateAsset2Func: func(_ context.Context, _ *core.UpdateAssetContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.UpdateAssetIssue(accountAddress, "desc", "https://test.com", 100, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestUpdateAssetIssue_ResultCodeError(t *testing.T) {
	mock := &mockWalletServer{
		UpdateAsset2Func: func(_ context.Context, _ *core.UpdateAssetContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Txid: []byte{0x01},
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{},
				},
				Result: &api.Return{
					Result:  false,
					Code:    api.Return_BANDWITH_ERROR,
					Message: []byte("bandwidth error"),
				},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.UpdateAssetIssue(accountAddress, "desc", "https://test.com", 100, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bandwidth error")
}

// ---------------------------------------------------------------------------
// TransferAsset
// ---------------------------------------------------------------------------

func TestTransferAsset_Success(t *testing.T) {
	expectedFrom, _ := common.DecodeCheck(accountAddress)
	expectedTo, _ := common.DecodeCheck(accountAddressWitness)
	mock := &mockWalletServer{
		TransferAsset2Func: func(_ context.Context, in *core.TransferAssetContract) (*api.TransactionExtention, error) {
			assert.Equal(t, expectedFrom, in.OwnerAddress)
			assert.Equal(t, expectedTo, in.ToAddress)
			assert.Equal(t, []byte("TRX"), in.AssetName)
			assert.Equal(t, int64(1000), in.Amount)
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	tx, err := c.TransferAsset(accountAddress, accountAddressWitness, "TRX", 1000)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestTransferAsset_InvalidFromAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	_, err := c.TransferAsset("INVALID", accountAddressWitness, "TRX", 1000)
	require.Error(t, err)
}

func TestTransferAsset_InvalidToAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	_, err := c.TransferAsset(accountAddress, "INVALID", "TRX", 1000)
	require.Error(t, err)
}

func TestTransferAsset_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		TransferAsset2Func: func(_ context.Context, _ *core.TransferAssetContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.TransferAsset(accountAddress, accountAddressWitness, "TRX", 1000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestTransferAsset_ResultCodeError(t *testing.T) {
	mock := &mockWalletServer{
		TransferAsset2Func: func(_ context.Context, _ *core.TransferAssetContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Txid: []byte{0x01},
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{},
				},
				Result: &api.Return{
					Result:  false,
					Code:    api.Return_BANDWITH_ERROR,
					Message: []byte("bandwidth error"),
				},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.TransferAsset(accountAddress, accountAddressWitness, "TRX", 1000)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bandwidth error")
}

// ---------------------------------------------------------------------------
// ParticipateAssetIssue
// ---------------------------------------------------------------------------

func TestParticipateAssetIssue_Success(t *testing.T) {
	expectedFrom, _ := common.DecodeCheck(accountAddress)
	expectedTo, _ := common.DecodeCheck(accountAddressWitness)
	mock := &mockWalletServer{
		ParticipateAssetIssue2Func: func(_ context.Context, in *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
			assert.Equal(t, expectedFrom, in.OwnerAddress)
			assert.Equal(t, expectedTo, in.ToAddress)
			assert.Equal(t, []byte("1000001"), in.AssetName)
			assert.Equal(t, int64(500), in.Amount)
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	tx, err := c.ParticipateAssetIssue(accountAddress, accountAddressWitness, "1000001", 500)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestParticipateAssetIssue_InvalidFromAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	_, err := c.ParticipateAssetIssue("INVALID", accountAddressWitness, "1000001", 500)
	require.Error(t, err)
}

func TestParticipateAssetIssue_InvalidIssuerAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	_, err := c.ParticipateAssetIssue(accountAddress, "INVALID", "1000001", 500)
	require.Error(t, err)
}

func TestParticipateAssetIssue_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		ParticipateAssetIssue2Func: func(_ context.Context, _ *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.ParticipateAssetIssue(accountAddress, accountAddressWitness, "1000001", 500)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestParticipateAssetIssue_ResultCodeError(t *testing.T) {
	mock := &mockWalletServer{
		ParticipateAssetIssue2Func: func(_ context.Context, _ *core.ParticipateAssetIssueContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Txid: []byte{0x01},
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{},
				},
				Result: &api.Return{
					Result:  false,
					Code:    api.Return_BANDWITH_ERROR,
					Message: []byte("bandwidth error"),
				},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.ParticipateAssetIssue(accountAddress, accountAddressWitness, "1000001", 500)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bandwidth error")
}

// ---------------------------------------------------------------------------
// UnfreezeAsset
// ---------------------------------------------------------------------------

func TestUnfreezeAsset_Success(t *testing.T) {
	expectedAddr, _ := common.DecodeCheck(accountAddress)
	mock := &mockWalletServer{
		UnfreezeAsset2Func: func(_ context.Context, in *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
			assert.Equal(t, expectedAddr, in.OwnerAddress)
			return fakeTxExtention(), nil
		},
	}
	c := newMockClient(t, mock)

	tx, err := c.UnfreezeAsset(accountAddress)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestUnfreezeAsset_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	_, err := c.UnfreezeAsset("INVALID")
	require.Error(t, err)
}

func TestUnfreezeAsset_BadTransaction(t *testing.T) {
	mock := &mockWalletServer{
		UnfreezeAsset2Func: func(_ context.Context, _ *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.UnfreezeAsset(accountAddress)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad transaction")
}

func TestUnfreezeAsset_ResultCodeError(t *testing.T) {
	mock := &mockWalletServer{
		UnfreezeAsset2Func: func(_ context.Context, _ *core.UnfreezeAssetContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Txid: []byte{0x01},
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{},
				},
				Result: &api.Return{
					Result:  false,
					Code:    api.Return_BANDWITH_ERROR,
					Message: []byte("bandwidth error"),
				},
			}, nil
		},
	}
	c := newMockClient(t, mock)

	_, err := c.UnfreezeAsset(accountAddress)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bandwidth error")
}

// ---------------------------------------------------------------------------
// AssetIssue - gRPC error propagation
// ---------------------------------------------------------------------------

func TestAssetIssue_GRPCError(t *testing.T) {
	mock := &mockWalletServer{
		CreateAssetIssue2Func: func(_ context.Context, _ *core.AssetIssueContract) (*api.TransactionExtention, error) {
			return nil, fmt.Errorf("grpc unavailable")
		},
	}
	c := newMockClient(t, mock)

	from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen := validAssetIssueArgs()
	_, err := c.AssetIssue(from, name, desc, abbr, url, prec, total, start, end, free, pub, trx, ico, vote, frozen)
	require.Error(t, err)
}
