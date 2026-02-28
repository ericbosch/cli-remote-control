package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSPAFallbackServesIndexForDeepLink(t *testing.T) {
	dir := t.TempDir()
	index := "<!doctype html><html><body><div id=\"root\"></div></body></html>"
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(index), 0600); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0700); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log('ok')"), 0600); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	h := spaFileServer(dir)

	req := httptest.NewRequest(http.MethodGet, "http://example/sessions/abc123", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(strings.ToLower(ct), "text/html") {
		t.Fatalf("expected html content-type, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "id=\"root\"") {
		t.Fatalf("expected index.html body, got %q", rr.Body.String())
	}
}

func TestSPAFallbackDoesNotServeIndexForMissingAsset(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("index"), 0600); err != nil {
		t.Fatalf("write index.html: %v", err)
	}

	h := spaFileServer(dir)

	req := httptest.NewRequest(http.MethodGet, "http://example/assets/missing.js", nil)
	req.Header.Set("Accept", "*/*")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

