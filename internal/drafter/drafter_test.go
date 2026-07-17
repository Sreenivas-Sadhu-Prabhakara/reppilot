package drafter

import (
	"strings"
	"testing"
	"time"

	"reppilot/internal/domain"
)

const (
	biz   = "Meera's Salon"
	phone = "+91-9812345678"
)

func review(rating int, reviewer, text string) domain.Review {
	return domain.Review{
		ID: "rv-001", Reviewer: reviewer, Rating: rating, Text: text,
		Date: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestNegativeEnglishHasApologyRecoveryAndOffline(t *testing.T) {
	d := Template{}
	for _, rating := range []int{1, 2} {
		for _, tone := range []string{ToneProfessional, ToneWarm, ToneBrief} {
			rv := review(rating, "Rohan Verma", "Worst service, waiting was endless")
			got := d.Draft(rv, biz, phone, tone, LangEnglish)
			low := strings.ToLower(got)
			if !strings.Contains(low, "sorry") && !strings.Contains(low, "apolog") {
				t.Errorf("%s %d-star: no apology in %q", tone, rating, got)
			}
			if !strings.Contains(low, "next visit is on us") {
				t.Errorf("%s %d-star: no service-recovery offer in %q", tone, rating, got)
			}
			if !strings.Contains(got, phone) {
				t.Errorf("%s %d-star: no take-it-offline phone in %q", tone, rating, got)
			}
			if !strings.Contains(got, "Rohan") {
				t.Errorf("%s %d-star: reviewer first name missing in %q", tone, rating, got)
			}
		}
	}
}

func TestNegativeHinglishHasApologyRecoveryAndOffline(t *testing.T) {
	d := Template{}
	for _, tone := range []string{ToneProfessional, ToneWarm, ToneBrief} {
		rv := review(1, "Pooja Mehta", "Pathetic. Manager ne complaint bhi nahi suni")
		got := d.Draft(rv, biz, phone, tone, LangHinglish)
		low := strings.ToLower(got)
		if !strings.Contains(low, "afsos") && !strings.Contains(low, "sorry") && !strings.Contains(low, "maafi") {
			t.Errorf("%s: no Hinglish apology in %q", tone, got)
		}
		if !strings.Contains(low, "agli visit hum par") {
			t.Errorf("%s: no service-recovery offer in %q", tone, got)
		}
		if !strings.Contains(got, phone) {
			t.Errorf("%s: no take-it-offline phone in %q", tone, got)
		}
		if !strings.Contains(got, "Pooja") {
			t.Errorf("%s: reviewer first name missing in %q", tone, got)
		}
	}
}

func TestNeutralThanksPlusImprovement(t *testing.T) {
	d := Template{}
	rv := review(3, "Sneha Iyer", "Service accha tha but waiting bahut zyada")

	en := d.Draft(rv, biz, phone, ToneProfessional, LangEnglish)
	if !strings.Contains(strings.ToLower(en), "thank") {
		t.Errorf("3-star English should thank the reviewer: %q", en)
	}
	if !strings.Contains(strings.ToLower(en), "improving") {
		t.Errorf("3-star English should note improvement: %q", en)
	}

	hi := d.Draft(rv, biz, phone, ToneProfessional, LangHinglish)
	if !strings.Contains(strings.ToLower(hi), "dhanyavaad") {
		t.Errorf("3-star Hinglish should thank the reviewer: %q", hi)
	}
	if !strings.Contains(strings.ToLower(hi), "kaam kar rahe") {
		t.Errorf("3-star Hinglish should note improvement: %q", hi)
	}
	// Neutral replies must not offer service recovery or push offline.
	if strings.Contains(en, phone) || strings.Contains(hi, phone) {
		t.Error("3-star reply should not include the offline phone number")
	}
}

func TestPositiveGratitudeAndInviteBack(t *testing.T) {
	d := Template{}
	for _, rating := range []int{4, 5} {
		rv := review(rating, "Aarav Nair", "Loved the haircut, staff was great")
		en := d.Draft(rv, biz, phone, ToneWarm, LangEnglish)
		low := strings.ToLower(en)
		if !strings.Contains(low, "happy") && !strings.Contains(low, "thank") {
			t.Errorf("%d-star English lacks gratitude: %q", rating, en)
		}
		if !strings.Contains(low, "come back") && !strings.Contains(low, "welcoming you back") && !strings.Contains(low, "see you again") {
			t.Errorf("%d-star English lacks invite-back: %q", rating, en)
		}
		hi := d.Draft(rv, biz, phone, ToneWarm, LangHinglish)
		if !strings.Contains(strings.ToLower(hi), "wapas aaiye") {
			t.Errorf("%d-star Hinglish lacks invite-back: %q", rating, hi)
		}
	}
}

func TestKeywordEcho(t *testing.T) {
	d := Template{}
	rv := review(5, "Divya Rao", "The haircut was fabulous and quick")
	got := d.Draft(rv, biz, phone, ToneBrief, LangEnglish)
	if !strings.Contains(got, "haircut") {
		t.Errorf("expected keyword 'haircut' echoed, got %q", got)
	}

	if kw := Keyword("Service accha tha but waiting bahut zyada", LangHinglish); kw != "service" {
		t.Errorf("Keyword() = %q, want 'service' (first domain keyword)", kw)
	}
	if kw := Keyword("!!!", LangEnglish); kw != "experience" {
		t.Errorf("Keyword fallback = %q, want 'experience'", kw)
	}
}

func TestTonesProduceDistinctDrafts(t *testing.T) {
	d := Template{}
	rv := review(5, "Karthik Menon", "Great ambience and friendly staff")
	seen := map[string]string{}
	for _, tone := range []string{ToneProfessional, ToneWarm, ToneBrief} {
		for _, lang := range []string{LangEnglish, LangHinglish} {
			out := d.Draft(rv, biz, phone, tone, lang)
			if prev, dup := seen[out]; dup {
				t.Errorf("tone/lang %s/%s produced same draft as %s", tone, lang, prev)
			}
			seen[out] = tone + "/" + lang
		}
	}
}

func TestDeterministicDrafts(t *testing.T) {
	d := Template{}
	rv := review(2, "Neha Gupta", "Billing was a mess and staff seemed rushed")
	a := d.Draft(rv, biz, phone, ToneWarm, LangHinglish)
	b := d.Draft(rv, biz, phone, ToneWarm, LangHinglish)
	if a != b {
		t.Error("same input produced different drafts")
	}
}

func TestUnknownToneAndLanguageFallBack(t *testing.T) {
	d := Template{}
	rv := review(4, "Isha Das", "Nice quality")
	got := d.Draft(rv, biz, phone, "sassy", "french")
	want := d.Draft(rv, biz, phone, ToneProfessional, LangEnglish)
	if got != want {
		t.Errorf("unknown tone/language should fall back to Professional English")
	}
}
