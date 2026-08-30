package cmd

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
)

const maxIssueRatioDecimals = 6

func parseAmountArg(raw, name string, decimals int) (int64, error) {
	v, err := common.ParseAmount(raw, decimals)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func parseTRXArg(raw, name string) (int64, error) {
	return parseAmountArg(raw, name, common.AmountDecimalPoint)
}

// isDecimalZero reports whether raw is a decimal zero at any scale ("0",
// "0.0", "+0.000"). Flags that used to be float64 tested `== 0` to skip
// optional work; now that they are strings, comparing against the literal
// "0" would treat "0.0" as non-zero and change behaviour.
func isDecimalZero(raw string) bool {
	v, err := common.ParseAmount(raw, 0)
	return err == nil && v == 0
}

// parseIssueRatio parses a TRC10 issue TRX:token ratio. "a:b" is two
// integers. A plain decimal ("1.5") is converted to an equivalent integer
// ratio with at most 6 fractional digits ("1.5" → 15:10). This is a ratio,
// not a monetary amount, but it is parsed exactly rather than through
// float32 so the stored numerator/denominator are not silently rounded.
func parseIssueRatio(raw string) (trxNum, tokenNum int32, err error) {
	if colon := strings.Index(raw, ":"); colon >= 0 {
		trx, err := strconv.ParseInt(raw[:colon], 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("RATIO trx numerator: %w", err)
		}
		token, err := strconv.ParseInt(raw[colon+1:], 10, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("RATIO token denominator: %w", err)
		}
		if trx < 0 || token < 0 {
			return 0, 0, fmt.Errorf("RATIO %q is negative", raw)
		}
		return int32(trx), int32(token), nil
	}

	var lastErr error
	for d := 0; d <= maxIssueRatioDecimals; d++ {
		v, err := common.ParseAmount(raw, d)
		if err != nil {
			lastErr = err
			continue
		}
		scale := int64(1)
		for i := 0; i < d; i++ {
			scale *= 10
		}
		if v != int64(int32(v)) || scale != int64(int32(scale)) {
			return 0, 0, fmt.Errorf("RATIO %q overflows int32", raw)
		}
		return int32(v), int32(scale), nil
	}
	if lastErr != nil {
		return 0, 0, fmt.Errorf("RATIO: %w", lastErr)
	}
	return 0, 0, fmt.Errorf("invalid ratio %q", raw)
}

// roundBancorQuote returns the nearest integer of
// input * reserveOut / (reserveIn + input), matching the previous
// math.Floor(x + 0.5) rounding for non-negative values.
func roundBancorQuote(input, reserveIn, reserveOut int64) (int64, error) {
	den := new(big.Int).Add(big.NewInt(reserveIn), big.NewInt(input))
	if den.Sign() <= 0 {
		return 0, fmt.Errorf("invalid exchange reserve")
	}
	num := new(big.Int).Mul(big.NewInt(input), big.NewInt(reserveOut))
	adj := new(big.Int).Quo(new(big.Int).Set(den), big.NewInt(2))
	num.Add(num, adj)
	num.Quo(num, den)
	if !num.IsInt64() {
		return 0, fmt.Errorf("quoted amount overflows int64")
	}
	return num.Int64(), nil
}
