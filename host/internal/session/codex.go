package session

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ericbosch/cli-remote-control/host/internal/codexrpc"
	"github.com/ericbosch/cli-remote-control/host/internal/events"
)

type codexInitializeParams struct {
	ClientInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
	Capabilities *struct {
		ExperimentalAPI bool `json:"experimentalApi"`
	} `json:"capabilities,omitempty"`
}

type codexThreadStartParams struct {
	ApprovalPolicy string  `json:"approvalPolicy,omitempty"`
	Cwd            *string `json:"cwd,omitempty"`
	Sandbox        *string `json:"sandbox,omitempty"`
}

type codexThreadStartResponse struct {
	Thread struct {
		ID string `json:"id"`
	} `json:"thread"`
}

type codexTurnStartParams struct {
	ThreadID string           `json:"threadId"`
	Input    []codexUserInput `json:"input"`
}

type codexUserInput struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func newCodexSession(ctx context.Context, id, name string, args map[string]interface{}, logDir string, eventsDir string, bufKB int) (*Session, error) {
	if bufKB <= 0 {
		bufKB = defaultBufKB
	}
	ctx, cancel := context.WithCancel(ctx)
	s := &Session{
		ID:        id,
		Name:      name,
		Engine:    "codex",
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

	client, err := codexrpc.Start(ctx)
	if err != nil {
		lf.Close()
		cancel()
		return nil, err
	}
	s.codex = client
	s.cmd = client.Cmd()

	client.SetNotificationHandler(func(method string, params json.RawMessage) {
		switch method {
		case "item/agentMessage/delta":
			var p struct {
				Delta string `json:"delta"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return
			}
			if p.Delta == "" {
				return
			}
			s.writeLegacyOutput([]byte(p.Delta))
			_, _ = s.PublishEvent(events.EventKindAssistant, map[string]any{"data": p.Delta})
		case "item/reasoning/textDelta":
			var p struct {
				Delta string `json:"delta"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return
			}
			if p.Delta == "" {
				return
			}
			_, _ = s.PublishEvent(events.EventKindThinkingDelta, map[string]any{"delta": p.Delta})
		case "item/completed":
			var p struct {
				Item map[string]any `json:"item"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return
			}
			if p.Item == nil {
				return
			}
			if t, _ := p.Item["type"].(string); t == "agentMessage" {
				txt := extractTextFromThreadItem(p.Item)
				if txt != "" {
					s.writeLegacyOutput([]byte(txt))
					_, _ = s.PublishEvent(events.EventKindAssistant, map[string]any{"data": txt})
				}
			}
		case "turn/completed":
			_, _ = s.PublishEvent(events.EventKindThinkingDone, map[string]any{})
		case "error":
			var p struct {
				Error struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(params, &p); err != nil {
				return
			}
			if p.Error.Message != "" {
				_, _ = s.PublishEvent(events.EventKindError, map[string]any{"message": p.Error.Message})
			}
		}
	})

	initCtx, cancelInit := context.WithTimeout(ctx, 10*time.Second)
	defer cancelInit()

	var initParams codexInitializeParams
	initParams.ClientInfo.Name = "cli-remote-control"
	initParams.ClientInfo.Version = "dev"
	initParams.Capabilities = &struct {
		ExperimentalAPI bool `json:"experimentalApi"`
	}{ExperimentalAPI: true}

	var initResp any
	if err := client.Call(initCtx, "initialize", initParams, &initResp); err != nil {
		return nil, err
	}

	workspacePath, _ := args["workspacePath"].(string)
	var threadParams codexThreadStartParams
	threadParams.ApprovalPolicy = "never"
	if workspacePath != "" {
		threadParams.Cwd = &workspacePath
	}
	sandbox := "workspace-write"
	threadParams.Sandbox = &sandbox

	var threadResp codexThreadStartResponse
	if err := client.Call(initCtx, "thread/start", threadParams, &threadResp); err != nil {
		return nil, err
	}
	if threadResp.Thread.ID == "" {
		return nil, errors.New("codex thread/start returned empty thread id")
	}
	s.codexThreadID = threadResp.Thread.ID

	_, _ = s.PublishEvent(events.EventKindStatus, map[string]any{"state": "running"})

	if prompt, _ := args["prompt"].(string); prompt != "" {
		if err := s.codexStartTurn(prompt); err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Session) codexStartTurn(prompt string) error {
	s.mu.RLock()
	client := s.codex
	threadID := s.codexThreadID
	s.mu.RUnlock()
	if client == nil {
		return errors.New("codex client not initialized")
	}
	if threadID == "" {
		return errors.New("codex thread not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	params := codexTurnStartParams{
		ThreadID: threadID,
		Input: []codexUserInput{
			{Type: "text", Text: prompt},
		},
	}
	var resp any
	return client.Call(ctx, "turn/start", params, &resp)
}

func (s *Session) writeLegacyOutput(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	s.mu.Lock()
	if s.ring != nil {
		s.ring.Write(chunk)
	}
	if s.logFile != nil {
		_, _ = s.logFile.Write(chunk)
	}
	for ch := range s.subs {
		select {
		case ch <- chunk:
		default:
		}
	}
	s.mu.Unlock()
}

func extractTextFromThreadItem(item map[string]any) string {
	raw, ok := item["content"]
	if !ok {
		return ""
	}
	parts, ok := raw.([]any)
	if !ok {
		return ""
	}
	out := ""
	for _, p := range parts {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		if m["type"] != "text" {
			continue
		}
		if txt, ok := m["text"].(string); ok {
			out += txt
		}
	}
	return out
}
