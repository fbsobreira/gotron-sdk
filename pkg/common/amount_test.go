package common_test

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		decimals int
		want     int64
	}{
		// W48 regression: 1.1 TRX must be 1_100_000 SUN, not 1_099_999.
		{"1.1 TRX to SUN", "1.1", common.AmountDecimalPoint, 1_100_000},
		{"1 TRX to SUN", "1", common.AmountDecimalPoint, 1_000_000},
		{"0.000001 TRX is 1 SUN", "0.000001", common.AmountDecimalPoint, 1},
		{"integer with trailing point", "5.", common.AmountDecimalPoint, 5_000_000},
		{"leading plus", "+1.1", common.AmountDecimalPoint, 1_100_000},
		{"leading decimal point", ".5", 1, 5},
		{"half with 6 decimals", "0.5", common.AmountDecimalPoint, 500_000},
		{"zero integer", "0", 6, 0},
		{"zero fractional", "0.0", 6, 0},
		{"zero with extra trailing zeros", "0.000000", 6, 0},
		{"trailing zeros beyond scale are insignificant", "1.1000000", 6, 1_100_000},
		{"whitespace is trimmed", "  1.1  ", 6, 1_100_000},
		{"zero decimals integer", "42", 0, 42},
		{"18 decimals one token", "1", 18, 1_000_000_000_000_000_000},
		{"leading zeros", "0001.5", 1, 15},
		// W49: amounts above 2^53 SUN must round-trip exactly.
		{"90 billion TRX", "90000000000", common.AmountDecimalPoint, 90_000_000_000_000_000},
		{"2^53 + 1 SUN as TRX", "9007199254.740993", common.AmountDecimalPoint, 9_007_199_254_740_993},
		{"int64 max at 0 decimals", "9223372036854775807", 0, math.MaxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := common.ParseAmount(tt.in, tt.decimals)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseAmount_Rejects(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		decimals int
		err      string
	}{
		{"empty", "", 6, `"" is empty`},
		{"whitespace only", "   ", 6, "is empty"},
		{"negative", "-1", 6, `"-1" is negative`},
		{"negative fractional", "-0.1", 6, "is negative"},
		{"NaN", "NaN", 6, "not a finite number"},
		{"nan lower", "nan", 6, "not a finite number"},
		{"Inf", "Inf", 6, "not a finite number"},
		{"+Inf", "+Inf", 6, "not a finite number"},
		{"-Inf", "-Inf", 6, "not a finite number"},
		{"infinity", "Infinity", 6, "not a finite number"},
		{"garbage", "abc", 6, "invalid amount"},
		{"scientific notation", "1e6", 6, "invalid amount"},
		{"two dots", "1.2.3", 6, "invalid amount"},
		{"bare plus", "+", 6, "invalid amount"},
		{"bare dot", ".", 6, "invalid amount"},
		{"over-precise TRX", "1.1234567", 6, "more than 6 decimal places"},
		{"fraction against 0 decimals", "1.1", 0, "more than 0 decimal places"},
		{"overflow int64", "9223372036854775808", 0, "overflows int64"},
		{"overflow after scale", "9223372036855", 6, "overflows int64"},
		{"negative decimals", "1", -1, "decimals must be non-negative"},
		{"decimals above max", "1", common.MaxAmountDecimals + 1, "exceeds maximum"},
		{"hex", "0x10", 0, "invalid amount"},
		{"internal space", "1 000", 0, "invalid amount"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := common.ParseAmount(tt.in, tt.decimals)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.err)
		})
	}
}

func TestParseAmount_OnePointOneIsNotTruncated(t *testing.T) {
	// W48: the float64 path truncated 1.1 TRX to 1_099_999 SUN on some
	// conversions (int64(1.1 * 1e6)). Exact decimal parsing must yield
	// 1_100_000 regardless of how float64 rounds the same literals.
	got, err := common.ParseAmount("1.1", 6)
	require.NoError(t, err)
	assert.Equal(t, int64(1_100_000), got)
}

func TestParseAmount_AboveFloatMantissa(t *testing.T) {
	const ninetyBillionTRX int64 = 90_000_000_000_000_000
	got, err := common.ParseAmount("90000000000", 6)
	require.NoError(t, err)
	assert.Equal(t, ninetyBillionTRX, got)

	// 2^53+1 SUN cannot be represented in float64 (mantissa is 53 bits).
	const aboveMantissa int64 = 9_007_199_254_740_993
	got, err = common.ParseAmount("9007199254.740993", 6)
	require.NoError(t, err)
	assert.Equal(t, aboveMantissa, got)
}

func TestParseAmount_ErrorNamesInput(t *testing.T) {
	_, err := common.ParseAmount("-3.14", 6)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "-3.14"), "error must name the offending value")
}

func TestFormatAmount(t *testing.T) {
	tests := []struct {
		name     string
		units    int64
		decimals int
		want     string
	}{
		{"1.1 TRX", 1_100_000, common.AmountDecimalPoint, "1.1"},
		{"whole TRX", 1_000_000, common.AmountDecimalPoint, "1"},
		{"one SUN", 1, common.AmountDecimalPoint, "0.000001"},
		{"half", 500_000, common.AmountDecimalPoint, "0.5"},
		{"zero", 0, common.AmountDecimalPoint, "0"},
		{"above 2^53", 90_000_000_000_000_000, common.AmountDecimalPoint, "90000000000"},
		{"zero decimals", 42, 0, "42"},
		{"negative", -1_100_000, common.AmountDecimalPoint, "-1.1"},
		{"min int64", math.MinInt64, common.AmountDecimalPoint, "-9223372036854.775808"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, common.FormatAmount(tc.units, tc.decimals))
		})
	}
}

// FormatAmount output must always be a valid JSON number literal so the CLI
// can emit it as json.Number and keep `"amount": 1.1` rather than `"1.1"`.
func TestFormatAmount_IsValidJSONNumber(t *testing.T) {
	for _, units := range []int64{0, 1, 500_000, 1_100_000, 90_000_000_000_000_000, -1_100_000, math.MinInt64} {
		s := common.FormatAmount(units, common.AmountDecimalPoint)
		b, err := json.Marshal(map[string]interface{}{"amount": json.Number(s)})
		require.NoError(t, err, "FormatAmount(%d) produced invalid JSON number %q", units, s)
		assert.NotContains(t, string(b), `"amount":"`, "amount must stay a JSON number, not a string")
	}
}

// FormatAmount is the exact inverse of ParseAmount.
func TestFormatAmount_RoundTrip(t *testing.T) {
	for _, units := range []int64{0, 1, 500_000, 1_100_000, 8_200_000, 90_000_000_000_000_000} {
		s := common.FormatAmount(units, common.AmountDecimalPoint)
		got, err := common.ParseAmount(s, common.AmountDecimalPoint)
		require.NoError(t, err)
		assert.Equal(t, units, got, "round trip failed for %d via %q", units, s)
	}
}
