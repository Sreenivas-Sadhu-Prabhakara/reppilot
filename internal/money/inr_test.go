package money

import "testing"

func TestFormatINR(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{12500, "₹12,500"},
		{999, "₹999"},
		{0, "₹0"},
		{100, "₹100"},
		{1234567 * 10, "₹1.2 Cr"}, // 1,23,45,670
		{12000000, "₹1.2 Cr"},
		{10000000, "₹1 Cr"},
		{3650000, "₹36.5 L"},
		{100000, "₹1 L"},
		{99999, "₹99,999"},
		{1234567, "₹12.3 L"},
		{-12500, "-₹12,500"},
	}
	for _, c := range cases {
		if got := FormatINR(c.in); got != c.want {
			t.Errorf("FormatINR(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGroupIndian(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{999, "999"},
		{1000, "1,000"},
		{12500, "12,500"},
		{99999, "99,999"},
	}
	for _, c := range cases {
		if got := groupIndian(c.in); got != c.want {
			t.Errorf("groupIndian(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}
