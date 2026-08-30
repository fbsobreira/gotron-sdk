package cmd

import (
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
