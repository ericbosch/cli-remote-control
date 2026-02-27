package session

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/ericbosch/cli-remote-control/host/internal/events"
)

const defaultBufKB = 64

// Session represents a single PTY session (e.g. shell).
type Session struct {
	ID      string
	Name    string
	Engine  string
	Created time.Time

	mu          sync.RWMutex
	state       string // "running", "exited"
	exitCode    int
	ptmx        *os.File
	cmd         *exec.Cmd
	logFile     *os.File
	ring        *RingBuffer
	eventsBuf   *events.Buffer
	eventsStore *events.JSONLStore
	subs        map[chan []byte]struct{}
	eventSubs   map[chan events.SessionEvent]struct{}
	closed      bool
	cancel      context.CancelFunc
	done        chan struct{}
}

// NewSession creates a session for the given engine. Caller must call Run().
func NewSession(ctx context.Context, id, name, engine string, args map[string]interface{}, logDir string, eventsDir string, bufKB int) (*Session, error) {
	switch engine {
	case "cursor":
		s, err := newCursorSession(ctx, id, name, args, logDir, eventsDir, bufKB)
		if err != nil {
			log.Printf("cursor engine unavailable (%v); falling back to shell PTY mock", err)
			return newShellSession(ctx, id, name, "cursor-mock", logDir, eventsDir, bufKB)
		}
		return s, nil
	default:
		return newShellSession(ctx, id, name, engine, logDir, eventsDir, bufKB)
	}
}

// newShellSession starts a bash shell in a PTY.
func newShellSession(ctx context.Context, id, name, engine, logDir string, eventsDir string, bufKB int) (*Session, error) {
	if bufKB <= 0 {
		bufKB = defaultBufKB
	}
	ctx, cancel := context.WithCancel(ctx)
	s := &Session{
		ID:        id,
		Name:      name,
		Engine:    engine,
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
	if err := os.MkdirAll(logDir, 0750); err != nil {
		cancel()
		return nil, err
	}
	logPath := filepath.Join(logDir, id+".log")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		cancel()
		return nil, err
	}
	s.logFile = lf
	cmd := exec.CommandContext(ctx, "bash")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	ptmx, err := pty.Start(cmd)
	if err != nil {
		lf.Close()
		cancel()
		return nil, err
	}
	s.ptmx = ptmx
	s.cmd = cmd

	go s.copyOutput(lf)
	_, _ = s.PublishEvent(events.EventKindStatus, map[string]any{"state": "running"})
	return s, nil
}

// newCursorSession starts the Cursor CLI agent in a PTY.
// It uses the official "agent" entrypoint and relies on browser-based login.
// If the agent binary is missing or fails to start, this returns an error so the caller can fall back.
func newCursorSession(ctx context.Context, id, name string, args map[string]interface{}, logDir string, eventsDir string, bufKB int) (*Session, error) {
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
	if err := os.MkdirAll(logDir, 0750); err != nil {
		cancel()
		return nil, err
	}
	logPath := filepath.Join(logDir, id+".log")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		cancel()
		return nil, err
	}
	s.logFile = lf

	workspacePath, _ := args["workspacePath"].(string)
	prompt, _ := args["prompt"].(string)

	// Prefer "cursor agent" (official CLI entrypoint); fall back to bare "agent" if needed.
	cursorBin := "cursor"
	if _, err := exec.LookPath(cursorBin); err != nil {
		cursorBin = "agent"
	}

	cmdArgs := []string{}
	if cursorBin == "cursor" {
		cmdArgs = append(cmdArgs, "agent")
	}
	if prompt != "" {
		cmdArgs = append(cmdArgs, "-p", prompt)
	}

	cmd := exec.CommandContext(ctx, cursorBin, cmdArgs...)
	if workspacePath != "" {
		cmd.Dir = workspacePath
	}
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		lf.Close()
		cancel()
		// Wrap error to signal cursor unavailability; caller will fall back.
		return nil, errors.New("failed to start cursor agent; ensure Cursor IDE is installed and logged in")
	}
	s.ptmx = ptmx
	s.cmd = cmd

	go s.copyOutput(lf)
	_, _ = s.PublishEvent(events.EventKindStatus, map[string]any{"state": "running"})
	return s, nil
}

func (s *Session) copyOutput(logFile *os.File) {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])

			s.mu.Lock()
			s.ring.Write(chunk)
			logFile.Write(chunk)
			for ch := range s.subs {
				select {
				case ch <- chunk:
				default:
				}
			}
			s.mu.Unlock()

			_, _ = s.PublishEvent(events.EventKindAssistant, map[string]any{
				"stream": "stdout",
				"data":   string(chunk),
			})
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("session %s read error: %v", s.ID, err)
			}
			break
		}
	}
}

