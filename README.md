# RepPilot — Reputation Autopilot

A review inbox with AI-drafted replies, WhatsApp review-request campaigns and a
weekly reputation digest — built for local Indian businesses (salons, clinics,
restaurants, cafes).

**Who pays for it:** the owner of a local business who lives and dies by their
Google rating but has no time to answer reviews. RepPilot drafts every reply in
their tone and language (English or Hinglish), chases happy customers for new
reviews over WhatsApp, and mails a weekly digest with a competitor check.
**₹999/mo** subscription.

Everything in this MVP runs offline with deterministic mocks — same business
name + city + category always produces the same profile, reviews and
competitors. No API keys needed.

## Quickstart

```bash
make run          # serves UI + API on http://localhost:8102
```

Open http://localhost:8102, type a business (e.g. *Meera's Salon / Pune /
Salon*) and click **Connect profile**.

Other targets: `make build` (binary in `bin/`), `make test`.

## API summary

Base path: `/api/v1`

| Method | Path                     | What it does |
|--------|--------------------------|--------------|
| GET    | `/health`                | `{"status":"ok"}` + provider modes |
| POST   | `/profile/connect`       | `{business_name, city, category}` → mock GBP fetch: profile, 25-40 reviews (12 months, anchored to 2026-07-15), 3 competitors |
| GET    | `/profile`               | Connected profile + unanswered count |
| GET    | `/reviews?filter=&rating=` | List reviews; `filter` = `unanswered` / `answered`, `rating` = 1-5 |
| POST   | `/reviews/{id}/draft`    | `{tone, language}` → rating-aware personalized reply draft |
| POST   | `/reviews/{id}/reply`    | `{reply}` → marks the review answered, stores the reply |
| POST   | `/reviews/draft-all`     | `{tone, language}` → drafts every unanswered review |
| POST   | `/campaigns`             | `{name, customers}` (one `Name, +91-98xxxxxxxx` per line) → personalized WhatsApp messages queued to the outbox |
| GET    | `/campaigns`             | List campaigns, newest first |
| GET    | `/outbox`                | Queued WhatsApp messages (campaigns + digests) |
| GET    | `/digest`                | Weekly digest: 12-month rating trend, unanswered count, response rate, competitor table |
| POST   | `/digest/send`           | `{phone}` → digest as a WhatsApp message in the outbox |

Reply drafter: tones **Professional / Warm / Brief**, languages **English /
Hinglish**. 1-2★ → apology + service-recovery offer + take-it-offline;
3★ → thanks + improvement note; 4-5★ → gratitude + invite back. Replies are
personalized with the reviewer's first name and echo a keyword from their
review.

## Upgrade to live

Each integration sits behind a small interface; the mock is the only shipped
implementation. Setting these env vars is how a live build would switch over:

| Env var                 | Interface        | Mock behaviour today                      | Live behaviour |
|-------------------------|------------------|-------------------------------------------|----------------|
| `GOOGLE_PLACES_API_KEY` | `gbp.Provider`   | Deterministic profile/reviews from FNV hash | Live Google Business Profile fetch |
| `ANTHROPIC_API_KEY`     | `drafter.Drafter`| Rating-aware template engine               | Claude-drafted replies |
| `AISENSY_API_KEY`       | `wa.Sender`      | Messages queued to local outbox            | Real WhatsApp sends via AiSensy |

## Storage

In-memory store guarded by a mutex, snapshotted as JSON to `./data/store.json`
after every write and reloaded on boot. Delete the file to reset the demo.

## Configuration

| Env var | Default | Meaning |
|---------|---------|---------|
| `PORT`  | `8102`  | HTTP listen port |
