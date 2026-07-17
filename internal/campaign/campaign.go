// Package campaign parses pasted customer lists and personalizes
// review-request WhatsApp messages.
package campaign

import (
	"fmt"
	"strings"

	"reppilot/internal/domain"
)

// ParseCustomers reads one "Name, +91-98xxxxxxxx" per line. Lines that do not
// yield a name and a valid Indian mobile number are returned in skipped.
func ParseCustomers(raw string) (customers []domain.Customer, skipped []string) {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.LastIndex(line, ",")
		if idx < 0 {
			skipped = append(skipped, line)
			continue
		}
		name := strings.TrimSpace(line[:idx])
		phone, ok := NormalizePhone(line[idx+1:])
		if name == "" || !ok {
			skipped = append(skipped, line)
			continue
		}
		customers = append(customers, domain.Customer{Name: name, Phone: phone})
	}
	return customers, skipped
}

// NormalizePhone accepts "+91-9812345678", "+91 98123 45678", "9812345678"
// etc. and normalizes to "+91-XXXXXXXXXX". Indian mobiles start with 6-9.
func NormalizePhone(raw string) (string, bool) {
	digits := strings.Builder{}
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	d = strings.TrimPrefix(d, "0")
	if len(d) == 12 && strings.HasPrefix(d, "91") {
		d = d[2:]
	}
	if len(d) != 10 || d[0] < '6' || d[0] > '9' {
		return "", false
	}
	return "+91-" + d, true
}

// BuildMessage personalizes the review-request WhatsApp text for a customer.
func BuildMessage(c domain.Customer, businessName, city, reviewLink string) string {
	first := c.Name
	if fields := strings.Fields(c.Name); len(fields) > 0 {
		first = fields[0]
	}
	return fmt.Sprintf(
		"Hi %s! Thank you for visiting %s, %s. Could you take 30 seconds to share your experience? Your review helps a local business grow: %s — Team %s",
		first, businessName, city, reviewLink, businessName,
	)
}
