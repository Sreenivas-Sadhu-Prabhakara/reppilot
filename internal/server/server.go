// Package server wires the JSON API and the embedded web UI.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"

	"reppilot/internal/campaign"
	"reppilot/internal/digest"
	"reppilot/internal/domain"
	"reppilot/internal/drafter"
	"reppilot/internal/gbp"
	"reppilot/internal/store"
	"reppilot/internal/wa"
)

// Server holds the app's dependencies.
type Server struct {
	store   *store.Store
	gbp     gbp.Provider
	drafter drafter.Drafter
	sender  wa.Sender
	webFS   fs.FS
}

// New builds a Server.
func New(st *store.Store, provider gbp.Provider, d drafter.Drafter, sender wa.Sender, webFS fs.FS) *Server {
	return &Server{store: st, gbp: provider, drafter: d, sender: sender, webFS: webFS}
}

// Handler returns the routed http.Handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("POST /api/v1/profile/connect", s.handleConnect)
	mux.HandleFunc("GET /api/v1/profile", s.handleProfile)
	mux.HandleFunc("GET /api/v1/reviews", s.handleReviews)
	mux.HandleFunc("POST /api/v1/reviews/{id}/draft", s.handleDraft)
	mux.HandleFunc("POST /api/v1/reviews/{id}/reply", s.handleReply)
	mux.HandleFunc("POST /api/v1/reviews/draft-all", s.handleDraftAll)
	mux.HandleFunc("POST /api/v1/campaigns", s.handleCreateCampaign)
	mux.HandleFunc("GET /api/v1/campaigns", s.handleListCampaigns)
	mux.HandleFunc("GET /api/v1/outbox", s.handleOutbox)
	mux.HandleFunc("GET /api/v1/digest", s.handleDigest)
	mux.HandleFunc("POST /api/v1/digest/send", s.handleDigestSend)

	mux.Handle("GET /", http.FileServerFS(s.webFS))

	return logRequests(mux)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Microsecond))
	})
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	if err := dec.Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

var errNoProfile = errors.New("no profile connected yet — POST /api/v1/profile/connect first")

