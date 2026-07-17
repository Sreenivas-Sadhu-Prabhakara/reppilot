// Package digest computes the weekly reputation digest.
package digest

import (
	"fmt"
	"math"
	"strings"
	"time"

	"reppilot/internal/domain"
	"reppilot/internal/money"
)

// MonthStat is one bar in the 12-month rating trend.
type MonthStat struct {
	Month     string  `json:"month"` // e.g. "Aug 2025"
	Count     int     `json:"count"`
	AvgRating float64 `json:"avg_rating"` // 0 when no reviews that month
}

// CompetitorRow compares the business against one nearby competitor.
type CompetitorRow struct {
	Name        string  `json:"name"`
	Rating      float64 `json:"rating"`
	ReviewCount int     `json:"review_count"`
	Delta       float64 `json:"delta"` // our rating minus theirs
}

// Digest is the weekly reputation report.
type Digest struct {
	Business       string          `json:"business"`
	City           string          `json:"city"`
	Category       string          `json:"category"`
	WeekOf         string          `json:"week_of"`
	Rating         float64         `json:"rating"`
	ReviewCount    int             `json:"review_count"`
	ReviewsTracked int             `json:"reviews_tracked"`
	Unanswered     int             `json:"unanswered"`
	ResponseRate   float64         `json:"response_rate"` // percent, 1 decimal
	Trend          []MonthStat     `json:"trend"`
	Competitors    []CompetitorRow `json:"competitors"`
	PlanPrice      string          `json:"plan_price"`
	GeneratedAt    time.Time       `json:"generated_at"`
}

// Build computes the digest for the last 12 months ending at anchor.
func Build(p domain.Profile, reviews []*domain.Review, anchor time.Time) Digest {
	type acc struct {
		sum, n int
	}
	months := make([]MonthStat, 12)
	accs := make([]acc, 12)
	start := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -11, 0)
	for i := 0; i < 12; i++ {
		months[i].Month = start.AddDate(0, i, 0).Format("Jan 2006")
	}

	unanswered := 0
	for _, rv := range reviews {
		if !rv.Replied {
			unanswered++
		}
		idx := monthsBetween(start, rv.Date)
		if idx >= 0 && idx < 12 {
			accs[idx].sum += rv.Rating
			accs[idx].n++
		}
	}
	for i := range months {
		months[i].Count = accs[i].n
		if accs[i].n > 0 {
			months[i].AvgRating = math.Round(float64(accs[i].sum)/float64(accs[i].n)*10) / 10
		}
	}

	respRate := 0.0
	if len(reviews) > 0 {
		respRate = math.Round(float64(len(reviews)-unanswered)/float64(len(reviews))*1000) / 10
	}

	rows := make([]CompetitorRow, 0, len(p.Competitors))
	for _, c := range p.Competitors {
		rows = append(rows, CompetitorRow{
			Name:        c.Name,
			Rating:      c.Rating,
			ReviewCount: c.ReviewCount,
			Delta:       math.Round((p.Rating-c.Rating)*10) / 10,
		})
	}

	return Digest{
		Business:       p.BusinessName,
		City:           p.City,
		Category:       p.Category,
		WeekOf:         anchor.Format("2 Jan 2006"),
		Rating:         p.Rating,
		ReviewCount:    p.ReviewCount,
		ReviewsTracked: len(reviews),
		Unanswered:     unanswered,
		ResponseRate:   respRate,
		Trend:          months,
		Competitors:    rows,
		PlanPrice:      money.FormatINR(999) + "/mo",
		GeneratedAt:    time.Now().UTC(),
	}
}

func monthsBetween(start, t time.Time) int {
	return (t.Year()-start.Year())*12 + int(t.Month()) - int(start.Month())
}

// WhatsAppText renders the digest as a compact WhatsApp message.
func (d Digest) WhatsAppText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "RepPilot Weekly — %s, %s (week of %s)\n", d.Business, d.City, d.WeekOf)
	fmt.Fprintf(&b, "Rating: %.1f★ across %d reviews\n", d.Rating, d.ReviewCount)
	fmt.Fprintf(&b, "Unanswered reviews: %d | Response rate: %.1f%%\n", d.Unanswered, d.ResponseRate)
	if len(d.Competitors) > 0 {
		b.WriteString("Nearby: ")
		parts := make([]string, 0, len(d.Competitors))
		for _, c := range d.Competitors {
			parts = append(parts, fmt.Sprintf("%s %.1f★", c.Name, c.Rating))
		}
		b.WriteString(strings.Join(parts, ", "))
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "Reply to your customers today — RepPilot (%s)", d.PlanPrice)
	return b.String()
}
