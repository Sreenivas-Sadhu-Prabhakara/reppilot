// Package drafter turns a review into an owner reply. The default
// implementation is a deterministic template engine; a live implementation
// backed by Claude would be selected when ANTHROPIC_API_KEY is set
// (documented in README; intentionally not implemented).
package drafter

import (
	"strings"

	"reppilot/internal/domain"
)

// Tones and languages the drafter understands.
const (
	ToneProfessional = "Professional"
	ToneWarm         = "Warm"
	ToneBrief        = "Brief"

	LangEnglish  = "English"
	LangHinglish = "Hinglish"
)

// Drafter produces a reply draft for a review.
type Drafter interface {
	Draft(rv domain.Review, businessName, businessPhone, tone, language string) string
	Mode() string
}

// Template is the built-in rating-aware template engine.
type Template struct{}

// Mode reports the drafter mode for /health.
func (Template) Mode() string { return "template" }

// bucket collapses star ratings into a response strategy.
// 1-2 stars: apologize, offer service recovery, take it offline.
// 3 stars: thank + note the improvement.
// 4-5 stars: gratitude + invite back.
func bucket(rating int) string {
	switch {
	case rating <= 2:
		return "negative"
	case rating == 3:
		return "neutral"
	default:
		return "positive"
	}
}

// templates: tone|language|bucket -> reply with {name} {kw} {phone} {biz}.
var templates = map[string]string{
	// --- English ---
	"Professional|English|positive": "Dear {name}, thank you for the wonderful feedback — we are delighted that the {kw} met your expectations. Your support means a great deal to everyone at {biz}, and we look forward to welcoming you back soon.",
	"Warm|English|positive":         "Hi {name}! This truly made our day — so happy you loved the {kw}. Come back and see us at {biz} soon, we'll be waiting!",
	"Brief|English|positive":        "Thanks a ton, {name}! Glad the {kw} hit the mark. See you again at {biz}.",

	"Professional|English|neutral": "Dear {name}, thank you for the honest feedback. We are glad parts of your visit went well, and we are actively improving the {kw}. We hope to earn that fifth star on your next visit to {biz}.",
	"Warm|English|neutral":         "Hi {name}, thanks for being straight with us. We hear you on the {kw} and we're already working on it — give {biz} another chance and let us earn that extra star.",
	"Brief|English|neutral":        "Thanks for the feedback, {name}. We're fixing the {kw} — hope to do better on your next visit to {biz}.",

	"Professional|English|negative": "Dear {name}, please accept our sincere apologies — this is not the standard we hold ourselves to at {biz}, especially regarding the {kw}. We would like to make it right: your next visit is on us. Kindly call us directly at {phone} so we can resolve this personally.",
	"Warm|English|negative":         "Hi {name}, we're really sorry we let you down with the {kw} — that's not the experience anyone should have at {biz}. Your next visit is on us. Please call us at {phone} and we'll make it right, promise.",
	"Brief|English|negative":        "{name}, we're sorry about the {kw}. We'd like to make it right — your next visit is on us. Please call us at {phone}. — Team {biz}",

	// --- Hinglish ---
	"Professional|Hinglish|positive": "Dear {name} ji, aapke kind words ke liye bahut dhanyavaad — khushi hui ki {kw} aapko pasand aaya. {biz} me aapka phir se swagat hai!",
	"Warm|Hinglish|positive":         "{name} ji, dil khush kar diya aapne! {kw} pasand aaya, sunke mazaa aa gaya. Jaldi wapas aaiye {biz} — hum wait karenge!",
	"Brief|Hinglish|positive":        "Shukriya {name} ji! {kw} pasand aaya, sunke accha laga. Phir aaiye {biz}!",

	"Professional|Hinglish|neutral": "Dear {name} ji, aapke honest feedback ke liye dhanyavaad. {kw} par hum kaam kar rahe hain — agli visit par {biz} aapko aur behtar experience dega, yeh hamara vaada hai.",
	"Warm|Hinglish|neutral":         "{name} ji, sach bataane ke liye shukriya! {kw} wali baat note kar li hai, sudhaar shuru ho chuka hai. {biz} ko ek aur mauka dijiye — paanchva star hum kama ke rahenge.",
	"Brief|Hinglish|neutral":        "Thanks {name} ji. {kw} par kaam chal raha hai — agli baar {biz} me behtar milega.",

	"Professional|Hinglish|negative": "Dear {name} ji, humein bahut afsos hai ki aapka experience accha nahi raha, khaas kar {kw} ko lekar. Hum ise theek karna chahte hain — aapki agli visit hum par. Kripya humein {phone} par call karein taaki hum ise personally resolve kar saken.",
	"Warm|Hinglish|negative":         "{name} ji, {kw} ko lekar jo hua uske liye dil se sorry. Aisa experience {biz} me kisi ko nahi milna chahiye. Aapki agli visit hum par — bas ek call kijiye {phone} par, hum sab theek kar denge.",
	"Brief|Hinglish|negative":        "Sorry {name} ji, {kw} ke liye maafi chahte hain. Agli visit hum par — please call kijiye {phone}. — Team {biz}",
}

