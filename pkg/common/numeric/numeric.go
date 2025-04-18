package numeric

// Incorporated from cosmos-sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

// Dec represent a decimal. NOTE: never use new(Dec) or else we will panic unmarshalling into the
// nil embedded big.Int
type Dec struct {
	*big.Int `json:"int"`
}

// number of decimal places
const (
	Precision = 18

	// bytes required to represent the above precision
	// Ceiling[Log2[999 999 999 999 999 999]]
	DecimalPrecisionBits = 60
)

var (
	precisionReuse       = new(big.Int).Exp(big.NewInt(10), big.NewInt(Precision), nil)
	fivePrecision        = new(big.Int).Quo(precisionReuse, big.NewInt(2))
	precisionMultipliers []*big.Int
	zeroInt              = big.NewInt(0)
	oneInt               = big.NewInt(1)
	tenInt               = big.NewInt(10)
)

// Set precision multipliers
func init() {
	precisionMultipliers = make([]*big.Int, Precision+1)
	for i := 0; i <= Precision; i++ {
		precisionMultipliers[i] = calcPrecisionMultiplier(int64(i))
	}
}

func precisionInt() *big.Int {
	return new(big.Int).Set(precisionReuse)
}

// ZeroDec ...
func ZeroDec() Dec { return Dec{new(big.Int).Set(zeroInt)} }

// OneDec ...
func OneDec() Dec { return Dec{precisionInt()} }

// SmallestDec ...
func SmallestDec() Dec { return Dec{new(big.Int).Set(oneInt)} }

// calculate the precision multiplier
func calcPrecisionMultiplier(prec int64) *big.Int {
	if prec > Precision {
		panic(fmt.Sprintf("too much precision, maximum %v, provided %v", Precision, prec))
	}
	zerosToAdd := Precision - prec
	multiplier := new(big.Int).Exp(tenInt, big.NewInt(zerosToAdd), nil)
	return multiplier
}

// get the precision multiplier, do not mutate result
func precisionMultiplier(prec int64) *big.Int {
	if prec > Precision {
		panic(fmt.Sprintf("too much precision, maximum %v, provided %v", Precision, prec))
	}
	return precisionMultipliers[prec]
}

//______________________________________________________________________________________________

// NewDec creates a new Dec from integer assuming whole number
func NewDec(i int64) Dec {
	return NewDecWithPrec(i, 0)
}

// NewDecWithPrec creates a new Dec from integer with decimal place at prec
// CONTRACT: prec <= Precision
func NewDecWithPrec(i, prec int64) Dec {
	return Dec{
		new(big.Int).Mul(big.NewInt(i), precisionMultiplier(prec)),
	}
}

// NewDecFromBigInt creates a new Dec from big integer assuming whole numbers
// CONTRACT: prec <= Precision
func NewDecFromBigInt(i *big.Int) Dec {
	return NewDecFromBigIntWithPrec(i, 0)
}

// NewDecFromBigIntWithPrec creates a new Dec from big integer assuming whole numbers
// CONTRACT: prec <= Precision
func NewDecFromBigIntWithPrec(i *big.Int, prec int64) Dec {
	return Dec{
		new(big.Int).Mul(i, precisionMultiplier(prec)),
	}
}

// NewDecFromInt creates a new Dec from big integer assuming whole numbers
// CONTRACT: prec <= Precision
func NewDecFromInt(i *big.Int) Dec {
	return NewDecFromIntWithPrec(i, 0)
}

// NewDecFromIntWithPrec creates a new Dec from big integer with decimal place at prec
// CONTRACT: prec <= Precision
func NewDecFromIntWithPrec(i *big.Int, prec int64) Dec {
	return Dec{
		new(big.Int).Mul(i, precisionMultiplier(prec)),
	}
}

