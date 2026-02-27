package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWSTicketManager_SingleUseAndExpiry(t *testing.T) {
	m := newWSTicketManager()
	now := time.Unix(100, 0)

	ticket, exp, err := m.Issue(now)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if ticket == "" {
		t.Fatal("expected non-empty ticket")
	}
	if exp.Before(now) {
		t.Fatalf("expected exp after now, got %v", exp)
	}

	if ok := m.Consume(ticket, now); !ok {
		t.Fatal("expected Consume ok")
	}
	if ok := m.Consume(ticket, now); ok {
		t.Fatal("expected ticket to be single-use")
	}

	t2, _, err := m.Issue(now)
	if err != nil {
		t.Fatalf("Issue2: %v", err)
	}
	if ok := m.Consume(t2, now.Add(wsTicketTTL).Add(1*time.Millisecond)); ok {
		t.Fatal("expected expired ticket to fail")
	}
}

func TestAPI_IssueWSTicket_RequiresAuthAndWorks(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "t0k", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	h := s.authMiddleware(false, http.HandlerFunc(s.handleAPI))

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "http://example/api/ws-ticket", nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("missing auth: status=%d body=%s", rr.Code, rr.Body.String())
		}
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "http://example/api/ws-ticket", nil)
	req.Header.Set("Authorization", "Bearer t0k")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Ticket    string `json:"ticket"`
		ExpiresMS int64  `json:"expires_ms"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if resp.Ticket == "" {
		t.Fatal("expected ticket")
	}
	if resp.ExpiresMS == 0 {
		t.Fatal("expected expires_ms")
	}
	if ok := s.tickets.Consume(resp.Ticket, time.Now()); !ok {
		t.Fatal("expected issued ticket to be valid")
	}
}
