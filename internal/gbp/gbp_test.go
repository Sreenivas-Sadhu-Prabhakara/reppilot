package gbp

import (
	"reflect"
	"strings"
	"testing"
)

func TestMockDeterminism(t *testing.T) {
	m := Mock{}
	p1, r1 := m.Connect("Meera's Salon", "Pune", "Salon")
	p2, r2 := m.Connect("Meera's Salon", "Pune", "Salon")

	// ConnectedAt is wall-clock; everything else must match exactly.
	p1.ConnectedAt = p2.ConnectedAt
	if !reflect.DeepEqual(p1, p2) {
		t.Fatalf("profiles differ for identical input:\n%+v\n%+v", p1, p2)
	}
	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("reviews differ for identical input")
	}
}

func TestMockDifferentInputsDiffer(t *testing.T) {
	m := Mock{}
	_, r1 := m.Connect("Meera's Salon", "Pune", "Salon")
	_, r2 := m.Connect("Meera's Salon", "Mumbai", "Salon")
	if reflect.DeepEqual(r1, r2) {
		t.Fatal("different cities produced identical review sets")
	}
}

func TestMockReviewShape(t *testing.T) {
	m := Mock{}
	p, reviews := m.Connect("Anand Dental Clinic", "Chennai", "Clinic")

	if len(reviews) < 25 || len(reviews) > 40 {
		t.Fatalf("want 25-40 reviews, got %d", len(reviews))
	}
	if p.Rating < 1 || p.Rating > 5 {
		t.Fatalf("profile rating out of range: %v", p.Rating)
	}
	if len(p.Competitors) != 3 {
		t.Fatalf("want 3 competitors, got %d", len(p.Competitors))
	}
	if !strings.HasPrefix(p.Phone, "+91-98") || len(p.Phone) != len("+91-9800000000") {
		t.Fatalf("bad phone format: %q", p.Phone)
	}

	unanswered := 0
	stars := map[int]bool{}
	for i, rv := range reviews {
		if rv.Rating < 1 || rv.Rating > 5 {
			t.Fatalf("review %s rating out of range: %d", rv.ID, rv.Rating)
		}
		stars[rv.Rating] = true
		if !rv.Replied {
			unanswered++
		}
		if rv.Date.After(Anchor) || rv.Date.Before(Anchor.AddDate(-1, 0, -1)) {
			t.Fatalf("review %s dated outside the 12-month window: %v", rv.ID, rv.Date)
		}
		if i > 0 && reviews[i-1].Date.Before(rv.Date) {
			t.Fatal("reviews not sorted newest-first")
		}
	}
	// ~60% unanswered; allow generous randomness band.
	frac := float64(unanswered) / float64(len(reviews))
	if frac < 0.35 || frac > 0.85 {
		t.Fatalf("unanswered fraction %v outside plausible band around 0.6", frac)
	}
	if len(stars) < 3 {
		t.Fatalf("expected a mix of star ratings, got only %d distinct values", len(stars))
	}
}

func TestCompetitorPoolMatchesCategory(t *testing.T) {
	m := Mock{}
	p, _ := m.Connect("Tandoor House", "Delhi", "Restaurant")
	for _, c := range p.Competitors {
		found := false
		for _, name := range competitorNames["restaurant"] {
			if c.Name == name {
				found = true
			}
		}
		if !found {
			t.Fatalf("competitor %q not from the restaurant pool", c.Name)
		}
	}
}
