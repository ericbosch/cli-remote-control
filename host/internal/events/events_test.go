package events

import (
	"encoding/json"
	"testing"
)

func TestSessionEventJSONRoundTrip(t *testing.T) {
	payload, err := MarshalPayload(map[string]any{"text": "hi", "n": 1})
	if err != nil {
		t.Fatalf("payload: %v", err)
	}
	ev := SessionEvent{
		SessionID: "s1",
		Engine:    "cursor",
		TsMS:      123,
		Seq:       7,
		Kind:      EventKindAssistant,
		Payload:   payload,
	}
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got SessionEvent
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.SessionID != ev.SessionID || got.Engine != ev.Engine || got.TsMS != ev.TsMS || got.Seq != ev.Seq || got.Kind != ev.Kind {
		t.Fatalf("roundtrip mismatch: got=%+v want=%+v", got, ev)
	}
	if string(got.Payload) != string(ev.Payload) {
		t.Fatalf("payload mismatch: got=%s want=%s", string(got.Payload), string(ev.Payload))
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}
