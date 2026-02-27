package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const (
	wsTicketTTL = 60 * time.Second
)

type wsTicketManager struct {
	mu      sync.Mutex
	expires map[string]time.Time
}

func newWSTicketManager() *wsTicketManager {
	return &wsTicketManager{
		expires: make(map[string]time.Time),
	}
}

func (m *wsTicketManager) Issue(now time.Time) (ticket string, expiresAt time.Time, err error) {
	// 24 bytes => 32 chars base64url (no padding)
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	ticket = base64.RawURLEncoding.EncodeToString(b)
	expiresAt = now.Add(wsTicketTTL)

	m.mu.Lock()
	m.expires[ticket] = expiresAt
	m.mu.Unlock()

	return ticket, expiresAt, nil
}

// Consume validates and deletes a ticket (single-use).
func (m *wsTicketManager) Consume(ticket string, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	exp, ok := m.expires[ticket]
	if !ok {
		return false
	}
	delete(m.expires, ticket)
	return now.Before(exp)
}

func (s *Server) issueWSTicket(w http.ResponseWriter, r *http.Request) {
	ticket, exp, err := s.tickets.Issue(time.Now())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ticket":     ticket,
		"expires_ms": exp.UnixMilli(),
	})
}
