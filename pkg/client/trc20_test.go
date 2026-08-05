package client_test

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	client "github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// constantContractMock returns a mock that responds to TriggerConstantContract
// with the given raw bytes as constant result.
func constantContractMock(result []byte) *mockWalletServer {
	return &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true, Code: api.Return_SUCCESS},
				ConstantResult: [][]byte{result},
			}, nil
		},
	}
}

func TestTRC20ContractBalance(t *testing.T) {
	// 1000000 = 0xF4240
	result, _ := hex.DecodeString("00000000000000000000000000000000000000000000000000000000000f4240")
	c := newMockClient(t, constantContractMock(result))

	balance, err := c.TRC20ContractBalance("TMuA6YqfCeX8EhbfYEg5y7S4DqzSJireY9", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	assert.Equal(t, int64(1_000_000), balance.Int64())
}

func TestTRC20ContractBalance_InvalidAddress(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.TRC20ContractBalance("not-valid", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.Error(t, err)
}

func TestTRC20ContractBalance_InvalidContract(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.TRC20ContractBalance("TMuA6YqfCeX8EhbfYEg5y7S4DqzSJireY9", "not-valid")
	require.Error(t, err)
}

func TestTRC20GetDecimals(t *testing.T) {
	tests := []struct {
		name     string
		hexValue string
		expected int64
	}{
		{"zero", "0000000000000000000000000000000000000000000000000000000000000000", 0},
		{"six", "0000000000000000000000000000000000000000000000000000000000000006", 6},
		{"eighteen", "0000000000000000000000000000000000000000000000000000000000000012", 18},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := hex.DecodeString(tt.hexValue)
			c := newMockClient(t, constantContractMock(result))

			decimals, err := c.TRC20GetDecimals("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, decimals.Int64())
		})
	}
}

func TestTRC20GetName(t *testing.T) {
	// ABI-encoded string "Tether USD":
	// offset (32) + length (10) + data
	nameHex := "0000000000000000000000000000000000000000000000000000000000000020" + // offset = 32
		"000000000000000000000000000000000000000000000000000000000000000a" + // length = 10
		"5465746865722055534400000000000000000000000000000000000000000000" // "Tether USD"
	result, _ := hex.DecodeString(nameHex)
	c := newMockClient(t, constantContractMock(result))

	name, err := c.TRC20GetName("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	assert.Equal(t, "Tether USD", name)
}

func TestTRC20GetSymbol(t *testing.T) {
	// ABI-encoded string "USDT":
	symbolHex := "0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000004" +
		"5553445400000000000000000000000000000000000000000000000000000000"
	result, _ := hex.DecodeString(symbolHex)
	c := newMockClient(t, constantContractMock(result))

	symbol, err := c.TRC20GetSymbol("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	assert.Equal(t, "USDT", symbol)
}

func TestTRC20GetName_32ByteUTF8(t *testing.T) {
	// Some tokens return name as 32 bytes of raw UTF-8 (no offset/length)
	// "WTRX" padded with zeros to 32 bytes
	nameHex := "5754525800000000000000000000000000000000000000000000000000000000"
	result, _ := hex.DecodeString(nameHex)
	c := newMockClient(t, constantContractMock(result))

	name, err := c.TRC20GetName("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.NoError(t, err)
	assert.Equal(t, "WTRX", name)
}

func TestTRC20Send(t *testing.T) {
	mock := &mockWalletServer{
		TriggerContractFunc: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			// Verify the data contains the transfer method signature (a9059cbb)
			dataHex := hex.EncodeToString(in.Data)
			assert.True(t, len(dataHex) >= 8)
			assert.Equal(t, "a9059cbb", dataHex[:8])

			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Txid:        []byte{0x01},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	amount := big.NewInt(1_000_000)
	tx, err := c.TRC20Send(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		amount,
		100_000_000,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestTRC20Send_InvalidTo(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})
	_, err := c.TRC20Send("TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", "invalid", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", big.NewInt(1), 100)
	require.Error(t, err)
}

func TestTRC20Approve(t *testing.T) {
	mock := &mockWalletServer{
		TriggerContractFunc: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			dataHex := hex.EncodeToString(in.Data)
			// approve method signature: 095ea7b3
			assert.Equal(t, "095ea7b3", dataHex[:8])
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Txid:        []byte{0x02},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.TRC20Approve(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		big.NewInt(999_999_999),
		100_000_000,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestTRC20TransferFrom(t *testing.T) {
	mock := &mockWalletServer{
		TriggerContractFunc: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			dataHex := hex.EncodeToString(in.Data)
			// transferFrom method signature: 23b872dd
			assert.Equal(t, "23b872dd", dataHex[:8])
			return &api.TransactionExtention{
				Result:      &api.Return{Result: true, Code: api.Return_SUCCESS},
				Txid:        []byte{0x03},
				Transaction: &core.Transaction{RawData: &core.TransactionRaw{}},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	tx, err := c.TRC20TransferFrom(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", // owner
		"TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", // from
		"TUoHaVjx7n5xz8LwPRDckgFrDWhMhuSuJM", // to
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", // contract
		big.NewInt(500_000),
		100_000_000,
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
}

func TestTRC20TransferFrom_InvalidOwner(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	t.Run("empty owner", func(t *testing.T) {
		_, err := c.TRC20TransferFrom("", "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", "TUoHaVjx7n5xz8LwPRDckgFrDWhMhuSuJM", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", big.NewInt(1), 100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid owner address")
	})

	t.Run("invalid owner", func(t *testing.T) {
		_, err := c.TRC20TransferFrom("not-valid", "TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", "TUoHaVjx7n5xz8LwPRDckgFrDWhMhuSuJM", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", big.NewInt(1), 100)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid owner address")
	})
}

func TestTRC20Call_RPCError(t *testing.T) {
	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return nil, fmt.Errorf("server unavailable")
		},
	}

	c := newMockClient(t, mock)
	_, err := c.TRC20GetDecimals("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "server unavailable")
}

func TestTRC20Call_ResultCodeError(t *testing.T) {
	mock := &mockWalletServer{
		TriggerConstantContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			return &api.TransactionExtention{
				Result: &api.Return{
					Result:  false,
					Code:    api.Return_CONTRACT_EXE_ERROR,
					Message: []byte("REVERT"),
				},
			}, nil
		},
	}

	c := newMockClient(t, mock)
	_, err := c.TRC20GetDecimals("TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REVERT")
}

func TestParseTRC20NumericProperty(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	t.Run("with 0x prefix", func(t *testing.T) {
		val, err := c.ParseTRC20NumericProperty("0x0000000000000000000000000000000000000000000000000000000000000012")
		require.NoError(t, err)
		assert.Equal(t, int64(18), val.Int64())
	})

	t.Run("without prefix", func(t *testing.T) {
		val, err := c.ParseTRC20NumericProperty("0000000000000000000000000000000000000000000000000000000000000006")
		require.NoError(t, err)
		assert.Equal(t, int64(6), val.Int64())
	})

	t.Run("empty string", func(t *testing.T) {
		val, err := c.ParseTRC20NumericProperty("")
		require.NoError(t, err)
		assert.Equal(t, int64(0), val.Int64())
	})

	t.Run("invalid hex", func(t *testing.T) {
		_, err := c.ParseTRC20NumericProperty("not-hex-at-all")
		require.Error(t, err)
	})

	t.Run("wrong length", func(t *testing.T) {
		_, err := c.ParseTRC20NumericProperty("abcdef")
		require.Error(t, err)
	})
}

func TestParseTRC20StringProperty(t *testing.T) {
	c := newMockClient(t, &mockWalletServer{})

	t.Run("ABI encoded with offset/length", func(t *testing.T) {
		// "Hello" encoded: offset=32, length=5, data
		data := "0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000005" +
			"48656c6c6f000000000000000000000000000000000000000000000000000000"
		val, err := c.ParseTRC20StringProperty(data)
		require.NoError(t, err)
		assert.Equal(t, "Hello", val)
	})

	t.Run("32-byte raw UTF-8", func(t *testing.T) {
		// "TRX" as 32 bytes
		data := "5452580000000000000000000000000000000000000000000000000000000000"
		val, err := c.ParseTRC20StringProperty(data)
		require.NoError(t, err)
		assert.Equal(t, "TRX", val)
	})

	t.Run("with 0x prefix", func(t *testing.T) {
		data := "0x0000000000000000000000000000000000000000000000000000000000000020" +
			"0000000000000000000000000000000000000000000000000000000000000003" +
			"4142430000000000000000000000000000000000000000000000000000000000"
		val, err := c.ParseTRC20StringProperty(data)
		require.NoError(t, err)
		assert.Equal(t, "ABC", val)
	})

	t.Run("empty data", func(t *testing.T) {
		_, err := c.ParseTRC20StringProperty("")
		require.Error(t, err)
	})

	// A hostile token contract controls the declared string length, which reaches
	// here from TRC20GetName and TRC20GetSymbol. Each length below broke the old
	// bounds check in one of two ways, both verified against the original code:
	//
	//	panic         2*int(l) overflowed to a negative value, so the check passed
	//	              and the slice then ran with a huge upper bound.
	//	silent empty  Uint64 truncated a >64-bit length to its low bits, or 2*l
	//	              wrapped to exactly 0, yielding "" with a nil error.
	//
	// Both are now rejected: IsUint64 catches the truncation, and the bound is a
	// division so no multiplication can overflow.
	t.Run("hostile declared length", func(t *testing.T) {
		lengthWord := func(v string) string { return strings.Repeat("0", 64-len(v)) + v }
		for _, tc := range []struct {
			name, length, oldBehaviour string
		}{
			{"2^62", lengthWord("4000000000000000"), "panic"},
			{"2^63", lengthWord("8000000000000000"), "silent empty"},
			{"max uint64", lengthWord("ffffffffffffffff"), "panic"},
			{"2^254", "4" + strings.Repeat("0", 63), "silent empty"},
			{"max uint256", strings.Repeat("f", 64), "panic"},
			{"exceeds payload", lengthWord("40"), "correctly rejected"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				data := "0000000000000000000000000000000000000000000000000000000000000020" +
					tc.length +
					"48656c6c6f000000000000000000000000000000000000000000000000000000"
				require.NotPanics(t, func() {
					_, err := c.ParseTRC20StringProperty(data)
					require.Error(t, err, "old behaviour here was: %s", tc.oldBehaviour)
				})
			})
		}
	})
}

// estimateMock returns a mock that only handles TriggerConstantContract,
// verifying WithEstimate() routes to the dry-run path.
func estimateMock(t *testing.T, wantSigPrefix string) *mockWalletServer {
	t.Helper()
	return &mockWalletServer{
		TriggerContractFunc: func(_ context.Context, _ *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			require.Fail(t, "unexpected TriggerContract call: WithEstimate() must use TriggerConstantContract")
			return nil, nil
		},
		TriggerConstantContractFunc: func(_ context.Context, in *core.TriggerSmartContract) (*api.TransactionExtention, error) {
			dataHex := hex.EncodeToString(in.Data)
			assert.True(t, len(dataHex) >= 8, "data too short")
			assert.Equal(t, wantSigPrefix, dataHex[:8])
			return &api.TransactionExtention{
				Result:         &api.Return{Result: true, Code: api.Return_SUCCESS},
				ConstantResult: [][]byte{{0x01}},
			}, nil
		},
	}
}

func TestTRC20Send_WithEstimate(t *testing.T) {
	// a9059cbb = transfer(address,uint256)
	c := newMockClient(t, estimateMock(t, "a9059cbb"))

	tx, err := c.TRC20Send(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		big.NewInt(1_000_000),
		100_000_000,
		client.WithEstimate(),
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetConstantResult())
}

func TestTRC20Approve_WithEstimate(t *testing.T) {
	// 095ea7b3 = approve(address,uint256)
	c := newMockClient(t, estimateMock(t, "095ea7b3"))

	tx, err := c.TRC20Approve(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b",
		"TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH",
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
		big.NewInt(999_999_999),
		100_000_000,
		client.WithEstimate(),
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetConstantResult())
}

func TestTRC20TransferFrom_WithEstimate(t *testing.T) {
	// 23b872dd = transferFrom(address,address,uint256)
	c := newMockClient(t, estimateMock(t, "23b872dd"))

	tx, err := c.TRC20TransferFrom(
		"TPpw7soPWEDQWXPCGUMagYPryaWrYR5b3b", // owner
		"TLyqzVGLV1srkB7dToTAEqgDSfPtXRJZYH", // from
		"TUoHaVjx7n5xz8LwPRDckgFrDWhMhuSuJM", // to
		"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t", // contract
		big.NewInt(500_000),
		100_000_000,
		client.WithEstimate(),
	)
	require.NoError(t, err)
	require.NotNil(t, tx)
	assert.NotEmpty(t, tx.GetConstantResult())
}
