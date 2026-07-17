// Package money formats amounts using Indian numbering conventions.
package money

import (
	"fmt"
	"strings"
)

// FormatINR renders rupee amounts the way Indians read them:
// ₹1.2 Cr (crores), ₹36.5 L (lakhs), ₹12,500 (Indian digit grouping).
func FormatINR(amount float64) string {
	neg := amount < 0
	if neg {
		amount = -amount
	}
	var s string
	switch {
	case amount >= 1e7:
		s = "₹" + trimTrailingZero(fmt.Sprintf("%.1f", amount/1e7)) + " Cr"
	case amount >= 1e5:
		s = "₹" + trimTrailingZero(fmt.Sprintf("%.1f", amount/1e5)) + " L"
	default:
		s = "₹" + groupIndian(int64(amount+0.5))
	}
	if neg {
		return "-" + s
	}
	return s
}

func trimTrailingZero(s string) string {
	return strings.TrimSuffix(s, ".0")
}

// groupIndian applies the 2,2,3 grouping: 1234567 -> 12,34,567.
func groupIndian(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	head, tail := s[:len(s)-3], s[len(s)-3:]
	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, ",") + "," + tail
}
