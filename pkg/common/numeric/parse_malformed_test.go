package numeric_test

import (
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common/numeric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The scientific-notation regex was unanchored, so malformed input like "1e5x"
// matched, strconv.Atoi("5x") failed silently leaving the exponent at 0, and the
// function returned 1 with a nil error — a silently wrong monetary value.
func TestNewDecFromString_Malformed(t *testing.T) {
	for _, in := range []string{
		"1e5x",
		"1e5.5",
		"x1e5",
		"1e5e5",
		"1e",
		"e5",
		"abc",
		"",
		" 1e5",
		"1e5 ",
		"--1",
		"1.2.3",
		// Pow negates a negative exponent; math.MinInt stays negative when
		// negated, so this recursed until the stack overflowed. Large positive
		// exponents build an astronomically large big.Int instead.
		"1e-9223372036854775808",
		"1e9223372036854775807",
		"1e100000",
		"1e-100000",
		// 10^77 exceeds Dec's bit cap and panicked with "Int overflow".
		"1e77",
		"1e-77",
	} {
		t.Run(in, func(t *testing.T) {
			require.NotPanics(t, func() {
				_, err := numeric.NewDecFromString(in)
				require.Error(t, err, "malformed input %q must not parse", in)
			})
		})
	}
}

func TestNewDecFromString_Valid(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"1e5", "100000.000000000000000000"},
		{"1e76", "1" + strings.Repeat("0", 76) + ".000000000000000000"},
		{"2.5e3", "2500.000000000000000000"},
		{".5", "0.500000000000000000"},
		{"1", "1.000000000000000000"},
		{"0", "0.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := numeric.NewDecFromString(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got.String())
		})
	}
}

func TestNewDecFromString_Negative(t *testing.T) {
	_, err := numeric.NewDecFromString("-1")
	require.Error(t, err)
}

// Invalid hex left a nil *big.Int from SetString flowing into big.Int.Mul, which
// panicked. The signature cannot report the condition, so these return ZeroDec.
func TestNewDecFromHex_Malformed(t *testing.T) {
	for _, in := range []string{
		"",
		"0x",
		"zz",
		"0xzz",
		"12g4",
		"0x 1",
		"-1",
		// big.Int.SetString accepts a sign, so these previously produced a
		// negative Dec out of a hex parser.
		"-abc",
		"-abcd",
		"+abcd",
		"0x-abcd",
	} {
		t.Run(in, func(t *testing.T) {
			require.NotPanics(t, func() {
				got := numeric.NewDecFromHex(in)
				assert.True(t, got.IsZero(), "malformed hex %q should give zero, got %s", in, got)
			})
		})
	}
}

func TestNewDecFromHex_Valid(t *testing.T) {
	require.NotPanics(t, func() {
		assert.False(t, numeric.NewDecFromHex("0xff").IsZero())
		assert.False(t, numeric.NewDecFromHex("ff").IsZero())
	})
}