// NewDecFromStr creates a decimal from an input decimal string.
// valid must come in the form:
//
//	(-) whole integers (.) decimal integers
//
// examples of acceptable input include:
//
//	-123.456
//	456.7890
//	345
//	-456789
//
// NOTE - An error will return if more decimal places
// are provided in the string than the constant Precision.
//
// CONTRACT - This function does not mutate the input str.
func NewDecFromStr(str string) (d Dec, err error) {
	if len(str) == 0 {
		return d, errors.New("decimal string is empty")
	}

	// first extract any negative symbol
	neg := false
	if str[0] == '-' {
		neg = true
		str = str[1:]
	}

	if len(str) == 0 {
		return d, errors.New("decimal string is empty")
	}

	strs := strings.Split(str, ".")
	lenDecs := 0
	combinedStr := strs[0]

	if len(strs) == 2 { // has a decimal place
		lenDecs = len(strs[1])
		if lenDecs == 0 || len(combinedStr) == 0 {
			return d, errors.New("bad decimal length")
		}
		combinedStr += strs[1]

	} else if len(strs) > 2 {
		return d, errors.New("too many periods to be a decimal string")
	}

	if lenDecs > Precision {
		return d, fmt.Errorf("too much precision, maximum %v, len decimal %v", Precision, lenDecs)
	}

	// add some extra zero's to correct to the Precision factor
	zerosToAdd := Precision - lenDecs
	zeros := fmt.Sprintf(`%0`+strconv.Itoa(zerosToAdd)+`s`, "")
	combinedStr += zeros

	combined, ok := new(big.Int).SetString(combinedStr, 10) // base 10
	if !ok {
		return d, fmt.Errorf("bad string to integer conversion, combinedStr: %v", combinedStr)
	}
	if neg {
		combined = new(big.Int).Neg(combined)
	}
	return Dec{combined}, nil
}

// MustNewDecFromStr Decimal from string, panic on error
func MustNewDecFromStr(s string) Dec {
	dec, err := NewDecFromStr(s)
	if err != nil {
		panic(err)
	}
	return dec
}

// IsNil ...
func (d Dec) IsNil() bool { return d.Int == nil } // is decimal nil
// IsZero ...
func (d Dec) IsZero() bool { return (d.Int).Sign() == 0 } // is equal to zero
// IsNegative ...
func (d Dec) IsNegative() bool { return (d.Int).Sign() == -1 } // is negative
// IsPositive ...
func (d Dec) IsPositive() bool { return (d.Int).Sign() == 1 } // is positive
// Equal ...
func (d Dec) Equal(d2 Dec) bool { return (d.Int).Cmp(d2.Int) == 0 } // equal decimals
// GT ...
func (d Dec) GT(d2 Dec) bool { return (d.Int).Cmp(d2.Int) > 0 } // greater than
// GTE ...
func (d Dec) GTE(d2 Dec) bool { return (d.Int).Cmp(d2.Int) >= 0 } // greater than or equal
// LT ...
func (d Dec) LT(d2 Dec) bool { return (d.Int).Cmp(d2.Int) < 0 } // less than
// LTE ...
func (d Dec) LTE(d2 Dec) bool { return (d.Int).Cmp(d2.Int) <= 0 } // less than or equal
// Neg ...
func (d Dec) Neg() Dec { return Dec{new(big.Int).Neg(d.Int)} } // reverse the decimal sign
// Abs ...
func (d Dec) Abs() Dec { return Dec{new(big.Int).Abs(d.Int)} } // absolute value

// Add addition
func (d Dec) Add(d2 Dec) Dec {
	res := new(big.Int).Add(d.Int, d2.Int)

	if res.BitLen() > 255+DecimalPrecisionBits {
		panic("Int overflow")
	}
	return Dec{res}
}

// Sub subtraction
func (d Dec) Sub(d2 Dec) Dec {
	res := new(big.Int).Sub(d.Int, d2.Int)

	if res.BitLen() > 255+DecimalPrecisionBits {
		panic("Int overflow")
	}
	return Dec{res}
}

// Mul multiplication
func (d Dec) Mul(d2 Dec) Dec {
	mul := new(big.Int).Mul(d.Int, d2.Int)
	chopped := chopPrecisionAndRound(mul)

	if chopped.BitLen() > 255+DecimalPrecisionBits {
		panic("Int overflow")
	}
	return Dec{chopped}
}

