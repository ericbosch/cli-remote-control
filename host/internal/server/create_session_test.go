package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCreateSessionInvalidWorkspaceReturns400JSON(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "t", LogDir: t.TempDir()})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	ts := httptest.NewServer(s.mux)
	defer ts.Close()

	body := map[string]any{
		"engine":        "shell",
		"name":          "x",
		"workspacePath": "/__does_not_exist__",
		"prompt":        "",
		"mode":          "",
		"args":          map[string]any{},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/sessions", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer t")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.StatusCode)
	}
	var env apiErrorEnvelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error.Code != "invalid_workspace" {
		t.Fatalf("expected code invalid_workspace, got %q", env.Error.Code)
	}
	if env.Error.RequestID == "" {
		t.Fatalf("expected request_id")
	}
}

func TestCreateCodexSessionMissingCodexReturns424JSON(t *testing.T) {
	origPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })
	_ = os.Setenv("PATH", "")

	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "t", LogDir: t.TempDir()})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	ts := httptest.NewServer(s.mux)
	defer ts.Close()

	body := map[string]any{
		"engine":        "codex",
		"name":          "x",
		"workspacePath": "",
		"prompt":        "",
		"mode":          "",
		"args":          map[string]any{},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/sessions", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer t")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusFailedDependency {
		t.Fatalf("expected 424, got %d", res.StatusCode)
	}
	var env apiErrorEnvelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error.Code == "" || env.Error.Message == "" || env.Error.RequestID == "" {
		t.Fatalf("missing fields: %+v", env.Error)
	}
	if strings.Contains(strings.ToLower(env.Error.Message), "bearer ") {
		t.Fatalf("unexpected secret in message")
	}
}

