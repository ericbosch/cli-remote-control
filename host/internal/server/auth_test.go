package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware_BearerToken(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "t0k", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(corsMiddleware(s.mux))
	t.Cleanup(ts.Close)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
	req.Header.Set("Authorization", "Bearer t0k")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth status=%d", resp.StatusCode)
	}
}

func TestAuthMiddleware_RawToken(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "raw", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(corsMiddleware(s.mux))
	t.Cleanup(ts.Close)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
	req.Header.Set("Authorization", "raw")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("auth status=%d", resp.StatusCode)
	}
}

func TestAuthMiddleware_EmptyConfigTokenRejectsAll(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(corsMiddleware(s.mux))
	t.Cleanup(ts.Close)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 500 {
		t.Fatalf("expected 5xx for empty token, got %d", resp.StatusCode)
	}
}

func TestAPI_DoesNotAcceptQueryToken(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "q", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(corsMiddleware(s.mux))
	t.Cleanup(ts.Close)

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/sessions?token=q", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHealthz_NoAuth(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "x", LogDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(corsMiddleware(s.mux))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("healthz status=%d", resp.StatusCode)
	}
}
