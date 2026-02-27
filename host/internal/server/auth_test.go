package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware_BearerToken(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "t0k", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	h := corsMiddleware(s.mux)

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://example/api/sessions", nil)
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("unauth status=%d", rr.Code)
		}
	}

	{
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://example/api/sessions", nil)
		req.Header.Set("Authorization", "Bearer t0k")
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("auth status=%d", rr.Code)
		}
	}
}

func TestAuthMiddleware_RawToken(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "raw", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	h := corsMiddleware(s.mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example/api/sessions", nil)
	req.Header.Set("Authorization", "raw")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("auth status=%d", rr.Code)
	}
}

func TestAuthMiddleware_EmptyConfigTokenRejectsAll(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	h := corsMiddleware(s.mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example/api/sessions", nil)
	h.ServeHTTP(rr, req)
	if rr.Code < 500 {
		t.Fatalf("expected 5xx for empty token, got %d", rr.Code)
	}
}

func TestAPI_DoesNotAcceptQueryToken(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "q", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	h := corsMiddleware(s.mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example/api/sessions?token=q", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHealthz_NoAuth(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "x", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	h := corsMiddleware(s.mux)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example/healthz", nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("healthz status=%d", rr.Code)
	}
}
