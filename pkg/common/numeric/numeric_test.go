package numeric_test

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/common/numeric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Constructors
// ---------------------------------------------------------------------------

func TestZeroDec(t *testing.T) {
	d := numeric.ZeroDec()
	assert.True(t, d.IsZero(), "ZeroDec should be zero")
	assert.False(t, d.IsNil(), "ZeroDec should not be nil")
	assert.Equal(t, "0.000000000000000000", d.String())
}

func TestOneDec(t *testing.T) {
	d := numeric.OneDec()
	assert.True(t, d.Equal(numeric.NewDec(1)), "OneDec should equal NewDec(1)")
	assert.Equal(t, "1.000000000000000000", d.String())
}

func TestSmallestDec(t *testing.T) {
	d := numeric.SmallestDec()
	assert.True(t, d.IsPositive())
	assert.True(t, d.LT(numeric.OneDec()))
	assert.Equal(t, "0.000000000000000001", d.String())
}

func TestNewDec(t *testing.T) {
	tests := []struct {
		name string
		val  int64
		want string
	}{
		{"zero", 0, "0.000000000000000000"},
		{"positive", 42, "42.000000000000000000"},
		{"negative", -7, "-7.000000000000000000"},
		{"large", 1000000, "1000000.000000000000000000"},
		{"one", 1, "1.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := numeric.NewDec(tt.val)
			assert.Equal(t, tt.want, d.String())
		})
	}
}

func TestNewDecWithPrec(t *testing.T) {
	tests := []struct {
		name string
		val  int64
		prec int64
		want string
	}{
		{"whole number prec 0", 5, 0, "5.000000000000000000"},
		{"prec 1", 15, 1, "1.500000000000000000"},
		{"prec 2", 150, 2, "1.500000000000000000"},
		{"prec 6", 123456, 6, "0.123456000000000000"},
		{"prec 18 (max)", 1, 18, "0.000000000000000001"},
		{"negative with prec", -25, 1, "-2.500000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := numeric.NewDecWithPrec(tt.val, tt.prec)
			assert.Equal(t, tt.want, d.String())
		})
	}
}

func TestNewDecWithPrec_PanicOnOverflow(t *testing.T) {
	assert.Panics(t, func() {
		numeric.NewDecWithPrec(1, 19) // prec > Precision should panic
	})
}

func TestNewDecFromBigInt(t *testing.T) {
	tests := []struct {
		name string
		val  *big.Int
		want string
	}{
		{"zero", big.NewInt(0), "0.000000000000000000"},
		{"positive", big.NewInt(100), "100.000000000000000000"},
		{"negative", big.NewInt(-50), "-50.000000000000000000"},
		{"large", new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil), "1000000000000000000000000000000.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := numeric.NewDecFromBigInt(tt.val)
			assert.Equal(t, tt.want, d.String())
		})
	}
}

func TestNewDecFromBigIntWithPrec(t *testing.T) {
	d := numeric.NewDecFromBigIntWithPrec(big.NewInt(12345), 3)
	assert.Equal(t, "12.345000000000000000", d.String())
}

func TestNewDecFromInt(t *testing.T) {
	d := numeric.NewDecFromInt(big.NewInt(99))
	assert.Equal(t, "99.000000000000000000", d.String())
}

func TestNewDecFromIntWithPrec(t *testing.T) {
	d := numeric.NewDecFromIntWithPrec(big.NewInt(5000), 4)
	assert.Equal(t, "0.500000000000000000", d.String())
}

func TestNewDecFromStr(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"integer", "123", "123.000000000000000000", false},
		{"negative integer", "-456", "-456.000000000000000000", false},
		{"decimal", "1.5", "1.500000000000000000", false},
		{"negative decimal", "-3.14", "-3.140000000000000000", false},
		{"long decimal", "0.123456789012345678", "0.123456789012345678", false},
		{"zero", "0", "0.000000000000000000", false},
		{"leading zeros", "007", "7.000000000000000000", false},
		{"trailing zeros", "1.10", "1.100000000000000000", false},
		{"purely fractional", "0.001", "0.001000000000000000", false},
		{"large number", "999999999999999999", "999999999999999999.000000000000000000", false},

		// Error cases
		{"empty string", "", "", true},
		{"just dash", "-", "", true},
		{"too many decimals", "1.1234567890123456789", "", true},
		{"multiple dots", "1.2.3", "", true},
		{"bad decimal no fraction", "1.", "", true},
		{"bad decimal no whole", ".1", "", true},
		{"non-numeric", "abc", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := numeric.NewDecFromStr(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, d.String())
		})
	}
}