// MulTruncate multiplication truncate
func (d Dec) MulTruncate(d2 Dec) Dec {
	mul := new(big.Int).Mul(d.Int, d2.Int)
	chopped := chopPrecisionAndTruncate(mul)

	if chopped.BitLen() > 255+DecimalPrecisionBits {
		panic("Int overflow")
	}
	return Dec{chopped}
}

// MulInt multiplication
func (d Dec) MulInt(i *big.Int) Dec {
	mul := new(big.Int).Mul(d.Int, i)

	if mul.BitLen() > 255+DecimalPrecisionBits {
		panic("Int overflow")
	}
	return Dec{mul}
}

// MulInt64 - multiplication with int64
func (d Dec) MulInt64(i int64) Dec {
	mul := new(big.Int).Mul(d.Int, big.NewInt(i))

	if mul.BitLen() > 255+DecimalPrecisionBits {
		panic("Int overflow")
	}
	return Dec{mul}
}

// Quo quotient
func (d Dec) Quo(d2 Dec) Dec {

	// multiply precision twice
	mul := new(big.Int).Mul(d.Int, precisionReuse)
	mul.Mul(mul, precisionReuse)

	quo := new(big.Int).Quo(mul, d2.Int)
	chopped := chopPrecisionAndRound(quo)

	if chopped.BitLen() > 255+DecimalPrecisionBits {
		panic("Int overflow")
	}
	return Dec{chopped}
}

// QuoTruncate quotient truncate
func (d Dec) QuoTruncate(d2 Dec) Dec {

	// multiply precision twice
	mul := new(big.Int).Mul(d.Int, precisionReuse)
	mul.Mul(mul, precisionReuse)

	quo := new(big.Int).Quo(mul, d2.Int)
	chopped := chopPrecisionAndTruncate(quo)

	if chopped.BitLen() > 255+DecimalPrecisionBits {
		panic("Int overflow")
	}
	return Dec{chopped}
}

// QuoRoundUp quotient, round up
func (d Dec) QuoRoundUp(d2 Dec) Dec {
	// multiply precision twice
	mul := new(big.Int).Mul(d.Int, precisionReuse)
	mul.Mul(mul, precisionReuse)

	quo := new(big.Int).Quo(mul, d2.Int)
	chopped := chopPrecisionAndRoundUp(quo)

	if chopped.BitLen() > 255+DecimalPrecisionBits {
		panic("Int overflow")
	}
	return Dec{chopped}
}

// QuoInt quotient
func (d Dec) QuoInt(i *big.Int) Dec {
	mul := new(big.Int).Quo(d.Int, i)
	return Dec{mul}
}

// QuoInt64 - quotient with int64
func (d Dec) QuoInt64(i int64) Dec {
	mul := new(big.Int).Quo(d.Int, big.NewInt(i))
	return Dec{mul}
}

// IsInteger is integer, e.g. decimals are zero
func (d Dec) IsInteger() bool {
	return new(big.Int).Rem(d.Int, precisionReuse).Sign() == 0
}

// Format decimal state
func (d Dec) Format(s fmt.State, verb rune) {
	_, err := s.Write([]byte(d.String()))
	if err != nil {
		panic(err)
	}
}

func (d Dec) String() string {
	if d.Int == nil {
		return d.Int.String()
	}

	isNeg := d.IsNegative()
	if d.IsNegative() {
		d = d.Neg()
	}

	bzInt, err := d.Int.MarshalText()
	if err != nil {
		return ""
	}
	inputSize := len(bzInt)

	var bzStr []byte

	// TODO: Remove trailing zeros
	// case 1, purely decimal
	if inputSize <= Precision {
		bzStr = make([]byte, Precision+2)

		// 0. prefix
		bzStr[0] = byte('0')
		bzStr[1] = byte('.')

		// set relevant digits to 0
		for i := 0; i < Precision-inputSize; i++ {
			bzStr[i+2] = byte('0')
		}

		// set final digits
		copy(bzStr[2+(Precision-inputSize):], bzInt)

	} else {

		// inputSize + 1 to account for the decimal point that is being added
		bzStr = make([]byte, inputSize+1)
		decPointPlace := inputSize - Precision

		copy(bzStr, bzInt[:decPointPlace])                   // pre-decimal digits
		bzStr[decPointPlace] = byte('.')                     // decimal point
		copy(bzStr[decPointPlace+1:], bzInt[decPointPlace:]) // post-decimal digits
	}

	if isNeg {
		return "-" + string(bzStr)
	}

	return string(bzStr)
}

