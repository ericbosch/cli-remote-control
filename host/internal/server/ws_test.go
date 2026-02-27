package server

import (
	"encoding/json"
	"testing"
)

func TestServerMsgJSON(t *testing.T) {
	m := serverMsg{Type: "output", Stream: "stdout", Data: "hello"}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var out serverMsg
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Type != "output" || out.Stream != "stdout" || out.Data != "hello" {
		t.Errorf("roundtrip: %+v", out)
	}
}

func TestClientMsgJSON(t *testing.T) {
	m := clientMsg{Type: "input", Data: "ls\n"}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var out clientMsg
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Type != "input" || out.Data != "ls\n" {
		t.Errorf("roundtrip: %+v", out)
	}
}
