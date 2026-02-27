package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ericbosch/cli-remote-control/host/internal/events"
)

func findRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(filename)
	for i := 0; i < 12; i++ {
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

func TestCursorFixtureMapsAndDedupesAssistant(t *testing.T) {
	root := findRepoRoot(t)
	path := filepath.Join(root, "fixtures", "cursor-sample.full.ndjson")
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	d := events.NewDeduper(1024, events.DedupeOptions{IncludeTimestampMS: false})

	assistant := 0
	thinkingDeltas := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var row cursorNDJSONRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			continue
		}
		if row.Type == "thinking" && row.Subtype == "delta" && row.Text != "" {
			thinkingDeltas++
		}
		if row.Type != "assistant" {
			continue
		}
		txt := cursorExtractText(row)
		if strings.TrimSpace(txt) == "" {
			continue
		}
		raw, err := events.MarshalPayload(map[string]any{"data": txt})
		if err != nil {
			t.Fatalf("payload: %v", err)
		}
		ev := events.SessionEvent{SessionID: "sess", Engine: "cursor", TsMS: events.NowMS(), Kind: events.EventKindAssistant, Payload: raw}
		if d.Seen(ev) {
			continue
		}
		assistant++
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if thinkingDeltas == 0 {
		t.Fatalf("expected thinking deltas > 0")
	}
	if assistant != 1 {
		t.Fatalf("assistant after dedupe=%d want=1", assistant)
	}
}
