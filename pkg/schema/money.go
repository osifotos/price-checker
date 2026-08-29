package schema

import (
	"fmt"
	"math/big"
	"strings"
)

// StringToRat parses a decimal string such as "0.0000041667" or "12.48" into an
// exact *big.Rat. It rejects empty input, scientific notation, and the special
// float values. A leading "-" is accepted.
func StringToRat(s string) (*big.Rat, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("schema: empty decimal string")
	}
	if strings.ContainsAny(s, "eE") {
		return nil, fmt.Errorf("schema: scientific notation not allowed: %q", s)
	}
	body := strings.TrimPrefix(s, "-")
	if body == "" || strings.Trim(body, "0123456789.") != "" || strings.Count(body, ".") > 1 {
		return nil, fmt.Errorf("schema: not a decimal string: %q", s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("schema: cannot parse decimal string: %q", s)
	}
	return r, nil
}

// RatToString formats r as a plain decimal string with exactly dp digits after
// the decimal point, rounding half away from zero. dp must be >= 0.
func RatToString(r *big.Rat, dp int) string {
	if r == nil {
		r = new(big.Rat)
	}
	if dp < 0 {
		dp = 0
	}
	neg := r.Sign() < 0
	abs := new(big.Rat).Abs(r)

	// scale = 10^dp
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(dp)), nil)
	scaled := new(big.Rat).Mul(abs, new(big.Rat).SetInt(scale))

	// round half away from zero: floor(scaled + 1/2)
	half := big.NewRat(1, 2)
	scaled.Add(scaled, half)
	q := new(big.Int).Quo(scaled.Num(), scaled.Denom()) // truncation toward zero; operand is non-negative

	digits := q.String()
	var out string
	if dp == 0 {
		out = digits
	} else {
		for len(digits) <= dp {
			digits = "0" + digits
		}
		out = digits[:len(digits)-dp] + "." + digits[len(digits)-dp:]
	}
	if neg && q.Sign() != 0 {
		out = "-" + out
	}
	return out
}

// MustRat parses s or panics; for use in tests and constant fixtures only.
func MustRat(s string) *big.Rat {
	r, err := StringToRat(s)
	if err != nil {
		panic(err)
	}
	return r
}
