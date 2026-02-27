package events

import (
	"encoding/json"
	"errors"
	"time"
)

type EventKind string

const (
	EventKindSystem        EventKind = "system"
	EventKindUser          EventKind = "user"
	EventKindAssistant     EventKind = "assistant"
	EventKindThinkingDelta EventKind = "thinking_delta"
	EventKindThinkingDone  EventKind = "thinking_done"
	EventKindToolCall      EventKind = "tool_call"
	EventKindToolOutput    EventKind = "tool_output"
	EventKindStatus        EventKind = "status"
	EventKindError         EventKind = "error"
	EventKindMetrics       EventKind = "metrics"
)

type SessionEvent struct {
	SessionID string          `json:"session_id"`
	Engine    string          `json:"engine"`
	TsMS      int64           `json:"ts_ms"`
	Seq       uint64          `json:"seq"`
	Kind      EventKind       `json:"kind"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

func (e SessionEvent) Validate() error {
	if e.SessionID == "" {
		return errors.New("session_id required")
	}
	if e.Engine == "" {
		return errors.New("engine required")
	}
	if e.Kind == "" {
		return errors.New("kind required")
	}
	if e.TsMS <= 0 {
		return errors.New("ts_ms must be set")
	}
	if e.Seq == 0 {
		return errors.New("seq must be set")
	}
	return nil
}

func NowMS() int64 {
	return time.Now().UnixMilli()
}

func MarshalPayload(v any) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
