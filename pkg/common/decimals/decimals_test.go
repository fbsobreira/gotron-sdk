package decimals_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common/decimals"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to build a *big.Float with 256-bit precision from a float64.
func bf(f float64) *big.Float {
	return decimals.NewFloat(f)
}

// helper to compare two big.Float values within a tolerance.
func assertClose(t *testing.T, expected, got *big.Float, tol float64, msgAndArgs ...interface{}) {
	t.Helper()
	diff := decimals.Abs(decimals.Sub(expected, got))
	tolF := decimals.NewFloat(tol)
	if !decimals.Lesser(diff, tolF) {
		t.Errorf("values not close enough: expected %s, got %s, diff %s, tolerance %s – %v",
			expected.Text('g', 20), got.Text('g', 20), diff.Text('g', 20), tolF.Text('g', 20), msgAndArgs)
	}
}

// ---------------------------------------------------------------------------
// NewFloat
// ---------------------------------------------------------------------------

func TestNewFloat(t *testing.T) {
	tests := []struct {
		name string
		in   float64
	}{
		{"zero", 0},
		{"positive integer", 42},
		{"negative", -3.14},
		{"very small", 1e-30},
		{"very large", 1e+60},
		{"max float64", math.MaxFloat64},
		{"smallest positive", math.SmallestNonzeroFloat64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := decimals.NewFloat(tt.in)
			require.NotNil(t, f)
			assert.Equal(t, uint(256), f.Prec(), "precision must be 256")
			got, _ := f.Float64()
			assert.InDelta(t, tt.in, got, math.Abs(tt.in)*1e-10+1e-300)
		})
	}
}

// ---------------------------------------------------------------------------
// Zero
// ---------------------------------------------------------------------------

func TestZero(t *testing.T) {
	z := decimals.Zero()
	require.NotNil(t, z)
	assert.Equal(t, uint(256), z.Prec())
	assert.Equal(t, 0, z.Sign(), "zero must have sign 0")
}

func TestZero_Independence(t *testing.T) {
	a := decimals.Zero()
	b := decimals.Zero()
	a.SetFloat64(99)
	assert.Equal(t, 0, b.Sign(), "Zero() must return independent values")
}

// ---------------------------------------------------------------------------
// Abs
// ---------------------------------------------------------------------------

func TestAbs(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{"positive", 5.5, 5.5},
		{"negative", -5.5, 5.5},
		{"zero", 0, 0},
		{"large negative", -1e+50, 1e+50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimals.Abs(bf(tt.in))
			assert.Equal(t, 0, got.Cmp(bf(tt.want)))
		})
	}
}

func TestAbs_DoesNotMutateInput(t *testing.T) {
	in := bf(-7)
	_ = decimals.Abs(in)
	assert.True(t, in.Sign() < 0, "Abs must not mutate the input")
}

// ---------------------------------------------------------------------------
// Add
// ---------------------------------------------------------------------------

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
		want float64
	}{
		{"positive+positive", 1, 2, 3},
		{"positive+negative", 5, -3, 2},
		{"negative+negative", -4, -6, -10},
		{"zero+value", 0, 42, 42},
		{"value+zero", 42, 0, 42},
		{"zero+zero", 0, 0, 0},
		{"cancel out", 7, -7, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimals.Add(bf(tt.a), bf(tt.b))
			assert.Equal(t, 0, got.Cmp(bf(tt.want)))
		})
	}
}

// ---------------------------------------------------------------------------
// Sub
// ---------------------------------------------------------------------------

func TestSub(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
		want float64
	}{
		{"positive-positive", 5, 3, 2},
		{"positive-negative", 5, -3, 8},
		{"negative-positive", -2, 3, -5},
		{"zero-value", 0, 10, -10},
		{"value-zero", 10, 0, 10},
		{"same values", 7, 7, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimals.Sub(bf(tt.a), bf(tt.b))
			assert.Equal(t, 0, got.Cmp(bf(tt.want)))
		})
	}
}

// ---------------------------------------------------------------------------
// Mul
// ---------------------------------------------------------------------------

func TestMul(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
		want float64
	}{
		{"positive*positive", 3, 4, 12},
		{"positive*negative", 3, -4, -12},
		{"negative*negative", -3, -4, 12},
		{"by zero", 99, 0, 0},
		{"zero by value", 0, 99, 0},
		{"by one", 7, 1, 7},
		{"fractional", 0.5, 0.5, 0.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimals.Mul(bf(tt.a), bf(tt.b))
			assert.Equal(t, 0, got.Cmp(bf(tt.want)))
		})
	}
}

// ---------------------------------------------------------------------------
// Div
// ---------------------------------------------------------------------------

