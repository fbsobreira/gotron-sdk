package cmd

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveTRC20Amount(t *testing.T) {
	e18, ok := new(big.Int).SetString("1000000000000000000", 10)
	require.True(t, ok)
	lookupErr := errors.New("constant call rejected")
	huge, ok := new(big.Int).SetString("100000000000000000000", 10)
	require.True(t, ok)

	tests := []struct {
		name        string
		amount      string
		override    int64
		lookup      *big.Int // returned by lookup when lookupErr is nil
		lookupErr   error
		wantLookup  bool
		want        *big.Int
		errContains string
	}{
		{
			name: "lookup decimals", amount: "1.5", override: -1,
			lookup: big.NewInt(6), wantLookup: true, want: big.NewInt(1_500_000),
		},
		{
			name: "18 decimals exceeds int64 range", amount: "1", override: -1,
			lookup: big.NewInt(18), wantLookup: true, want: e18,
		},
		{
			name: "lookup failure is fatal and suggests --decimals", amount: "1.5", override: -1,
			lookupErr: lookupErr, wantLookup: true, errContains: "pass --decimals",
		},
		{
			name: "override skips lookup", amount: "0.15", override: 2,
			wantLookup: false, want: big.NewInt(15),
		},
		{
			name: "override zero is honoured", amount: "7", override: 0,
			wantLookup: false, want: big.NewInt(7),
		},
		{
			name: "override above maximum", amount: "1", override: 300,
			wantLookup: false, errContains: "out of range",
		},
		{
			name: "looked-up decimals above maximum", amount: "1", override: -1,
			lookup: huge, wantLookup: true, errContains: "out of range",
		},
		{
			name: "negative looked-up decimals", amount: "1", override: -1,
			lookup: big.NewInt(-1), wantLookup: true, errContains: "out of range",
		},
		{
			name: "more fractional digits than decimals", amount: "1.1234567", override: -1,
			lookup: big.NewInt(6), wantLookup: true, errContains: "more than 6 decimal places",
		},
		{
			name: "fractional amount for zero-decimal token", amount: "1.5", override: 0,
			wantLookup: false, errContains: "more than 0 decimal places",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			got, err := resolveTRC20Amount(tc.amount, tc.override, func() (*big.Int, error) {
				called = true
				if tc.lookupErr != nil {
					return nil, tc.lookupErr
				}
				return tc.lookup, nil
			})

			assert.Equal(t, tc.wantLookup, called, "lookup called")
			if tc.errContains != "" {
				require.Error(t, err)
				assert.Nil(t, got, "no amount on error, so TRC20Send is never reached")
				assert.Contains(t, err.Error(), tc.errContains)
				if tc.lookupErr != nil {
					assert.ErrorIs(t, err, tc.lookupErr)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 0, tc.want.Cmp(got), "got %s, want %s", got, tc.want)
		})
	}
}
