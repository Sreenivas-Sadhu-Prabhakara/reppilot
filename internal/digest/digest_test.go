package digest

import (
	"strings"
	"testing"
	"time"

	"reppilot/internal/domain"
)

func TestBuildDigestMath(t *testing.T) {
	anchor := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	p := domain.Profile{
		BusinessName: "Meera's Salon", City: "Pune", Category: "Salon",
		Rating: 4.2, ReviewCount: 120,
		Competitors: []domain.Competitor{
			{Name: "Style Adda", Rating: 4.5, ReviewCount: 300},
			{Name: "Scissors Talk", Rating: 3.9, ReviewCount: 80},
		},
	}
	reviews := []*domain.Review{
		{ID: "rv-001", Rating: 5, Date: anchor.AddDate(0, 0, -2), Replied: true},
		{ID: "rv-002", Rating: 3, Date: anchor.AddDate(0, 0, -3)},
		{ID: "rv-003", Rating: 1, Date: anchor.AddDate(0, -1, 0)},
		{ID: "rv-004", Rating: 4, Date: anchor.AddDate(0, -11, 0), Replied: true},
	}

	d := Build(p, reviews, anchor)

	if d.Unanswered != 2 {
		t.Errorf("unanswered = %d, want 2", d.Unanswered)
	}
	if d.ResponseRate != 50.0 {
		t.Errorf("response rate = %v, want 50.0", d.ResponseRate)
	}
	if len(d.Trend) != 12 {
		t.Fatalf("trend months = %d, want 12", len(d.Trend))
	}
	if d.Trend[0].Month != "Aug 2025" || d.Trend[11].Month != "Jul 2026" {
		t.Errorf("trend window wrong: %s .. %s", d.Trend[0].Month, d.Trend[11].Month)
	}
	// July 2026 has rv-001 (5) and rv-002 (3): avg 4.0, count 2.
	july := d.Trend[11]
	if july.Count != 2 || july.AvgRating != 4.0 {
		t.Errorf("July stat = %+v, want count 2 avg 4.0", july)
	}
	// August 2025 holds rv-004.
	if d.Trend[0].Count != 1 {
		t.Errorf("Aug 2025 count = %d, want 1", d.Trend[0].Count)
	}
	if len(d.Competitors) != 2 {
		t.Fatalf("competitor rows = %d, want 2", len(d.Competitors))
	}
	if d.Competitors[0].Delta != -0.3 {
		t.Errorf("delta vs Style Adda = %v, want -0.3", d.Competitors[0].Delta)
	}
	if d.Competitors[1].Delta != 0.3 {
		t.Errorf("delta vs Scissors Talk = %v, want 0.3", d.Competitors[1].Delta)
	}
	if d.PlanPrice != "₹999/mo" {
		t.Errorf("plan price = %q, want ₹999/mo", d.PlanPrice)
	}

	text := d.WhatsAppText()
	for _, want := range []string{"Meera's Salon", "Unanswered reviews: 2", "Style Adda"} {
		if !strings.Contains(text, want) {
			t.Errorf("WhatsApp text missing %q: %s", want, text)
		}
	}
}

func TestBuildDigestEmptyReviews(t *testing.T) {
	anchor := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	d := Build(domain.Profile{BusinessName: "X"}, nil, anchor)
	if d.ResponseRate != 0 || d.Unanswered != 0 {
		t.Errorf("empty digest should be all zeroes: %+v", d)
	}
}
