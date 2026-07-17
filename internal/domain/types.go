// Package domain holds the core types shared across RepPilot.
package domain

import "time"

// Profile is a connected Google Business Profile (mock).
type Profile struct {
	BusinessName string       `json:"business_name"`
	City         string       `json:"city"`
	Category     string       `json:"category"`
	Phone        string       `json:"phone"`
	Rating       float64      `json:"rating"`
	ReviewCount  int          `json:"review_count"`
	ReviewLink   string       `json:"review_link"`
	ConnectedAt  time.Time    `json:"connected_at"`
	Competitors  []Competitor `json:"competitors"`
}

// Competitor is a nearby business in the same category.
type Competitor struct {
	Name        string  `json:"name"`
	Rating      float64 `json:"rating"`
	ReviewCount int     `json:"review_count"`
}

// Review is a single customer review in the inbox.
type Review struct {
	ID        string     `json:"id"`
	Reviewer  string     `json:"reviewer"`
	Rating    int        `json:"rating"`
	Text      string     `json:"text"`
	Date      time.Time  `json:"date"`
	Replied   bool       `json:"replied"`
	Reply     string     `json:"reply,omitempty"`
	RepliedAt *time.Time `json:"replied_at,omitempty"`
	Draft     string     `json:"draft,omitempty"`
}

// Customer is one recipient in a review-request campaign.
type Customer struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

// Campaign is a review-request campaign sent over WhatsApp.
type Campaign struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	Customers []Customer `json:"customers"`
	Sent      int        `json:"sent"`
	Skipped   []string   `json:"skipped,omitempty"`
}

// OutboxMessage is a WhatsApp message queued by the mock sender.
type OutboxMessage struct {
	ID        string    `json:"id"`
	Channel   string    `json:"channel"`
	Kind      string    `json:"kind"` // "campaign" or "digest"
	To        string    `json:"to"`
	Name      string    `json:"name"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