func TestMustNewDecFromStr(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		d := numeric.MustNewDecFromStr("3.14")
		assert.Equal(t, "3.140000000000000000", d.String())
	})
	t.Run("panics on invalid", func(t *testing.T) {
		assert.Panics(t, func() {
			numeric.MustNewDecFromStr("not-a-number")
		})
	})
}

// ---------------------------------------------------------------------------
// Predicates
// ---------------------------------------------------------------------------

func TestIsNil(t *testing.T) {
	var nilDec numeric.Dec
	assert.True(t, nilDec.IsNil(), "default Dec should be nil")
	assert.False(t, numeric.ZeroDec().IsNil(), "ZeroDec should not be nil")
}

func TestIsZero(t *testing.T) {
	assert.True(t, numeric.ZeroDec().IsZero())
	assert.True(t, numeric.NewDec(0).IsZero())
	assert.False(t, numeric.OneDec().IsZero())
	assert.False(t, numeric.NewDec(-1).IsZero())
}

func TestIsNegative(t *testing.T) {
	assert.True(t, numeric.NewDec(-1).IsNegative())
	assert.False(t, numeric.ZeroDec().IsNegative())
	assert.False(t, numeric.OneDec().IsNegative())
}

func TestIsPositive(t *testing.T) {
	assert.True(t, numeric.OneDec().IsPositive())
	assert.True(t, numeric.NewDec(100).IsPositive())
	assert.False(t, numeric.ZeroDec().IsPositive())
	assert.False(t, numeric.NewDec(-1).IsPositive())
}

func TestComparisons(t *testing.T) {
	one := numeric.NewDec(1)
	two := numeric.NewDec(2)
	alsoOne := numeric.NewDec(1)
	neg := numeric.NewDec(-5)

	t.Run("Equal", func(t *testing.T) {
		assert.True(t, one.Equal(alsoOne))
		assert.False(t, one.Equal(two))
	})
	t.Run("GT", func(t *testing.T) {
		assert.True(t, two.GT(one))
		assert.False(t, one.GT(two))
		assert.False(t, one.GT(alsoOne))
	})
	t.Run("GTE", func(t *testing.T) {
		assert.True(t, two.GTE(one))
		assert.True(t, one.GTE(alsoOne))
		assert.False(t, one.GTE(two))
	})
	t.Run("LT", func(t *testing.T) {
		assert.True(t, one.LT(two))
		assert.True(t, neg.LT(one))
		assert.False(t, two.LT(one))
		assert.False(t, one.LT(alsoOne))
	})
	t.Run("LTE", func(t *testing.T) {
		assert.True(t, one.LTE(two))
		assert.True(t, one.LTE(alsoOne))
		assert.False(t, two.LTE(one))
	})
}

func TestIsInteger(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		want bool
	}{
		{"integer 5", numeric.NewDec(5), true},
		{"integer 0", numeric.ZeroDec(), true},
		{"integer -3", numeric.NewDec(-3), true},
		{"fractional 1.5", numeric.MustNewDecFromStr("1.5"), false},
		{"smallest dec", numeric.SmallestDec(), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dec.IsInteger())
		})
	}
}