func TestDiv(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
		want float64
	}{
		{"positive/positive", 12, 4, 3},
		{"positive/negative", 12, -4, -3},
		{"negative/negative", -12, -4, 3},
		{"zero/nonzero", 0, 5, 0},
		{"by one", 7, 1, 7},
		{"fractional result", 1, 3, 1.0 / 3.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimals.Div(bf(tt.a), bf(tt.b))
			gotF, _ := got.Float64()
			assert.InDelta(t, tt.want, gotF, 1e-10)
		})
	}
}

// ---------------------------------------------------------------------------
// Pow
// ---------------------------------------------------------------------------

func TestPow(t *testing.T) {
	tests := []struct {
		name string
		base float64
		exp  int64
		want float64
	}{
		{"2^1", 2, 1, 2},
		{"2^10", 2, 10, 1024},
		{"10^6", 10, 6, 1e6},
		{"3^3", 3, 3, 27},
		{"1^100", 1, 100, 1},
		{"0.5^2", 0.5, 2, 0.25},
		{"10^18 (TRX decimals)", 10, 18, 1e18},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimals.Pow(bf(tt.base), tt.exp)
			gotF, _ := got.Float64()
			assert.InDelta(t, tt.want, gotF, math.Abs(tt.want)*1e-10)
		})
	}
}

func TestPow_LargeExponent(t *testing.T) {
	// 2^64 should be exactly representable in 256-bit precision.
	got := decimals.Pow(bf(2), 64)
	expected := bf(0).SetUint64(math.MaxUint64) // 2^64 - 1
	expected = decimals.Add(expected, bf(1))    // 2^64
	assert.Equal(t, 0, got.Cmp(expected))
}

func TestPow_ExponentZero(t *testing.T) {
	// Mathematical identity: a^0 = 1.
	got := decimals.Pow(bf(5), 0)
	assert.Equal(t, 0, got.Cmp(bf(1)), "Pow(5, 0) must return 1")
}

// ---------------------------------------------------------------------------
// Root
// ---------------------------------------------------------------------------

func TestRoot(t *testing.T) {
	tests := []struct {
		name string
		val  float64
		n    uint64
		want float64
		tol  float64
	}{
		{"sqrt(4)", 4, 2, 2, 1e-15},
		{"sqrt(2)", 2, 2, math.Sqrt2, 1e-15},
		{"sqrt(9)", 9, 2, 3, 1e-15},
		{"cbrt(27)", 27, 3, 3, 1e-15},
		{"cbrt(8)", 8, 3, 2, 1e-15},
		{"4th root of 16", 16, 4, 2, 1e-15},
		{"sqrt(1)", 1, 2, 1, 1e-15},
		{"sqrt(100)", 100, 2, 10, 1e-14},
		{"cbrt(1000)", 1000, 3, 10, 1e-13},
		{"sqrt(0.25)", 0.25, 2, 0.5, 1e-15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimals.Root(bf(tt.val), tt.n)
			assertClose(t, bf(tt.want), got, tt.tol, tt.name)
		})
	}
}

func TestRoot_LargeValue(t *testing.T) {
	// sqrt(1e20) = 1e10
	got := decimals.Root(bf(1e20), 2)
	assertClose(t, bf(1e10), got, 1e-4)
}

// ---------------------------------------------------------------------------
// Lesser
// ---------------------------------------------------------------------------

func TestLesser(t *testing.T) {
	tests := []struct {
		name string
		x, y float64
		want bool
	}{
		{"less than", 1, 2, true},
		{"greater than", 2, 1, false},
		{"equal", 3, 3, false},
		{"negative less than positive", -1, 1, true},
		{"negative vs negative", -5, -3, true},
		{"zero vs positive", 0, 1, true},
		{"zero vs zero", 0, 0, false},
		{"positive vs zero", 1, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimals.Lesser(bf(tt.x), bf(tt.y))
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// FromString
// ---------------------------------------------------------------------------

func TestFromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
		ok    bool
	}{
		{"integer", "42", 42, true},
		{"negative", "-3.14", -3.14, true},
		{"zero", "0", 0, true},
		{"scientific notation", "1e18", 1e18, true},
		{"very precise", "3.141592653589793238462643383279", 3.141592653589793, true},
		{"large integer", "1000000000000000000", 1e18, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := decimals.FromString(tt.input)
			assert.Equal(t, tt.ok, ok)
			if ok {
				gotF, _ := got.Float64()
				assert.InDelta(t, tt.want, gotF, math.Abs(tt.want)*1e-10+1e-15)
			}
		})
	}
}

func TestFromString_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"letters", "abc"},
		{"special chars", "!@#"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := decimals.FromString(tt.input)
			assert.False(t, ok)
			assert.Nil(t, got)
		})
	}
}

