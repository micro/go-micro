package nano

import (
	"errors"
	"math/big"
	"strings"
)

// RawDecimals is the number of decimal places between one XNO and one raw:
// 1 XNO = 10^30 raw.
const RawDecimals = 30

// ErrInvalidAmount is returned for an amount that is not a positive decimal
// with at most RawDecimals fractional digits.
var ErrInvalidAmount = errors.New("nano: invalid XNO amount")

// RawFromXNO converts a decimal XNO string such as "0.001" to raw using
// integer arithmetic only. Floats, signs, exponents and more than 30
// fractional digits are refused.
func RawFromXNO(xno string) (*big.Int, error) {
	s := strings.TrimSpace(xno)
	if s == "" {
		return nil, ErrInvalidAmount
	}
	whole, frac, hasDot := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	if hasDot && frac == "" {
		return nil, ErrInvalidAmount
	}
	if len(frac) > RawDecimals || !allDigits(whole) || !allDigits(frac) {
		return nil, ErrInvalidAmount
	}
	digits := whole + frac + strings.Repeat("0", RawDecimals-len(frac))
	raw, ok := new(big.Int).SetString(digits, 10)
	if !ok || raw.Sign() <= 0 {
		return nil, ErrInvalidAmount
	}
	return raw, nil
}

// ParseRaw parses a raw amount as a node returns it (a base-10 integer
// string). It refuses anything that is not a non-negative integer.
func ParseRaw(raw string) (*big.Int, error) {
	if raw == "" || !allDigits(raw) {
		return nil, ErrInvalidAmount
	}
	v, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, ErrInvalidAmount
	}
	return v, nil
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