// ---------------------------------------------------------------------------
// Arithmetic
// ---------------------------------------------------------------------------

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		a, b numeric.Dec
		want string
	}{
		{"positive + positive", numeric.NewDec(3), numeric.NewDec(7), "10.000000000000000000"},
		{"positive + negative", numeric.NewDec(10), numeric.NewDec(-3), "7.000000000000000000"},
		{"zero + value", numeric.ZeroDec(), numeric.NewDec(5), "5.000000000000000000"},
		{"negative + negative", numeric.NewDec(-2), numeric.NewDec(-3), "-5.000000000000000000"},
		{"fractional", numeric.MustNewDecFromStr("1.5"), numeric.MustNewDecFromStr("2.3"), "3.800000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Add(tt.b)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestSub(t *testing.T) {
	tests := []struct {
		name string
		a, b numeric.Dec
		want string
	}{
		{"positive - positive", numeric.NewDec(10), numeric.NewDec(3), "7.000000000000000000"},
		{"result negative", numeric.NewDec(3), numeric.NewDec(10), "-7.000000000000000000"},
		{"sub zero", numeric.NewDec(5), numeric.ZeroDec(), "5.000000000000000000"},
		{"from zero", numeric.ZeroDec(), numeric.NewDec(5), "-5.000000000000000000"},
		{"identical values", numeric.NewDec(42), numeric.NewDec(42), "0.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Sub(tt.b)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestMul(t *testing.T) {
	tests := []struct {
		name string
		a, b numeric.Dec
		want string
	}{
		{"simple", numeric.NewDec(3), numeric.NewDec(4), "12.000000000000000000"},
		{"by zero", numeric.NewDec(100), numeric.ZeroDec(), "0.000000000000000000"},
		{"by one", numeric.NewDec(7), numeric.OneDec(), "7.000000000000000000"},
		{"negative * positive", numeric.NewDec(-3), numeric.NewDec(4), "-12.000000000000000000"},
		{"negative * negative", numeric.NewDec(-3), numeric.NewDec(-4), "12.000000000000000000"},
		{"fractional", numeric.MustNewDecFromStr("1.5"), numeric.MustNewDecFromStr("2.0"), "3.000000000000000000"},
		{"small fractions", numeric.MustNewDecFromStr("0.1"), numeric.MustNewDecFromStr("0.1"), "0.010000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Mul(tt.b)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestMulTruncate(t *testing.T) {
	// MulTruncate truncates instead of rounding
	a := numeric.MustNewDecFromStr("1.000000000000000001")
	b := numeric.NewDec(3)
	result := a.MulTruncate(b)
	// 1.000000000000000001 * 3 = 3.000000000000000003
	assert.Equal(t, "3.000000000000000003", result.String())
}

func TestMulInt(t *testing.T) {
	d := numeric.MustNewDecFromStr("2.5")
	result := d.MulInt(big.NewInt(4))
	assert.Equal(t, "10.000000000000000000", result.String())
}

func TestMulInt64(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		mul  int64
		want string
	}{
		{"simple", numeric.NewDec(5), 3, "15.000000000000000000"},
		{"by zero", numeric.NewDec(5), 0, "0.000000000000000000"},
		{"by one", numeric.NewDec(5), 1, "5.000000000000000000"},
		{"negative multiplier", numeric.NewDec(5), -2, "-10.000000000000000000"},
		{"fractional dec", numeric.MustNewDecFromStr("0.5"), 6, "3.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dec.MulInt64(tt.mul)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestQuo(t *testing.T) {
	tests := []struct {
		name string
		a, b numeric.Dec
		want string
	}{
		{"exact division", numeric.NewDec(10), numeric.NewDec(2), "5.000000000000000000"},
		{"fractional result", numeric.NewDec(1), numeric.NewDec(3), "0.333333333333333333"},
		{"by one", numeric.NewDec(7), numeric.OneDec(), "7.000000000000000000"},
		{"negative / positive", numeric.NewDec(-10), numeric.NewDec(3), "-3.333333333333333333"},
		{"large / small", numeric.NewDec(1000000), numeric.NewDec(7), "142857.142857142857142857"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.a.Quo(tt.b)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestQuo_DivisionByZeroPanics(t *testing.T) {
	assert.Panics(t, func() {
		numeric.NewDec(1).Quo(numeric.ZeroDec())
	})
}

func TestQuoTruncate(t *testing.T) {
	// 1 / 3 truncated should not round the last digit
	result := numeric.NewDec(1).QuoTruncate(numeric.NewDec(3))
	assert.Equal(t, "0.333333333333333333", result.String())

	// 2 / 3 truncated: 0.666666... truncated
	result2 := numeric.NewDec(2).QuoTruncate(numeric.NewDec(3))
	assert.Equal(t, "0.666666666666666666", result2.String())
}

func TestQuoRoundUp(t *testing.T) {
	// 2 / 3 rounded up should be 0.666...667
	result := numeric.NewDec(2).QuoRoundUp(numeric.NewDec(3))
	assert.Equal(t, "0.666666666666666667", result.String())

	// exact division should not add 1
	exact := numeric.NewDec(10).QuoRoundUp(numeric.NewDec(2))
	assert.Equal(t, "5.000000000000000000", exact.String())
}

func TestQuoInt(t *testing.T) {
	d := numeric.NewDec(10)
	result := d.QuoInt(big.NewInt(3))
	// QuoInt divides the internal big.Int directly by an integer
	// 10 * 10^18 / 3 = 3.333...333 * 10^18 (truncated to integer)
	expected := numeric.MustNewDecFromStr("3.333333333333333333")
	assert.True(t, result.Equal(expected), "got %s, want %s", result.String(), expected.String())
}

func TestQuoInt64(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		div  int64
		want string
	}{
		{"exact", numeric.NewDec(10), 2, "5.000000000000000000"},
		{"truncating", numeric.NewDec(10), 3, "3.333333333333333333"},
		{"by one", numeric.NewDec(7), 1, "7.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dec.QuoInt64(tt.div)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestNeg(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		want string
	}{
		{"negate positive", numeric.NewDec(5), "-5.000000000000000000"},
		{"negate negative", numeric.NewDec(-5), "5.000000000000000000"},
		{"negate zero", numeric.ZeroDec(), "0.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dec.Neg()
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestAbs(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		want string
	}{
		{"positive unchanged", numeric.NewDec(5), "5.000000000000000000"},
		{"negative becomes positive", numeric.NewDec(-5), "5.000000000000000000"},
		{"zero unchanged", numeric.ZeroDec(), "0.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dec.Abs()
			assert.Equal(t, tt.want, result.String())
		})
	}
}

// ---------------------------------------------------------------------------
// Rounding
// ---------------------------------------------------------------------------

func TestRoundInt64(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		want int64
	}{
		{"whole number", numeric.NewDec(5), 5},
		{"round down .4", numeric.MustNewDecFromStr("5.4"), 5},
		{"round up .6", numeric.MustNewDecFromStr("5.6"), 6},
		{"round half even (banker) .5 even", numeric.MustNewDecFromStr("4.5"), 4},
		{"round half even (banker) .5 odd", numeric.MustNewDecFromStr("5.5"), 6},
		{"zero", numeric.ZeroDec(), 0},
		{"negative round", numeric.MustNewDecFromStr("-3.7"), -4},
		{"negative .5 even", numeric.MustNewDecFromStr("-4.5"), -4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dec.RoundInt64())
		})
	}
}

func TestRoundInt(t *testing.T) {
	d := numeric.MustNewDecFromStr("7.8")
	result := d.RoundInt()
	assert.Equal(t, int64(8), result.Int64())

	d2 := numeric.MustNewDecFromStr("7.3")
	result2 := d2.RoundInt()
	assert.Equal(t, int64(7), result2.Int64())
}

func TestTruncateInt64(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		want int64
	}{
		{"whole number", numeric.NewDec(5), 5},
		{"truncate .9", numeric.MustNewDecFromStr("5.9"), 5},
		{"truncate .1", numeric.MustNewDecFromStr("5.1"), 5},
		{"zero", numeric.ZeroDec(), 0},
		{"negative truncate", numeric.MustNewDecFromStr("-3.7"), -3},
		{"negative truncate .1", numeric.MustNewDecFromStr("-3.1"), -3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dec.TruncateInt64())
		})
	}
}

func TestTruncateInt(t *testing.T) {
	d := numeric.MustNewDecFromStr("99.999")
	result := d.TruncateInt()
	assert.Equal(t, int64(99), result.Int64())
}

func TestTruncateDec(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		want string
	}{
		{"truncate fractional", numeric.MustNewDecFromStr("5.999"), "5.000000000000000000"},
		{"whole number unchanged", numeric.NewDec(10), "10.000000000000000000"},
		{"zero unchanged", numeric.ZeroDec(), "0.000000000000000000"},
		{"negative truncate", numeric.MustNewDecFromStr("-3.7"), "-3.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dec.TruncateDec()
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestCeil(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		want string
	}{
		{"positive fractional", numeric.MustNewDecFromStr("5.1"), "6.000000000000000000"},
		{"positive integer", numeric.NewDec(5), "5.000000000000000000"},
		{"zero", numeric.ZeroDec(), "0.000000000000000000"},
		{"negative fractional", numeric.MustNewDecFromStr("-3.7"), "-3.000000000000000000"},
		{"negative integer", numeric.NewDec(-3), "-3.000000000000000000"},
		{"just above zero", numeric.SmallestDec(), "1.000000000000000000"},
		{"large fractional", numeric.MustNewDecFromStr("99.001"), "100.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dec.Ceil()
			assert.Equal(t, tt.want, result.String())
		})
	}
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func TestString(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		want string
	}{
		{"zero", numeric.ZeroDec(), "0.000000000000000000"},
		{"one", numeric.OneDec(), "1.000000000000000000"},
		{"negative", numeric.NewDec(-42), "-42.000000000000000000"},
		{"large number", numeric.NewDec(123456789), "123456789.000000000000000000"},
		{"fractional", numeric.MustNewDecFromStr("3.14"), "3.140000000000000000"},
		{"smallest", numeric.SmallestDec(), "0.000000000000000001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dec.String())
		})
	}
}

func TestFormat(t *testing.T) {
	d := numeric.NewDec(42)
	assert.Equal(t, "42.000000000000000000", fmt.Sprintf("%v", d))
	assert.Equal(t, "42.000000000000000000", fmt.Sprintf("%s", d))
}

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
		want string
	}{
		{"zero", numeric.ZeroDec(), `"0.000000000000000000"`},
		{"one", numeric.OneDec(), `"1.000000000000000000"`},
		{"negative", numeric.NewDec(-5), `"-5.000000000000000000"`},
		{"fractional", numeric.MustNewDecFromStr("3.14"), `"3.140000000000000000"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.dec)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(data))
		})
	}
}

func TestMarshalJSON_NilInt(t *testing.T) {
	var d numeric.Dec
	data, err := d.MarshalJSON()
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"zero", `"0"`, "0.000000000000000000"},
		{"integer", `"42"`, "42.000000000000000000"},
		{"fractional", `"3.14"`, "3.140000000000000000"},
		{"negative", `"-7.5"`, "-7.500000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d numeric.Dec
			err := json.Unmarshal([]byte(tt.input), &d)
			require.NoError(t, err)
			assert.Equal(t, tt.want, d.String())
		})
	}
}

func TestUnmarshalJSON_Error(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not a string", `123`},
		{"bad decimal", `"abc"`},
		{"empty string", `""`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d numeric.Dec
			err := json.Unmarshal([]byte(tt.input), &d)
			assert.Error(t, err)
		})
	}
}

func TestJSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		dec  numeric.Dec
	}{
		{"zero", numeric.ZeroDec()},
		{"positive integer", numeric.NewDec(999)},
		{"negative integer", numeric.NewDec(-42)},
		{"fractional", numeric.MustNewDecFromStr("123.456789")},
		{"smallest", numeric.SmallestDec()},
		{"one", numeric.OneDec()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.dec)
			require.NoError(t, err)

			var restored numeric.Dec
			err = json.Unmarshal(data, &restored)
			require.NoError(t, err)

			assert.True(t, tt.dec.Equal(restored),
				"round trip failed: original=%s, restored=%s", tt.dec.String(), restored.String())
		})
	}
}

func TestMarshalYAML(t *testing.T) {
	d := numeric.NewDec(42)
	val, err := d.MarshalYAML()
	require.NoError(t, err)
	assert.Equal(t, "42.000000000000000000", val)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func TestDecsEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b []numeric.Dec
		want bool
	}{
		{"both empty", nil, nil, true},
		{"equal slices", []numeric.Dec{numeric.NewDec(1), numeric.NewDec(2)}, []numeric.Dec{numeric.NewDec(1), numeric.NewDec(2)}, true},
		{"different values", []numeric.Dec{numeric.NewDec(1)}, []numeric.Dec{numeric.NewDec(2)}, false},
		{"different lengths", []numeric.Dec{numeric.NewDec(1)}, []numeric.Dec{numeric.NewDec(1), numeric.NewDec(2)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, numeric.DecsEqual(tt.a, tt.b))
		})
	}
}

func TestMinDec(t *testing.T) {
	tests := []struct {
		name string
		a, b numeric.Dec
		want string
	}{
		{"first smaller", numeric.NewDec(1), numeric.NewDec(2), "1.000000000000000000"},
		{"second smaller", numeric.NewDec(5), numeric.NewDec(3), "3.000000000000000000"},
		{"equal", numeric.NewDec(4), numeric.NewDec(4), "4.000000000000000000"},
		{"negative values", numeric.NewDec(-10), numeric.NewDec(-5), "-10.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := numeric.MinDec(tt.a, tt.b)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestMaxDec(t *testing.T) {
	tests := []struct {
		name string
		a, b numeric.Dec
		want string
	}{
		{"first larger", numeric.NewDec(5), numeric.NewDec(3), "5.000000000000000000"},
		{"second larger", numeric.NewDec(1), numeric.NewDec(2), "2.000000000000000000"},
		{"equal", numeric.NewDec(4), numeric.NewDec(4), "4.000000000000000000"},
		{"negative values", numeric.NewDec(-10), numeric.NewDec(-5), "-5.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := numeric.MaxDec(tt.a, tt.b)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

func TestPow(t *testing.T) {
	tests := []struct {
		name string
		base numeric.Dec
		exp  int
		want string
	}{
		{"2^0", numeric.NewDec(2), 0, "1.000000000000000000"},
		{"2^1", numeric.NewDec(2), 1, "2.000000000000000000"},
		{"2^10", numeric.NewDec(2), 10, "1024.000000000000000000"},
		{"3^3", numeric.NewDec(3), 3, "27.000000000000000000"},
		{"10^6", numeric.NewDec(10), 6, "1000000.000000000000000000"},
		{"negative exp 2^-1", numeric.NewDec(2), -1, "0.500000000000000000"},
		{"negative exp 2^-2", numeric.NewDec(2), -2, "0.250000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := numeric.Pow(tt.base, tt.exp)
			assert.Equal(t, tt.want, result.String())
		})
	}
}

// ---------------------------------------------------------------------------
// NewDecFromString (scientific notation support)
// ---------------------------------------------------------------------------

func TestNewDecFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"plain integer", "42", "42.000000000000000000", false},
		{"decimal", "1.5", "1.500000000000000000", false},
		{"leading dot", ".5", "0.500000000000000000", false},
		{"scientific 1e2", "1e2", "100.000000000000000000", false},
		{"scientific 1.5e3", "1.5e3", "1500.000000000000000000", false},
		{"negative rejected", "-5", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := numeric.NewDecFromString(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, d.String())
		})
	}
}

// ---------------------------------------------------------------------------
// NewDecFromHex
// ---------------------------------------------------------------------------

func TestNewDecFromHex(t *testing.T) {
	tests := []struct {
		name string
		hex  string
		want string
	}{
		{"zero", "0", "0.000000000000000000"},
		{"one", "1", "1.000000000000000000"},
		{"0xff", "ff", "255.000000000000000000"},
		{"0xFF", "FF", "255.000000000000000000"},
		{"with prefix", "0xff", "255.000000000000000000"},
		{"0x100", "0x100", "256.000000000000000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := numeric.NewDecFromHex(tt.hex)
			assert.Equal(t, tt.want, d.String())
		})
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestArithmetic_Commutativity(t *testing.T) {
	a := numeric.MustNewDecFromStr("7.123456789012345678")
	b := numeric.MustNewDecFromStr("3.987654321098765432")

	assert.True(t, a.Add(b).Equal(b.Add(a)), "addition should be commutative")
	assert.True(t, a.Mul(b).Equal(b.Mul(a)), "multiplication should be commutative")
}

func TestArithmetic_AddSubInverse(t *testing.T) {
	a := numeric.MustNewDecFromStr("123.456")
	b := numeric.MustNewDecFromStr("78.9")

	result := a.Add(b).Sub(b)
	assert.True(t, a.Equal(result), "a + b - b should equal a, got %s", result.String())
}

func TestArithmetic_MulQuoInverse(t *testing.T) {
	a := numeric.NewDec(100)
	b := numeric.NewDec(7)

	// Due to precision limits, this may not be exact but should be very close
	result := a.Mul(b).Quo(b)
	diff := result.Sub(a).Abs()
	assert.True(t, diff.LT(numeric.MustNewDecFromStr("0.000000000000001")),
		"a * b / b should be approximately a, got %s", result.String())
}

func TestNeg_DoubleNeg(t *testing.T) {
	d := numeric.MustNewDecFromStr("42.5")
	assert.True(t, d.Equal(d.Neg().Neg()), "double negation should yield original")
}

func TestAbs_AlwaysPositive(t *testing.T) {
	neg := numeric.NewDec(-99)
	pos := numeric.NewDec(99)
	assert.True(t, neg.Abs().Equal(pos.Abs()))
	assert.True(t, neg.Abs().Equal(pos))
}

func TestVeryLargeNumbers(t *testing.T) {
	// Create a very large Dec from a big.Int
	large := new(big.Int).Exp(big.NewInt(10), big.NewInt(50), nil)
	d := numeric.NewDecFromBigInt(large)
	assert.True(t, d.IsPositive())
	assert.False(t, d.IsZero())

	// Verify the value can be added to itself without loss
	doubled := d.Add(d)
	assert.True(t, doubled.GT(d))
}

func TestPrecisionBoundary(t *testing.T) {
	// Full 18 decimal places
	d, err := numeric.NewDecFromStr("1.123456789012345678")
	require.NoError(t, err)
	assert.Equal(t, "1.123456789012345678", d.String())

	// One too many decimal places should error
	_, err = numeric.NewDecFromStr("1.1234567890123456789")
	assert.Error(t, err)
}

func TestNewDecFromStr_MaxPrecision(t *testing.T) {
	// 18 zeros after decimal point followed by nothing should work with just 0
	d, err := numeric.NewDecFromStr("0.000000000000000001")
	require.NoError(t, err)
	assert.True(t, d.Equal(numeric.SmallestDec()))
}

func TestQuoInt64_ByZeroPanics(t *testing.T) {
	assert.Panics(t, func() {
		numeric.NewDec(1).QuoInt64(0)
	})
}

func TestQuoInt_ByZeroPanics(t *testing.T) {
	assert.Panics(t, func() {
		numeric.NewDec(1).QuoInt(big.NewInt(0))
	})
}

func TestRoundInt64_OverflowPanics(t *testing.T) {
	// A number too large to fit in int64 should panic
	huge := new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	d := numeric.NewDecFromBigInt(huge)
	assert.Panics(t, func() {
		d.RoundInt64()
	})
}

func TestTruncateInt64_OverflowPanics(t *testing.T) {
	huge := new(big.Int).Exp(big.NewInt(10), big.NewInt(30), nil)
	d := numeric.NewDecFromBigInt(huge)
	assert.Panics(t, func() {
		d.TruncateInt64()
	})
}

func TestBankersRounding(t *testing.T) {
	// Bankers rounding: round half to even
	// 0.5 -> 0 (even), 1.5 -> 2 (even), 2.5 -> 2 (even), 3.5 -> 4 (even)
	tests := []struct {
		name string
		dec  numeric.Dec
		want int64
	}{
		{"0.5 -> 0", numeric.MustNewDecFromStr("0.5"), 0},
		{"1.5 -> 2", numeric.MustNewDecFromStr("1.5"), 2},
		{"2.5 -> 2", numeric.MustNewDecFromStr("2.5"), 2},
		{"3.5 -> 4", numeric.MustNewDecFromStr("3.5"), 4},
		{"4.5 -> 4", numeric.MustNewDecFromStr("4.5"), 4},
		{"5.5 -> 6", numeric.MustNewDecFromStr("5.5"), 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.dec.RoundInt64())
		})
	}
}
