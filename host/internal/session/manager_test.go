package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestManager_CreateListTerminate(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir, 8, filepath.Join(t.TempDir(), "events"))
	ctx := context.Background()

	s1, err := m.Create(ctx, "shell", "s1", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if s1.ID == "" {
		t.Error("expected non-empty ID")
	}
	if s1.Engine != "shell" {
		t.Errorf("engine: got %s", s1.Engine)
	}

	list := m.List()
	if len(list) != 1 {
		t.Errorf("list: got %d sessions", len(list))
	}

	err = m.Terminate(s1.ID)
	if err != nil {
		t.Errorf("terminate: %v", err)
	}

	list = m.List()
	if len(list) != 0 {
		t.Errorf("after terminate: got %d sessions", len(list))
	}

	got := m.Get(s1.ID)
	if got != nil {
		t.Error("Get after terminate should return nil")
	}
}

func TestManager_TerminateNotFound(t *testing.T) {
	dir := t.TempDir()
	m := NewManager(dir, 8, filepath.Join(t.TempDir(), "events"))
	err := m.Terminate("nonexistent")
	if err != ErrNotFound {
		t.Errorf("terminate nonexistent: got %v", err)
	}
}

func TestIdString(t *testing.T) {
	if idString(1) != "1" {
		t.Errorf("idString(1)=%s", idString(1))
	}
	if idString(36) != "10" {
		t.Errorf("idString(36)=%s", idString(36))
	}
}

func TestNewManager_CreatesLogDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "logs")
	m := NewManager(dir, 8, filepath.Join(t.TempDir(), "events"))
	ctx := context.Background()
	s, err := m.Create(ctx, "shell", "", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer m.Terminate(s.ID)
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("log dir not created: %v", err)
	}
}
