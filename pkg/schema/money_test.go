package schema_test

import (
	"math/big"
	"testing"

	"github.com/osifotos/price-checker/pkg/schema"
	"pgregory.net/rapid"
)

func TestRatToString(t *testing.T) {
	cases := []struct {
		in   string
		dp   int
		want string
	}{
		{"0", 2, "0.00"},
		{"12.4830", 2, "12.48"},
		{"12.485", 2, "12.49"},
		{"12.484", 2, "12.48"},
		{"-1.005", 2, "-1.01"},
		{"0.0000041667", 10, "0.0000041667"},
		{"1", 0, "1"},
		{"-0.001", 2, "0.00"},
	}
	for _, c := range cases {
		if got := schema.RatToString(schema.MustRat(c.in), c.dp); got != c.want {
			t.Errorf("RatToString(%s, %d) = %q, want %q", c.in, c.dp, got, c.want)
		}
	}
}

func TestStringToRatRejects(t *testing.T) {
	bad := []string{"", " ", "abc", "1.2.3", "1e3", "NaN", "Inf", "--1", "1,000", "."}
	for _, s := range bad {
		if _, err := schema.StringToRat(s); err == nil {
			t.Errorf("StringToRat(%q) = nil error, want error", s)
		}
	}
}

// PBT-02: format then parse is lossless and stable at the chosen precision.
func TestMoneyRoundTripProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		num := rapid.Int64Range(-1_000_000_000, 1_000_000_000).Draw(t, "num")
		dp := rapid.IntRange(0, 8).Draw(t, "dp")
		den := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(dp)), nil)
		orig := new(big.Rat).SetFrac(big.NewInt(num), den)

		s := schema.RatToString(orig, dp)
		parsed, err := schema.StringToRat(s)
		if err != nil {
			t.Fatalf("StringToRat(%q): %v", s, err)
		}
		if schema.RatToString(parsed, dp) != s {
			t.Fatalf("round trip changed %q -> %q", s, schema.RatToString(parsed, dp))
		}
	})
}
