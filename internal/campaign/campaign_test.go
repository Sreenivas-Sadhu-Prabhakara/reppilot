package campaign

import (
	"strings"
	"testing"

	"reppilot/internal/domain"
)

func TestParseCustomers(t *testing.T) {
	raw := `
Priya Sharma, +91-9812345678
Rohan Verma, +91 98123 45679
Amit Kulkarni, 9198765432
bad line without phone
Neha, +91-1234567890
, +91-9876543210
`
	customers, skipped := ParseCustomers(raw)
	if len(customers) != 3 {
		t.Fatalf("want 3 valid customers, got %d: %+v", len(customers), customers)
	}
	if customers[0].Name != "Priya Sharma" || customers[0].Phone != "+91-9812345678" {
		t.Errorf("first customer wrong: %+v", customers[0])
	}
	if customers[1].Phone != "+91-9812345679" {
		t.Errorf("spaced phone not normalized: %+v", customers[1])
	}
	if customers[2].Phone != "+91-9198765432" {
		t.Errorf("bare 10-digit phone not normalized: %+v", customers[2])
	}
	if len(skipped) != 3 {
		t.Fatalf("want 3 skipped lines, got %d: %v", len(skipped), skipped)
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"+91-9812345678", "+91-9812345678", true},
		{"919812345678", "+91-9812345678", true},
		{"09812345678", "+91-9812345678", true},
		{"98123 45678", "+91-9812345678", true},
		{"+91-1234567890", "", false}, // mobiles start 6-9
		{"98123", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := NormalizePhone(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalizePhone(%q) = %q,%v want %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestBuildMessagePersonalization(t *testing.T) {
	c := domain.Customer{Name: "Priya Sharma", Phone: "+91-9812345678"}
	link := "https://g.page/r/abc123/review"
	msg := BuildMessage(c, "Meera's Salon", "Pune", link)

	if !strings.Contains(msg, "Hi Priya!") {
		t.Errorf("message should greet by first name: %q", msg)
	}
	if strings.Contains(msg, "Sharma!") {
		t.Errorf("greeting should use first name only: %q", msg)
	}
	if !strings.Contains(msg, "Meera's Salon") || !strings.Contains(msg, "Pune") {
		t.Errorf("message should mention business and city: %q", msg)
	}
	if !strings.Contains(msg, link) {
		t.Errorf("message should carry the review link: %q", msg)
	}

	other := BuildMessage(domain.Customer{Name: "Rohan Verma"}, "Meera's Salon", "Pune", link)
	if other == msg {
		t.Error("messages for different customers should differ")
	}
}
