// Package decimals provides decimal formatting and conversion utilities.
package decimals

import (
	"math"
	"math/big"
)

// Pow returns a raised to the power e. Negative exponents are supported.
func Pow(a *big.Float, e int64) *big.Float {
	if e < 0 {
		if e == math.MinInt64 {
			return Div(NewFloat(1), Mul(a, Pow(a, -(e+1))))
		}
		return Div(NewFloat(1), Pow(a, -e))
	}
	result := NewFloat(1)
	for range e {
		result = Mul(result, a)
	}
	return result
}

// Root returns the n-th root of a using Newton's method with 256-bit precision.
func Root(a *big.Float, n uint64) *big.Float {
	limit := Pow(NewFloat(2), 256)
	n1 := n - 1
	n1f, rn := NewFloat(float64(n1)), Div(NewFloat(1.0), NewFloat(float64(n)))
	x, x0 := NewFloat(1.0), Zero()
	_ = x0
	for {
		potx, t2 := Div(NewFloat(1.0), x), a
		for b := n1; b > 0; b >>= 1 {
			if b&1 == 1 {
				t2 = Mul(t2, potx)
			}
			potx = Mul(potx, potx)
		}
		x0, x = x, Mul(rn, Add(Mul(n1f, x), t2))
		if Lesser(Mul(Abs(Sub(x, x0)), limit), x) {
			break
		}
	}
	return x
}

// Abs returns the absolute value of a.
func Abs(a *big.Float) *big.Float {
	return Zero().Abs(a)
}

// NewFloat creates a big.Float from f with 256-bit precision.
func NewFloat(f float64) *big.Float {
	r := big.NewFloat(f)
	r.SetPrec(256)
	return r
}

// Div returns a / b with 256-bit precision.
func Div(a, b *big.Float) *big.Float {
	return Zero().Quo(a, b)
}

// Zero returns a new big.Float zero with 256-bit precision.
func Zero() *big.Float {
	r := big.NewFloat(0.0)
	r.SetPrec(256)
	return r
}

// Mul returns a * b with 256-bit precision.
func Mul(a, b *big.Float) *big.Float {
	return Zero().Mul(a, b)
}

// Add returns a + b with 256-bit precision.
func Add(a, b *big.Float) *big.Float {
	return Zero().Add(a, b)
}

// Sub returns a - b with 256-bit precision.
func Sub(a, b *big.Float) *big.Float {
	return Zero().Sub(a, b)
}

// Lesser reports whether x is strictly less than y.
func Lesser(x, y *big.Float) bool {
	return x.Cmp(y) == -1
}

// FromString parses x as a decimal number into a big.Float with 256-bit precision.
func FromString(x string) (*big.Float, bool) {
	return Zero().SetString(x)
}

// ApplyDecimals multiplies x by 10^y and returns the result as a big.Int.
func ApplyDecimals(x *big.Float, y int64) (*big.Int, big.Accuracy) {
	return Mul(x, Pow(NewFloat(10), y)).Int(nil)
}

// RemoveDecimals divides x by 10^y and returns the result as a big.Float.
func RemoveDecimals(x *big.Int, y int64) *big.Float {
	return Div(new(big.Float).SetInt(x), Pow(NewFloat(10), y))
}
