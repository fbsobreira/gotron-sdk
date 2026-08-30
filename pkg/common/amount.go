package common

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// MaxAmountDecimals is the largest scale ParseAmount accepts. TRC20
// decimals() is a uint8 in practice and int64 overflows at 10^19, so 18
// matches both the typical token limit and numeric.Precision.
const MaxAmountDecimals = 18

// ParseAmount converts a decimal amount string into an integer number of
// base units by scaling with 10^decimals. Parsing is exact: the input is
// never converted through float64.
//
// Zero is allowed; callers that require a strictly positive amount must
// check the result. Negative amounts, non-finite values (NaN/Inf), empty
// or non-decimal input, more fractional digits than decimals, values that
// overflow int64 after scaling, and a negative or too-large decimals
// count are rejected.
//
// Accepted forms: plain integers, optional leading "+", a leading or
// trailing decimal point (".5", "5."), and surrounding whitespace.
// Trailing zeros in the fractional part do not count toward the decimals
// limit ("1.1000000" is valid for 6 decimals). Scientific notation is
// rejected.
func ParseAmount(amount string, decimals int) (int64, error) {
	if decimals < 0 {
		return 0, fmt.Errorf("decimals must be non-negative, got %d", decimals)
	}
	if decimals > MaxAmountDecimals {
		return 0, fmt.Errorf("decimals %d exceeds maximum %d", decimals, MaxAmountDecimals)
	}

	orig := amount
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0, fmt.Errorf("%q is empty", orig)
	}
	if isNonFiniteAmount(amount) {
		return 0, fmt.Errorf("%q is not a finite number", orig)
	}
	if amount[0] == '-' {
		return 0, fmt.Errorf("%q is negative", orig)
	}
	if amount[0] == '+' {
		amount = amount[1:]
		if amount == "" {
			return 0, fmt.Errorf("invalid amount %q", orig)
		}
		if isNonFiniteAmount(amount) {
			return 0, fmt.Errorf("%q is not a finite number", orig)
		}
	}

	whole, frac, ok := splitDecimalAmount(amount)
	if !ok {
		return 0, fmt.Errorf("invalid amount %q", orig)
	}
	frac = strings.TrimRight(frac, "0")
	if len(frac) > decimals {
		return 0, fmt.Errorf("%q has more than %d decimal places", orig, decimals)
	}

	ten := big.NewInt(10)
	scale := new(big.Int).Exp(ten, big.NewInt(int64(decimals)), nil)

	n := new(big.Int)
	if whole == "" {
		n.SetInt64(0)
	} else if _, ok := n.SetString(whole, 10); !ok {
		return 0, fmt.Errorf("invalid amount %q", orig)
	}
	n.Mul(n, scale)

	if frac != "" {
		f := new(big.Int)
		if _, ok := f.SetString(frac, 10); !ok {
			return 0, fmt.Errorf("invalid amount %q", orig)
		}
		pad := decimals - len(frac)
		if pad > 0 {
			f.Mul(f, new(big.Int).Exp(ten, big.NewInt(int64(pad)), nil))
		}
		n.Add(n, f)
	}

	if !n.IsInt64() {
		return 0, fmt.Errorf("%q overflows int64 after scaling by 10^%d", orig, decimals)
	}
	return n.Int64(), nil
}

func isNonFiniteAmount(s string) bool {
	switch strings.ToLower(s) {
	case "nan", "+nan", "-nan", "inf", "+inf", "-inf", "infinity", "+infinity", "-infinity":
		return true
	default:
		return false
	}
}

func splitDecimalAmount(s string) (whole, frac string, ok bool) {
	if strings.ContainsAny(s, "eE") {
		return "", "", false
	}
	dot := strings.IndexByte(s, '.')
	if dot < 0 {
		if !isAllDigits(s) {
			return "", "", false
		}
		return s, "", true
	}
	if strings.IndexByte(s[dot+1:], '.') >= 0 {
		return "", "", false
	}
	whole, frac = s[:dot], s[dot+1:]
	if whole == "" && frac == "" {
		return "", "", false
	}
	if whole != "" && !isAllDigits(whole) {
		return "", "", false
	}
	if frac != "" && !isAllDigits(frac) {
		return "", "", false
	}
	return whole, frac, true
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// FormatAmount renders an integer number of base units as a canonical
// decimal string at the given scale. It is the inverse of ParseAmount:
// ParseAmount(FormatAmount(v, d), d) returns v.
//
// Trailing fractional zeros are trimmed and the result is always a valid
// JSON number literal — never ".5" or "5." — so callers can emit it as a
// json.Number and keep numeric JSON output.
func FormatAmount(units int64, decimals int) string {
	if decimals <= 0 {
		return strconv.FormatInt(units, 10)
	}

	n := new(big.Int).SetInt64(units)
	neg := n.Sign() < 0
	n.Abs(n)

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole, rem := new(big.Int), new(big.Int)
	whole.QuoRem(n, scale, rem)

	out := whole.String()
	if rem.Sign() != 0 {
		frac := rem.String()
		if pad := decimals - len(frac); pad > 0 {
			frac = strings.Repeat("0", pad) + frac
		}
		out += "." + strings.TrimRight(frac, "0")
	}
	if neg {
		out = "-" + out
	}
	return out
}