func TestFromString_Precision(t *testing.T) {
	f, ok := decimals.FromString("42")
	require.True(t, ok)
	assert.Equal(t, uint(256), f.Prec(), "FromString must return 256-bit precision")
}

// ---------------------------------------------------------------------------
// ApplyDecimals
// ---------------------------------------------------------------------------

func TestApplyDecimals(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		decimals int64
		want     int64
	}{
		{"1 TRX to SUN (6 decimals)", 1, 6, 1_000_000},
		{"0.5 with 2 decimals", 0.5, 2, 50},
		{"2.5 with 1 decimal", 2.5, 1, 25},
		{"0.001 with 3 decimals", 0.001, 3, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := decimals.ApplyDecimals(bf(tt.value), tt.decimals)
			require.NotNil(t, got)
			assert.Equal(t, big.NewInt(tt.want).String(), got.String())
		})
	}
}

func TestApplyDecimals_LargeValue(t *testing.T) {
	// 1000000 TRX with 6 decimals = 1e12 SUN
	got, _ := decimals.ApplyDecimals(bf(1_000_000), 6)
	require.NotNil(t, got)
	expected := new(big.Int).SetUint64(1_000_000_000_000)
	assert.Equal(t, expected.String(), got.String())
}

func TestApplyDecimals_Zero(t *testing.T) {
	got, _ := decimals.ApplyDecimals(bf(0), 18)
	require.NotNil(t, got)
	assert.Equal(t, "0", got.String())
}

// ---------------------------------------------------------------------------
// RemoveDecimals
// ---------------------------------------------------------------------------

func TestRemoveDecimals(t *testing.T) {
	tests := []struct {
		name     string
		value    int64
		decimals int64
		want     float64
	}{
		{"1M SUN to TRX", 1_000_000, 6, 1.0},
		{"50 with 2 decimals", 50, 2, 0.5},
		{"25 with 1 decimal", 25, 1, 2.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decimals.RemoveDecimals(big.NewInt(tt.value), tt.decimals)
			gotF, _ := got.Float64()
			assert.InDelta(t, tt.want, gotF, 1e-10)
		})
	}
}

func TestRemoveDecimals_Zero(t *testing.T) {
	got := decimals.RemoveDecimals(big.NewInt(0), 18)
	assert.Equal(t, 0, got.Sign())
}

func TestRemoveDecimals_LargeValue(t *testing.T) {
	// 1e18 with 18 decimals = 1.0
	val := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	got := decimals.RemoveDecimals(val, 18)
	gotF, _ := got.Float64()
	assert.InDelta(t, 1.0, gotF, 1e-10)
}

// ---------------------------------------------------------------------------
// Roundtrip: ApplyDecimals <-> RemoveDecimals
// ---------------------------------------------------------------------------

func TestRoundtrip_ApplyThenRemove(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		decimals int64
	}{
		{"1 TRX", 1, 6},
		{"100 tokens", 100, 18},
		{"0.5 tokens", 0.5, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applied, _ := decimals.ApplyDecimals(bf(tt.value), tt.decimals)
			require.NotNil(t, applied)
			removed := decimals.RemoveDecimals(applied, tt.decimals)
			removedF, _ := removed.Float64()
			assert.InDelta(t, tt.value, removedF, 1e-10)
		})
	}
}

// ---------------------------------------------------------------------------
// Arithmetic identity properties
// ---------------------------------------------------------------------------

func TestArithmetic_Identities(t *testing.T) {
	v := bf(42.5)

	t.Run("additive identity", func(t *testing.T) {
		got := decimals.Add(v, bf(0))
		assert.Equal(t, 0, got.Cmp(v))
	})

	t.Run("multiplicative identity", func(t *testing.T) {
		got := decimals.Mul(v, bf(1))
		assert.Equal(t, 0, got.Cmp(v))
	})

	t.Run("sub self equals zero", func(t *testing.T) {
		got := decimals.Sub(v, v)
		assert.Equal(t, 0, got.Sign())
	})

	t.Run("div self equals one", func(t *testing.T) {
		got := decimals.Div(v, v)
		assert.Equal(t, 0, got.Cmp(bf(1)))
	})
}

// ---------------------------------------------------------------------------
// Immutability: operations should not mutate inputs
// ---------------------------------------------------------------------------

func TestImmutability(t *testing.T) {
	a := bf(10)
	b := bf(3)

	aStr := a.Text('g', 20)
	bStr := b.Text('g', 20)

	_ = decimals.Add(a, b)
	_ = decimals.Sub(a, b)
	_ = decimals.Mul(a, b)
	_ = decimals.Div(a, b)

	assert.Equal(t, aStr, a.Text('g', 20), "Add/Sub/Mul/Div must not mutate first operand")
	assert.Equal(t, bStr, b.Text('g', 20), "Add/Sub/Mul/Div must not mutate second operand")
}