// ---------- handlers ----------

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":          "ok",
		"gbp_provider":    s.gbp.Mode(),
		"reply_drafter":   s.drafter.Mode(),
		"whatsapp_sender": s.sender.Mode(),
	})
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BusinessName string `json:"business_name"`
		City         string `json:"city"`
		Category     string `json:"category"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.BusinessName == "" || req.City == "" || req.Category == "" {
		writeErr(w, http.StatusBadRequest, "business_name, city and category are required")
		return
	}

	profile, reviews := s.gbp.Connect(req.BusinessName, req.City, req.Category)
	var out domain.Profile
	err := s.store.Update(func(st *store.State) error {
		st.Profile = &profile
		st.Reviews = make([]*domain.Review, len(reviews))
		for i := range reviews {
			rv := reviews[i]
			st.Reviews[i] = &rv
		}
		out = profile
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"profile":          out,
		"reviews_imported": len(reviews),
	})
}

func (s *Server) handleProfile(w http.ResponseWriter, _ *http.Request) {
	var resp map[string]any
	s.store.View(func(st *store.State) {
		if st.Profile == nil {
			resp = map[string]any{"connected": false}
			return
		}
		unanswered := 0
		for _, rv := range st.Reviews {
			if !rv.Replied {
				unanswered++
			}
		}
		resp = map[string]any{
			"connected":  true,
			"profile":    st.Profile,
			"tracked":    len(st.Reviews),
			"unanswered": unanswered,
		}
	})
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleReviews(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("filter") // "", all, unanswered, answered
	ratingStr := r.URL.Query().Get("rating")
	rating := 0
	if ratingStr != "" {
		n, err := strconv.Atoi(ratingStr)
		if err != nil || n < 1 || n > 5 {
			writeErr(w, http.StatusBadRequest, "rating must be 1-5")
			return
		}
		rating = n
	}

	out := []*domain.Review{}
	counts := map[string]int{"total": 0, "unanswered": 0, "answered": 0}
	s.store.View(func(st *store.State) {
		for _, rv := range st.Reviews {
			counts["total"]++
			if rv.Replied {
				counts["answered"]++
			} else {
				counts["unanswered"]++
			}
			if filter == "unanswered" && rv.Replied {
				continue
			}
			if filter == "answered" && !rv.Replied {
				continue
			}
			if rating != 0 && rv.Rating != rating {
				continue
			}
			c := *rv
			out = append(out, &c)
		}
	})
	writeJSON(w, http.StatusOK, map[string]any{"reviews": out, "counts": counts})
}

func (s *Server) handleDraft(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Tone     string `json:"tone"`
		Language string `json:"language"`
	}
	if !readJSON(w, r, &req) {
		return
	}

	var draft string
	err := s.store.Update(func(st *store.State) error {
		if st.Profile == nil {
			return errNoProfile
		}
		rv := st.FindReview(id)
		if rv == nil {
			return fmt.Errorf("review %s not found", id)
		}
		draft = s.drafter.Draft(*rv, st.Profile.BusinessName, st.Profile.Phone, req.Tone, req.Language)
		rv.Draft = draft
		return nil
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errNoProfile) {
			status = http.StatusConflict
		}
		writeErr(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": id, "draft": draft})
}

func (s *Server) handleReply(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Reply string `json:"reply"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Reply == "" {
		writeErr(w, http.StatusBadRequest, "reply text is required")
		return
	}

	var out domain.Review
	err := s.store.Update(func(st *store.State) error {
		rv := st.FindReview(id)
		if rv == nil {
			return fmt.Errorf("review %s not found", id)
		}
		now := time.Now().UTC()
		rv.Replied = true
		rv.Reply = req.Reply
		rv.RepliedAt = &now
		rv.Draft = ""
		out = *rv
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleDraftAll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tone     string `json:"tone"`
		Language string `json:"language"`
	}
	if !readJSON(w, r, &req) {
		return
	}

	drafted := 0
	err := s.store.Update(func(st *store.State) error {
		if st.Profile == nil {
			return errNoProfile
		}
		for _, rv := range st.Reviews {
			if rv.Replied {
				continue
			}
			rv.Draft = s.drafter.Draft(*rv, st.Profile.BusinessName, st.Profile.Phone, req.Tone, req.Language)
			drafted++
		}
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"drafted": drafted})
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Customers string `json:"customers"` // one "Name, +91-98xxxxxxxx" per line
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "campaign name is required")
		return
	}
	customers, skipped := campaign.ParseCustomers(req.Customers)
	if len(customers) == 0 {
		writeErr(w, http.StatusBadRequest, "no valid customers found — use one \"Name, +91-98xxxxxxxx\" per line")
		return
	}

	var created domain.Campaign
	err := s.store.Update(func(st *store.State) error {
		if st.Profile == nil {
			return errNoProfile
		}
		cmp := domain.Campaign{
			ID:        st.NextID("cmp"),
			Name:      req.Name,
			CreatedAt: time.Now().UTC(),
			Customers: customers,
			Skipped:   skipped,
		}
		for _, c := range customers {
			body := campaign.BuildMessage(c, st.Profile.BusinessName, st.Profile.City, st.Profile.ReviewLink)
			msg := s.sender.Send("campaign", c.Phone, c.Name, body)
			msg.ID = st.NextID("msg")
			st.Outbox = append(st.Outbox, &msg)
			cmp.Sent++
		}
		st.Campaigns = append(st.Campaigns, &cmp)
		created = cmp
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, _ *http.Request) {
	out := []*domain.Campaign{}
	s.store.View(func(st *store.State) {
		for i := len(st.Campaigns) - 1; i >= 0; i-- {
			c := *st.Campaigns[i]
			out = append(out, &c)
		}
	})
	writeJSON(w, http.StatusOK, map[string]any{"campaigns": out})
}

func (s *Server) handleOutbox(w http.ResponseWriter, _ *http.Request) {
	out := []*domain.OutboxMessage{}
	s.store.View(func(st *store.State) {
		for i := len(st.Outbox) - 1; i >= 0; i-- {
			m := *st.Outbox[i]
			out = append(out, &m)
		}
	})
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}

func (s *Server) buildDigest() (digest.Digest, error) {
	var d digest.Digest
	var err error
	s.store.View(func(st *store.State) {
		if st.Profile == nil {
			err = errNoProfile
			return
		}
		d = digest.Build(*st.Profile, st.Reviews, gbp.Anchor)
	})
	return d, err
}

func (s *Server) handleDigest(w http.ResponseWriter, _ *http.Request) {
	d, err := s.buildDigest()
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleDigestSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Phone string `json:"phone"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	phone, ok := campaign.NormalizePhone(req.Phone)
	if !ok {
		writeErr(w, http.StatusBadRequest, "phone must be a valid Indian mobile like +91-9812345678")
		return
	}
	d, err := s.buildDigest()
	if err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}

	var queued domain.OutboxMessage
	err = s.store.Update(func(st *store.State) error {
		msg := s.sender.Send("digest", phone, "Owner", d.WhatsAppText())
		msg.ID = st.NextID("msg")
		st.Outbox = append(st.Outbox, &msg)
		queued = msg
		return nil
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, queued)
}
