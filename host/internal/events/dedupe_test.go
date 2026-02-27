package events

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type cursorNDJSON struct {
	Type        string `json:"type"`
	SessionID   string `json:"session_id"`
	TimestampMS *int64 `json:"timestamp_ms"`
	Message     struct {
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

func findRepoRootFromThisFile(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(filename)
	for i := 0; i < 10; i++ {
		if st, err := os.Stat(filepath.Join(dir, ".git")); err == nil && st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("repo root not found")
	return ""
}

func TestDeduperFiltersDuplicateAssistantFromFixture(t *testing.T) {
	root := findRepoRootFromThisFile(t)
	path := filepath.Join(root, "fixtures", "cursor-sample.full.ndjson")

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	d := NewDeduper(1024, DedupeOptions{IncludeTimestampMS: false})

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	assistantCount := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var row cursorNDJSON
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		if row.Type != "assistant" {
			continue
		}
		txt := ""
		for _, c := range row.Message.Content {
			if c.Type == "text" {
				txt = c.Text
				break
			}
		}
		payload, err := MarshalPayload(map[string]any{"text": txt})
		if err != nil {
			t.Fatalf("payload: %v", err)
		}
		ev := SessionEvent{
			SessionID: row.SessionID,
			Engine:    "cursor",
			TsMS:      NowMS(),
			Kind:      EventKindAssistant,
			Payload:   payload,
		}

		if d.Seen(ev) {
			continue
		}
		assistantCount++
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if assistantCount != 1 {
		t.Fatalf("assistant events after dedupe=%d want=1", assistantCount)
	}
}