//     ____
//  __|    |__   "chop 'em
//       ` \     round!"
// ___||  ~  _     -bankers
// |         |      __
// |       | |   __|__|__
// |_____:  /   | $$$    |
//              |________|

// nolint - go-cyclo
// Remove a Precision amount of rightmost digits and perform bankers rounding
// on the remainder (gaussian rounding) on the digits which have been removed.
//
// Mutates the input. Use the non-mutative version if that is undesired
func chopPrecisionAndRound(d *big.Int) *big.Int {

	// remove the negative and add it back when returning
	if d.Sign() == -1 {
		// make d positive, compute chopped value, and then un-mutate d
		d = d.Neg(d)
		d = chopPrecisionAndRound(d)
		d = d.Neg(d)
		return d
	}

	// get the truncated quotient and remainder
	quo, rem := d, big.NewInt(0)
	quo, rem = quo.QuoRem(d, precisionReuse, rem)

	if rem.Sign() == 0 { // remainder is zero
		return quo
	}

	switch rem.Cmp(fivePrecision) {
	case -1:
		return quo
	case 1:
		return quo.Add(quo, oneInt)
	default: // bankers rounding must take place
		// always round to an even number
		if quo.Bit(0) == 0 {
			return quo
		}
		return quo.Add(quo, oneInt)
	}
}

func chopPrecisionAndRoundUp(d *big.Int) *big.Int {

	// remove the negative and add it back when returning
	if d.Sign() == -1 {
		// make d positive, compute chopped value, and then un-mutate d
		d = d.Neg(d)
		// truncate since d is negative...
		d = chopPrecisionAndTruncate(d)
		d = d.Neg(d)
		return d
	}

	// get the truncated quotient and remainder
	quo, rem := d, big.NewInt(0)
	quo, rem = quo.QuoRem(d, precisionReuse, rem)

	if rem.Sign() == 0 { // remainder is zero
		return quo
	}

	return quo.Add(quo, oneInt)
}

func chopPrecisionAndRoundNonMutative(d *big.Int) *big.Int {
	tmp := new(big.Int).Set(d)
	return chopPrecisionAndRound(tmp)
}

// RoundInt64 rounds the decimal using bankers rounding
func (d Dec) RoundInt64() int64 {
	chopped := chopPrecisionAndRoundNonMutative(d.Int)
	if !chopped.IsInt64() {
		panic("Int64() out of bound")
	}
	return chopped.Int64()
}

// RoundInt round the decimal using bankers rounding
func (d Dec) RoundInt() *big.Int {
	return chopPrecisionAndRoundNonMutative(d.Int)
}

//___________________________________________________________________________________

// similar to chopPrecisionAndRound, but always rounds down
func chopPrecisionAndTruncate(d *big.Int) *big.Int {
	return d.Quo(d, precisionReuse)
}

func chopPrecisionAndTruncateNonMutative(d *big.Int) *big.Int {
	tmp := new(big.Int).Set(d)
	return chopPrecisionAndTruncate(tmp)
}

// TruncateInt64 truncates the decimals from the number and returns an int64
func (d Dec) TruncateInt64() int64 {
	chopped := chopPrecisionAndTruncateNonMutative(d.Int)
	if !chopped.IsInt64() {
		panic("Int64() out of bound")
	}
	return chopped.Int64()
}

// TruncateInt truncates the decimals from the number and returns an Int
func (d Dec) TruncateInt() *big.Int {
	return chopPrecisionAndTruncateNonMutative(d.Int)
}

