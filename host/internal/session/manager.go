package session

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Manager creates and tracks sessions.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	nextID   atomic.Uint64
	logDir   string
	bufKB    int
}

// NewManager creates a session manager. bufKB is the ring buffer size per session in KB.
func NewManager(logDir string, bufKB int) *Manager {
	if bufKB <= 0 {
		bufKB = 64
	}
	return &Manager{
		sessions: make(map[string]*Session),
		logDir:   logDir,
		bufKB:    bufKB,
	}
}

// Create starts a new session with the given engine and optional name.
func (m *Manager) Create(ctx context.Context, engine, name string, args map[string]interface{}) (*Session, error) {
	id := m.nextID()
	sid := idString(id)
	if name == "" {
		name = "session-" + sid
	}
	s, err := NewSession(ctx, sid, name, engine, args, m.logDir, m.bufKB)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.sessions[sid] = s
	m.mu.Unlock()
	go s.Run()
	return s, nil
}

// Get returns a session by ID or nil.
func (m *Manager) Get(id string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[id]
}

// List returns a snapshot of all sessions.
func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s)
	}
	return out
}

// Terminate stops a session and removes it from the manager.
func (m *Manager) Terminate(id string) error {
	m.mu.Lock()
	s := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if s == nil {
		return ErrNotFound
	}
	return s.Terminate()
}

func (m *Manager) nextID() uint64 {
	return m.nextID.Add(1)
}

func idString(n uint64) string {
	return strconv.FormatUint(n, 36)
}
