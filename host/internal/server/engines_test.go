package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnginesIncludesShell(t *testing.T) {
	s, err := New(Config{Bind: "127.0.0.1", Port: "0", Token: "t", LogDir: t.TempDir()})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	ts := httptest.NewServer(s.mux)
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/engines", nil)
	req.Header.Set("Authorization", "Bearer t")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var engines []string
	if err := json.NewDecoder(res.Body).Decode(&engines); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, e := range engines {
		if e == "shell" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected engines to include shell, got %#v", engines)
	}
}