// Run waits for the process to exit and updates state.
func (s *Session) Run() {
	defer close(s.done)
	err := s.cmd.Wait()

	exitCode := 0
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			if status, ok := exit.Sys().(interface{ ExitStatus() int }); ok {
				exitCode = status.ExitStatus()
			}
		}
	}

	s.mu.Lock()
	s.state = "exited"
	s.exitCode = exitCode
	s.mu.Unlock()

	_, _ = s.PublishEvent(events.EventKindStatus, map[string]any{"state": "exited", "exit_code": exitCode})

	s.mu.Lock()
	if s.ptmx != nil {
		s.ptmx.Close()
	}
	if s.logFile != nil {
		s.logFile.Close()
	}
	s.closed = true
	for ch := range s.subs {
		close(ch)
	}
	for ch := range s.eventSubs {
		close(ch)
	}
	s.subs = nil
	s.eventSubs = nil
	s.mu.Unlock()
	s.cancel()
}

// WriteInput sends input to the PTY.
func (s *Session) WriteInput(data []byte) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed || s.ptmx == nil {
		return io.ErrClosedPipe
	}
	_, err := s.ptmx.Write(data)
	if err == nil {
		_, _ = s.PublishEvent(events.EventKindUser, map[string]any{"data": string(data)})
	}
	return err
}

// Resize sets the PTY window size.
func (s *Session) Resize(cols, rows int) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ptmx == nil {
		return nil
	}
	return pty.Setsize(s.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

// Subscribe returns a channel that receives output chunks. Caller must call Unsubscribe.
func (s *Session) Subscribe() chan []byte {
	ch := make(chan []byte, 64)
	s.mu.Lock()
	if !s.closed {
		s.subs[ch] = struct{}{}
	} else {
		close(ch)
	}
	s.mu.Unlock()
	return ch
}

func (s *Session) SubscribeEvents() chan events.SessionEvent {
	ch := make(chan events.SessionEvent, 256)
	s.mu.Lock()
	if !s.closed {
		s.eventSubs[ch] = struct{}{}
	} else {
		close(ch)
	}
	s.mu.Unlock()
	return ch
}

func (s *Session) UnsubscribeEvents(ch chan events.SessionEvent) {
	s.mu.Lock()
	delete(s.eventSubs, ch)
	s.mu.Unlock()
}

// Unsubscribe removes the channel from subscribers.
func (s *Session) Unsubscribe(ch chan []byte) {
	s.mu.Lock()
	delete(s.subs, ch)
	s.mu.Unlock()
}

// Replay returns the last N bytes from the ring buffer.
func (s *Session) Replay(limit int) []byte {
	return s.ring.Snapshot(limit)
}

func (s *Session) ReplayEventsFromSeq(from uint64) []events.SessionEvent {
	if s.eventsBuf == nil {
		return nil
	}
	return s.eventsBuf.ReplayFromSeq(from)
}

func (s *Session) ReplayEventsLastN(n int) []events.SessionEvent {
	if s.eventsBuf == nil {
		return nil
	}
	return s.eventsBuf.ReplayLastN(n)
}

func (s *Session) LastEventSeq() uint64 {
	if s.eventsBuf == nil {
		return 0
	}
	return s.eventsBuf.LastSeq()
}

// State returns current state and exit code (if exited).
func (s *Session) State() (state string, exitCode int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state, s.exitCode
}

// Terminate kills the session process.
func (s *Session) Terminate() error {
	s.mu.Lock()
	if s.cmd == nil || s.cmd.Process == nil {
		s.mu.Unlock()
		return nil
	}
	proc := s.cmd.Process
	s.mu.Unlock()
	// Kill process group for shell and children
	if proc != nil {
		_ = proc.Kill()
	}
	select {
	case <-s.done:
	case <-time.After(2 * time.Second):
	}
	return nil
}

// Info returns serializable session info.
func (s *Session) Info() map[string]interface{} {
	s.mu.RLock()
	state, code := s.state, s.exitCode
	s.mu.RUnlock()
	return map[string]interface{}{
		"id":        s.ID,
		"name":      s.Name,
		"engine":    s.Engine,
		"state":     state,
		"exit_code": code,
		"last_seq":  s.LastEventSeq(),
		"created":   s.Created.Format(time.RFC3339),
	}
}

// MarshalJSON for Session.Info().
func (s *Session) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Info())
}

func (s *Session) PublishEvent(kind events.EventKind, payload any) (events.SessionEvent, error) {
	if s.eventsBuf == nil {
		return events.SessionEvent{}, nil
	}
	raw, err := events.MarshalPayload(payload)
	if err != nil {
		return events.SessionEvent{}, err
	}
	base := events.SessionEvent{
		SessionID: s.ID,
		Engine:    s.Engine,
		TsMS:      events.NowMS(),
		Kind:      kind,
		Payload:   raw,
	}
	ev := s.eventsBuf.Append(base)
	if s.eventsStore != nil {
		if err := s.eventsStore.Append(s.ID, ev); err != nil {
			log.Printf("events persist failed (session=%s): %v", s.ID, err)
		}
	}

	s.mu.RLock()
	subs := make([]chan events.SessionEvent, 0, len(s.eventSubs))
	for ch := range s.eventSubs {
		subs = append(subs, ch)
	}
	s.mu.RUnlock()
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
	return ev, nil
}
