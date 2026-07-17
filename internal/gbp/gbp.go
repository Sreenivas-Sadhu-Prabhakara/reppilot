// Package gbp defines the Google Business Profile provider interface and its
// deterministic mock. A live implementation would be selected when
// GOOGLE_PLACES_API_KEY is set (documented in README; not implemented).
package gbp

import (
	"fmt"
	"hash/fnv"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"reppilot/internal/domain"
)

// Anchor is the fixed "today" that all mock review dates hang off, so demo
// data is stable no matter when the server runs.
var Anchor = time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)

// Provider fetches a business profile and its reviews.
type Provider interface {
	Connect(businessName, city, category string) (domain.Profile, []domain.Review)
	Mode() string
}

// Mock deterministically fabricates a profile, 25-40 reviews over the past
// 12 months, and 3 nearby competitors from an FNV hash of the input.
type Mock struct{}

// Mode reports the provider mode for /health.
func (Mock) Mode() string { return "mock" }

func seedFor(parts ...string) int64 {
	h := fnv.New64a()
	for _, p := range parts {
		h.Write([]byte(strings.ToLower(strings.TrimSpace(p))))
		h.Write([]byte{'|'})
	}
	return int64(h.Sum64() & math.MaxInt64)
}

var firstNames = []string{
	"Aarav", "Priya", "Rohan", "Sneha", "Vikram", "Ananya", "Karthik", "Divya",
	"Rahul", "Pooja", "Amit", "Neha", "Sanjay", "Kavya", "Arjun", "Meera",
	"Nikhil", "Shreya", "Rajesh", "Isha", "Manish", "Lakshmi", "Varun", "Anjali",
	"Deepak", "Ritika", "Suresh", "Nandini", "Harsha", "Gayatri", "Farhan", "Zoya",
	"Gurpreet", "Simran", "Joseph", "Tanvi",
}

var lastNames = []string{
	"Sharma", "Verma", "Iyer", "Reddy", "Patel", "Nair", "Gupta", "Mehta",
	"Kulkarni", "Banerjee", "Chopra", "Joshi", "Rao", "Menon", "Singh", "Das",
	"Pillai", "Malhotra", "Desai", "Bhat", "Khan", "Fernandes", "Agarwal", "Shetty",
}

// reviewTexts maps star rating -> candidate texts (mix of English & Hinglish).
var reviewTexts = map[int][]string{
	5: {
		"Absolutely loved the experience! Staff was courteous and the place was spotless.",
		"Ekdum mast service! Staff bahut friendly hai, highly recommend.",
		"Best in the area, hands down. Quality is consistently excellent.",
		"Paisa vasool! Itni acchi service kahin nahi milti yahan ke aas paas.",
		"Wonderful experience from start to finish. The attention to detail is superb.",
		"Superb! Booking se lekar billing tak sab smooth tha. Five stars banta hai.",
		"Truly professional team. They remembered my preferences from last time!",
		"Bahut hi accha experience. Staff ne har cheez patiently explain ki.",
	},
	4: {
		"Very good experience overall. Slight wait but totally worth it.",
		"Service acchi thi, thoda expensive hai but quality solid hai.",
		"Great place, friendly staff. Parking is a bit of a hassle though.",
		"Accha experience tha overall. Bas AC thoda kam tha waiting area me.",
		"Really good quality and courteous staff. Would have loved quicker billing.",
		"Achha hai! Staff polite hai, bas weekend pe thoda rush ho jaata hai.",
	},
	3: {
		"Service accha tha but waiting bahut zyada.",
		"Average experience. Nothing bad, nothing memorable either.",
		"Decent quality but billing me kaafi time laga. Can improve.",
		"Theek thaak hai. Staff accha hai but hygiene pe aur dhyan dena chahiye.",
		"Okay-ish experience. The service was fine but the ambience needs work.",
	},
	2: {
		"Disappointed. Appointment ke baad bhi 40 minute wait karna pada.",
		"Not up to the mark. Staff seemed rushed and billing was a mess.",
		"Below average. Service was slow and no one bothered to update us.",
		"Kaafi disappointing. Pehle jaisi quality nahi rahi ab.",
	},
	1: {
		"Very poor experience. Bilkul time waste, will not come back.",
		"Worst service ever, no one attended us for 30 minutes.",
		"Pathetic. Manager ne complaint bhi nahi suni properly.",
		"Terrible hygiene and rude staff. Avoid this place.",
	},
}

var ownerReplies = []string{
	"Thank you for your feedback, %s! Hope to see you again soon.",
	"Thanks a lot %s, your support means a lot to us!",
	"%s ji, feedback ke liye dhanyavaad. Aapka phir se swagat hai!",
	"We appreciate you taking the time, %s. See you again!",
}

