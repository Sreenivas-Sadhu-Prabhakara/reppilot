// Package wa defines the WhatsApp sender interface and its mock. A live
// implementation (AiSensy) would be selected when AISENSY_API_KEY is set
// (documented in README; intentionally not implemented).
package wa

import (
	"time"

	"reppilot/internal/domain"
)

// Sender delivers a WhatsApp message. The mock queues it into the outbox.
type Sender interface {
	Send(kind, to, name, body string) domain.OutboxMessage
	Mode() string
}

// Mock builds an outbox message instead of hitting a live API.
type Mock struct{}

// Mode reports the sender mode for /health.
func (Mock) Mode() string { return "mock" }

// Send returns a queued message; the caller assigns the ID and persists it.
func (Mock) Send(kind, to, name, body string) domain.OutboxMessage {
	return domain.OutboxMessage{
		Channel:   "whatsapp",
		Kind:      kind,
		To:        to,
		Name:      name,
		Body:      body,
		Status:    "queued (mock)",
		CreatedAt: time.Now().UTC(),
	}
}
