package session

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ericbosch/cli-remote-control/host/internal/events"
)

type cursorNDJSONRow struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	Text      string `json:"text,omitempty"`
	SessionID string `json:"session_id,omitempty"`

	TimestampMS *int64 `json:"timestamp_ms,omitempty"`

	Message struct {
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
	} `json:"message,omitempty"`
}

func newCursorSession(ctx context.Context, id, name string, args map[string]interface{}, logDir string, eventsDir string, bufKB int) (*Session, error) {
	s, err := newCursorNDJSONSession(ctx, id, name, args, logDir, eventsDir, bufKB)
	if err == nil {
		return s, nil
	}
	log.Printf("cursor NDJSON engine unavailable (%v); falling back to cursor PTY", err)
	return newCursorPTYSession(ctx, id, name, args, logDir, eventsDir, bufKB)
}

func newCursorNDJSONSession(ctx context.Context, id, name string, args map[string]interface{}, logDir string, eventsDir string, bufKB int) (*Session, error) {
	prompt, _ := args["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, errors.New("no prompt provided for NDJSON mode")
	}

	if bufKB <= 0 {
		bufKB = defaultBufKB
	}
	ctx, cancel := context.WithCancel(ctx)
	s := &Session{
		ID:        id,
		Name:      name,
		Engine:    "cursor",
		Created:   time.Now(),
		state:     "running",
		ring:      NewRingBuffer(bufKB * 1024),
		eventsBuf: events.NewBuffer(2048),
		subs:      make(map[chan []byte]struct{}),
		eventSubs: make(map[chan events.SessionEvent]struct{}),
		cancel:    cancel,
		done:      make(chan struct{}),
	}
	if eventsDir != "" {
		if store, err := events.NewJSONLStore(eventsDir); err == nil {
			s.eventsStore = store
		} else {
			log.Printf("events persistence disabled (dir=%s): %v", eventsDir, err)
		}
	}
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		cancel()
		return nil, err
	}
	logPath := filepath.Join(logDir, id+".log")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		cancel()
		return nil, err
	}
	s.logFile = lf

	workspacePath, _ := args["workspacePath"].(string)

	bin := "cursor"
	cmdArgs := []string{"agent", "--print", "--output-format", "stream-json", "--stream-partial-output", prompt}
	if _, err := exec.LookPath(bin); err != nil {
		bin = "agent"
		cmdArgs = []string{"--print", "--output-format", "stream-json", "--stream-partial-output", prompt}
		if _, err := exec.LookPath(bin); err != nil {
			lf.Close()
			cancel()
			return nil, errors.New("cursor agent binary not found")
		}
	}

	cmd := exec.CommandContext(ctx, bin, cmdArgs...)
	if workspacePath != "" {
		cmd.Dir = workspacePath
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		lf.Close()
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		lf.Close()
		cancel()
		return nil, err
	}

	s.cmd = cmd
	if err := cmd.Start(); err != nil {
		lf.Close()
		cancel()
		return nil, err
	}

	_, _ = s.PublishEvent(events.EventKindStatus, map[string]any{"state": "running"})

	deduper := events.NewDeduper(4096, events.DedupeOptions{IncludeTimestampMS: false})
	go s.readCursorNDJSON(stdout, deduper)
	go s.readCursorStderr(stderr)
	return s, nil
}

func (s *Session) readCursorStderr(r io.Reader) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		_, _ = s.PublishEvent(events.EventKindError, map[string]any{"message": line})
	}
}

func (s *Session) readCursorNDJSON(r io.Reader, deduper *events.Deduper) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		var row cursorNDJSONRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			_, _ = s.PublishEvent(events.EventKindError, map[string]any{"message": "invalid NDJSON line"})
			continue
		}

		switch row.Type {
		case "thinking":
			if row.Subtype == "delta" && row.Text != "" {
				_, _ = s.PublishEvent(events.EventKindThinkingDelta, map[string]any{"delta": row.Text})
			}
			if row.Subtype == "completed" {
				_, _ = s.PublishEvent(events.EventKindThinkingDone, map[string]any{})
			}
		case "assistant":
			txt := cursorExtractText(row)
			if strings.TrimSpace(txt) == "" {
				continue
			}

			raw, err := events.MarshalPayload(map[string]any{"data": txt})
			if err != nil {
				continue
			}
			probe := events.SessionEvent{
				SessionID: s.ID,
				Engine:    s.Engine,
				TsMS:      events.NowMS(),
				Kind:      events.EventKindAssistant,
				Payload:   raw,
			}
			if deduper.Seen(probe) {
				continue
			}

			s.writeLegacyOutput([]byte(txt))
			_, _ = s.PublishEvent(events.EventKindAssistant, map[string]any{"data": txt})
		}
	}
}

func cursorExtractText(row cursorNDJSONRow) string {
	if len(row.Message.Content) == 0 {
		return ""
	}
	out := strings.Builder{}
	for _, c := range row.Message.Content {
		if c.Type == "text" && c.Text != "" {
			out.WriteString(c.Text)
		}
	}
	return out.String()
}
