// Package store is an in-memory state container guarded by a mutex, with
// JSON snapshot persistence to disk after every write.
package store

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"reppilot/internal/domain"
)

// State is everything RepPilot remembers between restarts.
type State struct {
	Profile   *domain.Profile         `json:"profile,omitempty"`
	Reviews   []*domain.Review        `json:"reviews"`
	Campaigns []*domain.Campaign      `json:"campaigns"`
	Outbox    []*domain.OutboxMessage `json:"outbox"`
	Seq       int                     `json:"seq"`
}

// NextID mints a sequential ID like "cmp-3". Call only inside Update.
func (st *State) NextID(prefix string) string {
	st.Seq++
	return fmt.Sprintf("%s-%d", prefix, st.Seq)
}

// FindReview returns the review with the given ID, or nil.
func (st *State) FindReview(id string) *domain.Review {
	for _, rv := range st.Reviews {
		if rv.ID == id {
			return rv
		}
	}
	return nil
}

// Store guards State with a mutex and snapshots it to path on every write.
type Store struct {
	mu   sync.Mutex
	path string
	st   State
}

// Open loads a snapshot from path if one exists; a corrupt or missing file
// starts fresh (the mock provider can regenerate everything).
func Open(path string) *Store {
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	if err == nil {
		if uerr := json.Unmarshal(data, &s.st); uerr != nil {
			log.Printf("store: snapshot %s unreadable (%v); starting fresh", path, uerr)
			s.st = State{}
		}
	}
	return s
}

// View runs fn with read access under the lock.
func (s *Store) View(fn func(st *State)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn(&s.st)
}

// Update runs fn under the lock and saves a snapshot afterwards.
func (s *Store) Update(fn func(st *State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.st); err != nil {
		return err
	}
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.st, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