// Draft renders a personalized reply: reviewer first name + one keyword
// echoed from their review text, shaped by rating bucket, tone and language.
func (Template) Draft(rv domain.Review, businessName, businessPhone, tone, language string) string {
	tone = normalizeTone(tone)
	language = normalizeLanguage(language)
	key := tone + "|" + language + "|" + bucket(rv.Rating)
	tpl := templates[key]

	name := firstName(rv.Reviewer)
	kw := Keyword(rv.Text, language)

	r := strings.NewReplacer(
		"{name}", name,
		"{kw}", kw,
		"{phone}", businessPhone,
		"{biz}", businessName,
	)
	return r.Replace(tpl)
}

func normalizeTone(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "warm":
		return ToneWarm
	case "brief":
		return ToneBrief
	default:
		return ToneProfessional
	}
}

func normalizeLanguage(l string) string {
	if strings.ToLower(strings.TrimSpace(l)) == "hinglish" {
		return LangHinglish
	}
	return LangEnglish
}

func firstName(full string) string {
	fields := strings.Fields(strings.TrimSpace(full))
	if len(fields) == 0 {
		return "there"
	}
	return fields[0]
}

// domainKeywords are echoed preferentially, in review-text order.
var domainKeywords = map[string]bool{
	"service": true, "staff": true, "waiting": true, "wait": true,
	"billing": true, "ambience": true, "hygiene": true, "haircut": true,
	"doctor": true, "taste": true, "food": true, "quality": true,
	"appointment": true, "price": true, "parking": true, "booking": true,
	"cleanliness": true, "experience": true,
}

var stopwords = map[string]bool{
	"the": true, "and": true, "was": true, "but": true, "for": true,
	"with": true, "very": true, "this": true, "that": true, "they": true,
	"were": true, "have": true, "from": true, "will": true, "would": true,
	"bahut": true, "zyada": true, "accha": true, "acchi": true, "achha": true,
	"nahi": true, "thoda": true, "kaafi": true, "bilkul": true, "hain": true,
	"karna": true, "pehle": true, "jaisi": true, "again": true, "place": true,
	"though": true, "totally": true, "worth": true, "there": true, "come": true,
	"back": true, "ever": true, "avoid": true, "minute": true, "minutes": true,
}

// Keyword picks one word from the review text to echo back: a known domain
// keyword if present, else the longest meaningful word, else "experience".
func Keyword(text, language string) string {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9')
	})
	for _, w := range words {
		if domainKeywords[w] {
			return w
		}
	}
	longest := ""
	for _, w := range words {
		if len(w) >= 5 && !stopwords[w] && len(w) > len(longest) {
			longest = w
		}
	}
	if longest != "" {
		return longest
	}
	return "experience"
}
