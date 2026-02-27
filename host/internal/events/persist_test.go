package events

import (
	"path/filepath"
	"testing"
)

func TestJSONLStoreAppendAndLoadTail(t *testing.T) {
	dir := t.TempDir()
	store, err := NewJSONLStore(dir)
	if err != nil {
		t.Fatalf("NewJSONLStore: %v", err)
	}

	sid := "sess1"
	for i := 0; i < 5; i++ {
		ev := SessionEvent{SessionID: sid, Engine: "shell", TsMS: int64(100 + i), Seq: uint64(i + 1), Kind: EventKindAssistant}
		if err := store.Append(sid, ev); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	tail, err := store.LoadTail(sid, 2)
	if err != nil {
		t.Fatalf("LoadTail: %v", err)
	}
	if len(tail) != 2 {
		t.Fatalf("tail len=%d want=2", len(tail))
	}
	if tail[0].Seq != 4 || tail[1].Seq != 5 {
		t.Fatalf("tail seqs=%d,%d want=4,5", tail[0].Seq, tail[1].Seq)
	}

	if _, err := NewJSONLStore(filepath.Join(dir, "nested")); err != nil {
		t.Fatalf("NewJSONLStore nested: %v", err)
	}
}