var competitorNames = map[string][]string{
	"salon": {
		"Glow & Co Salon", "Style Adda", "Mirror Mirror Unisex Salon",
		"Scissors Talk", "The Hair Affair", "Roots & Shears",
	},
	"clinic": {
		"CityCare Clinic", "Arogya Multi-speciality Clinic", "LifePlus Clinic",
		"Sanjeevani Health Centre", "WellSpring Clinic", "Nirmal Polyclinic",
	},
	"restaurant": {
		"Spice Route", "Tandoor Tales", "The Curry Leaf",
		"Dilli Darbar", "Saffron Story", "Biryani Junction",
	},
	"cafe": {
		"Chai Tapri", "Filter Kaapi House", "Brew & Bun",
		"Adda Cafe", "Cutting Chai Co", "Roast Republic",
	},
	"generic": {
		"Sunrise Enterprises", "Shree Balaji Services", "Metro Point",
		"Lotus Corner", "Golden Gate Services", "Pearl Plaza",
	},
}

func competitorPool(category string) []string {
	c := strings.ToLower(category)
	for key, pool := range competitorNames {
		if key != "generic" && strings.Contains(c, key) {
			return pool
		}
	}
	if strings.Contains(c, "restau") || strings.Contains(c, "dhaba") || strings.Contains(c, "food") {
		return competitorNames["restaurant"]
	}
	return competitorNames["generic"]
}

// Connect fabricates the profile and reviews. Same input -> same output.
func (Mock) Connect(businessName, city, category string) (domain.Profile, []domain.Review) {
	seed := seedFor(businessName, city, category)
	r := rand.New(rand.NewSource(seed))

	n := 25 + r.Intn(16) // 25-40 reviews
	reviews := make([]domain.Review, 0, n)
	ratingSum := 0
	for i := 0; i < n; i++ {
		rating := pickRating(r)
		ratingSum += rating
		first := firstNames[r.Intn(len(firstNames))]
		last := lastNames[r.Intn(len(lastNames))]
		texts := reviewTexts[rating]
		text := texts[r.Intn(len(texts))]
		daysAgo := r.Intn(365)
		date := Anchor.AddDate(0, 0, -daysAgo).Add(-time.Duration(r.Intn(12)) * time.Hour)

		rv := domain.Review{
			Reviewer: first + " " + last,
			Rating:   rating,
			Text:     text,
			Date:     date,
		}
		// ~60% of reviews are missing an owner reply.
		if r.Float64() >= 0.60 {
			rv.Replied = true
			rv.Reply = fmt.Sprintf(ownerReplies[r.Intn(len(ownerReplies))], first)
			t := date.AddDate(0, 0, 1+r.Intn(5))
			rv.RepliedAt = &t
		}
		reviews = append(reviews, rv)
	}
	sort.Slice(reviews, func(i, j int) bool { return reviews[i].Date.After(reviews[j].Date) })
	for i := range reviews {
		reviews[i].ID = fmt.Sprintf("rv-%03d", i+1)
	}

	avg := math.Round(float64(ratingSum)/float64(n)*10) / 10

	pool := competitorPool(category)
	perm := r.Perm(len(pool))
	comps := make([]domain.Competitor, 0, 3)
	for _, idx := range perm[:3] {
		comps = append(comps, domain.Competitor{
			Name:        pool[idx],
			Rating:      math.Round((3.6+r.Float64()*1.2)*10) / 10,
			ReviewCount: 50 + r.Intn(700),
		})
	}

	profile := domain.Profile{
		BusinessName: strings.TrimSpace(businessName),
		City:         strings.TrimSpace(city),
		Category:     strings.TrimSpace(category),
		Phone:        fmt.Sprintf("+91-98%08d", r.Intn(100000000)),
		Rating:       avg,
		ReviewCount:  n + 40 + r.Intn(320), // lifetime count; inbox shows latest 12 months
		ReviewLink:   fmt.Sprintf("https://g.page/r/%x/review", uint64(seed)),
		ConnectedAt:  time.Now().UTC(),
		Competitors:  comps,
	}
	return profile, reviews
}

// pickRating draws a star rating with a realistic local-business skew:
// mostly 4-5 stars, a meaningful tail of 1-2 star complaints.
func pickRating(r *rand.Rand) int {
	p := r.Float64()
	switch {
	case p < 0.35:
		return 5
	case p < 0.65:
		return 4
	case p < 0.77:
		return 3
	case p < 0.87:
		return 2
	default:
		return 1
	}
}