// TruncateDec truncates the decimals from the number and returns a Dec
func (d Dec) TruncateDec() Dec {
	return NewDecFromBigInt(chopPrecisionAndTruncateNonMutative(d.Int))
}

// Ceil returns the smallest interger value (as a decimal) that is greater than
// or equal to the given decimal.
func (d Dec) Ceil() Dec {
	tmp := new(big.Int).Set(d.Int)

	quo, rem := tmp, big.NewInt(0)
	quo, rem = quo.QuoRem(tmp, precisionReuse, rem)

	// no need to round with a zero remainder regardless of sign
	if rem.Cmp(zeroInt) == 0 {
		return NewDecFromBigInt(quo)
	}

	if rem.Sign() == -1 {
		return NewDecFromBigInt(quo)
	}

	return NewDecFromBigInt(quo.Add(quo, oneInt))
}

//___________________________________________________________________________________

// MarshalJSON marshals the decimal
func (d Dec) MarshalJSON() ([]byte, error) {
	if d.Int == nil {
		return []byte{}, nil
	}

	return json.Marshal(d.String())
}

// UnmarshalJSON defines custom decoding scheme
func (d *Dec) UnmarshalJSON(bz []byte) error {
	if d.Int == nil {
		d.Int = new(big.Int)
	}

	var text string
	err := json.Unmarshal(bz, &text)
	if err != nil {
		return err
	}
	// TODO: Reuse dec allocation
	newDec, err := NewDecFromStr(text)
	if err != nil {
		return err
	}
	d.Int = newDec.Int
	return nil
}

// MarshalYAML returns Ythe AML representation.
func (d Dec) MarshalYAML() (interface{}, error) { return d.String(), nil }

//___________________________________________________________________________________
// helpers

// DecsEqual test if two decimal arrays are equal
func DecsEqual(d1s, d2s []Dec) bool {
	if len(d1s) != len(d2s) {
		return false
	}

	for i, d1 := range d1s {
		if !d1.Equal(d2s[i]) {
			return false
		}
	}
	return true
}

// MinDec minimum decimal between two
func MinDec(d1, d2 Dec) Dec {
	if d1.LT(d2) {
		return d1
	}
	return d2
}

// MaxDec maximum decimal between two
func MaxDec(d1, d2 Dec) Dec {
	if d1.LT(d2) {
		return d2
	}
	return d1
}

var (
	pattern, _ = regexp.Compile(`[0-9]+\.{0,1}[0-9]*e-{0,1}[0-9]+`)
)

// Pow calcs power of numeric with int
func Pow(base Dec, exp int) Dec {
	if exp < 0 {
		return Pow(NewDec(1).Quo(base), -exp)
	}
	result := NewDec(1)
	for {
		if exp%2 == 1 {
			result = result.Mul(base)
		}
		exp = exp >> 1
		if exp == 0 {
			break
		}
		base = base.Mul(base)
	}
	return result
}

// NewDecFromString from string to DEC
func NewDecFromString(i string) (Dec, error) {
	if strings.HasPrefix(i, "-") {
		return ZeroDec(), fmt.Errorf("can not be negative: %s", i)
	}
	if pattern.FindString(i) != "" {
		tokens := strings.Split(i, "e")
		a, _ := NewDecFromStr(tokens[0])
		b, _ := strconv.Atoi(tokens[1])
		return a.Mul(Pow(NewDec(10), b)), nil
	}
	if strings.HasPrefix(i, ".") {
		i = "0" + i
	}
	return NewDecFromStr(i)

}

// NewDecFromHex Assumes Hex string input
// Split into 2 64 bit integers to guarentee 128 bit precision
func NewDecFromHex(str string) Dec {
	str = strings.TrimPrefix(str, "0x")
	half := len(str) / 2
	right := str[half:]
	r, _ := big.NewInt(0).SetString(right, 16)
	if half == 0 {
		return NewDecFromBigInt(r)
	}
	left := str[:half]
	l, _ := big.NewInt(0).SetString(left, 16)
	return NewDecFromBigInt(l).Mul(
		Pow(NewDec(16), len(right)),
	).Add(NewDecFromBigInt(r))
}
