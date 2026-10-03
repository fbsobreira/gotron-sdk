package cmd

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTRXArg_OnePointOne(t *testing.T) {
	got, err := parseTRXArg("1.1", "AMOUNT")
	require.NoError(t, err)
	assert.Equal(t, int64(1_100_000), got)
}

func TestParseTRXArg_AboveFloatMantissa(t *testing.T) {
	got, err := parseTRXArg("90000000000", "AMOUNT")
	require.NoError(t, err)
	assert.Equal(t, int64(90_000_000_000_000_000), got)
}

func TestParseAmountArg_NamesArgument(t *testing.T) {
	_, err := parseAmountArg("-1", "AMOUNT", 6)
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "AMOUNT:"), "error must name the CLI argument")
	assert.Contains(t, err.Error(), "negative")
}

func TestParseIssueRatio(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantTrx   int32
		wantToken int32
	}{
		{"colon integers", "1:2", 1, 2},
		{"whole number", "2", 2, 1},
		{"one and a half", "1.5", 15, 10},
		{"six decimal places", "0.000001", 1, 1_000_000},
		{"trailing zeros", "2.0", 2, 1},
		{"zero", "0", 0, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trx, token, err := parseIssueRatio(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.wantTrx, trx)
			assert.Equal(t, tt.wantToken, token)
		})
	}
}

func TestParseIssueRatio_Rejects(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"over-precise", "1.1234567"},
		{"negative decimal", "-1.5"},
		{"negative colon", "1:-2"},
		{"garbage", "abc"},
		{"empty", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseIssueRatio(tt.in)
			require.Error(t, err)
		})
	}
}

func TestRoundBancorQuote(t *testing.T) {
	got, err := roundBancorQuote(100, 100, 100)
	require.NoError(t, err)
	assert.Equal(t, int64(50), got)

	got, err = roundBancorQuote(1, 3, 2)
	require.NoError(t, err)
	// 1*2/(3+1) = 0.5 → nearest is 1 with half-up via (num+den/2)/den
	assert.Equal(t, int64(1), got)
}

func TestRoundBancorQuote_ZeroReserve(t *testing.T) {
	_, err := roundBancorQuote(10, -10, 1)
	require.Error(t, err)
}

// Regression: --expected and --tokenValue used to be float64 and were tested
// with `== 0`. After the switch to string flags, comparing against the literal
// "0" treated "0.0" as a real amount — silently submitting a zero minimum
// return on exchange trade, and forcing an asset lookup on contract trigger.
func TestIsDecimalZero(t *testing.T) {
	zero := []string{"0", "0.0", "0.000", "00", ".0", "+0", "+0.0", "0.000000", " 0 "}
	for _, s := range zero {
		if !isDecimalZero(s) {
			t.Errorf("isDecimalZero(%q) = false, want true", s)
		}
	}

	nonZero := []string{"1", "0.5", "0.000001", "1.1", "0.0000000001", "-0", "abc", "", "1e0"}
	for _, s := range nonZero {
		if isDecimalZero(s) {
			t.Errorf("isDecimalZero(%q) = true, want false", s)
		}
	}
}

// W61: trc10 ico reported tokenAmount = spent * (trxNum/num), inverting the
// rate. Per java-tron's ParticipateAssetIssueActuator the tokens received are
// floor(spent * num / trxNum).
func TestIcoTokensReceived(t *testing.T) {
	tests := []struct {
		name     string
		spentSUN int64
		trxNum   int32
		num      int32
		want     int64
	}{
		{"one to one", 1_000_000, 1, 1, 1_000_000},
		{"ten tokens per sun", 1_000_000, 1, 10, 10_000_000},
		{"one token per ten sun", 1_000_000, 10, 1, 100_000},
		{"issue ratio 1.5 (15:10)", 1_000_000, 15, 10, 666_666},
		{"floors, does not round", 7, 2, 1, 3},
		{"product near int64 max", 90_000_000_000_000_000, 1, 100, 9_000_000_000_000_000_000},
		{"product exactly int64 max", math.MaxInt64, 1, 1, math.MaxInt64},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := icoTokensReceived(tc.spentSUN, tc.trxNum, tc.num)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("icoTokensReceived(%d, %d, %d) = %d, want %d",
					tc.spentSUN, tc.trxNum, tc.num, got, tc.want)
			}
		})
	}
}

// A zero or negative rate must error rather than divide by zero. The old code
// produced +Inf here, which json.Marshal refuses to encode — and the marshal
// error is discarded, so the command printed nothing (see issue #301).
func TestIcoTokensReceived_RejectsBadRate(t *testing.T) {
	for _, tc := range []struct{ trxNum, num int32 }{
		{0, 1}, {1, 0}, {0, 0}, {-1, 1}, {1, -1},
	} {
		if _, err := icoTokensReceived(1_000_000, tc.trxNum, tc.num); err == nil {
			t.Errorf("icoTokensReceived(_, %d, %d) = nil error, want error", tc.trxNum, tc.num)
		}
	}
}

// java-tron computes spent*num with multiplyExact, so an overflowing product
// is rejected on chain even when the final quotient would fit. The helper
// must reject it too, before the purchase is submitted.
func TestIcoTokensReceived_Overflow(t *testing.T) {
	for _, tc := range []struct {
		name     string
		spentSUN int64
		trxNum   int32
		num      int32
	}{
		{"result overflows", math.MaxInt64, 1, 1000},
		{"product overflows, quotient fits", 5_000_000_000_000, 2_000_000, 2_000_000},
		{"one past int64 max", math.MaxInt64/2 + 1, 1, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := icoTokensReceived(tc.spentSUN, tc.trxNum, tc.num)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "overflows int64")
		})
	}
}

// java-tron rejects a purchase whose token amount floors to zero.
func TestIcoTokensReceived_RejectsZeroTokens(t *testing.T) {
	for _, tc := range []struct {
		name     string
		spentSUN int64
		trxNum   int32
		num      int32
	}{
		{"zero spend", 0, 1, 1},
		{"spend below one token unit", 9, 10, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := icoTokensReceived(tc.spentSUN, tc.trxNum, tc.num)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "buys no tokens")
		})
	}
}
