//go:build codex_smoke

package session

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ericbosch/cli-remote-control/host/internal/events"
)

func TestCodexAppServerReplyOK(t *testing.T) {
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex not installed")
	}

	logDir := filepath.Join(t.TempDir(), "logs")
	eventsDir := filepath.Join(t.TempDir(), "events")

	args := map[string]interface{}{
		"workspacePath": filepath.Join("..", ".."),
		"prompt":        "Reply ONLY with OK",
	}
	s, err := NewSession(context.Background(), "codex-smoke", "codex-smoke", "codex", args, logDir, eventsDir, 64)
	if err != nil {
		t.Fatalf("NewSession(codex): %v", err)
	}
	defer s.Terminate()

	ch := s.SubscribeEvents()
	defer s.UnsubscribeEvents(ch)

	deadline := time.After(90 * time.Second)
	buf := strings.Builder{}
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for OK; got=%q", buf.String())
		case ev := <-ch:
			if ev.Kind != events.EventKindAssistant {
				continue
			}
			var p struct {
				Data string `json:"data"`
			}
			_ = json.Unmarshal(ev.Payload, &p)
			if p.Data != "" {
				buf.WriteString(p.Data)
				if strings.Contains(buf.String(), "OK") {
					return
				}
			}
		}
	}
}
