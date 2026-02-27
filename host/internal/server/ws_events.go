package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ericbosch/cli-remote-control/host/internal/events"
	"github.com/ericbosch/cli-remote-control/host/internal/session"
	"github.com/gorilla/websocket"
)

func (s *Server) handleWSEvents(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// /ws/events/{id}
	if len(path) < len("/ws/events/") || path[:len("/ws/events/")] != "/ws/events/" {
		http.NotFound(w, r)
		return
	}
	id := path[len("/ws/events/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	sess := s.manager.Get(id)
	if sess == nil {
		http.NotFound(w, r)
		return
	}
	upgrader := websocketUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	runSessionWSEvents(r.Context(), conn, sess, r.URL.Query().Get("from_seq"), r.URL.Query().Get("last_n"))
}

func runSessionWSEvents(ctx context.Context, conn *websocket.Conn, sess *session.Session, fromSeqRaw string, lastNRaw string) {
	fromSeq := uint64(0)
	if fromSeqRaw != "" {
		if v, err := strconv.ParseUint(fromSeqRaw, 10, 64); err == nil {
			fromSeq = v
		}
	}
	lastN := 0
	if lastNRaw != "" {
		if v, err := strconv.Atoi(lastNRaw); err == nil {
			lastN = v
		}
	}

	var replay []events.SessionEvent
	if fromSeqRaw != "" {
		replay = sess.ReplayEventsFromSeq(fromSeq)
	} else if lastN > 0 {
		replay = sess.ReplayEventsLastN(lastN)
	} else {
		replay = sess.ReplayEventsLastN(256)
	}
	for _, ev := range replay {
		_ = conn.WriteJSON(ev)
	}

	state, code := sess.State()
	if state == "exited" {
		exited, _ := sess.PublishEvent(events.EventKindStatus, map[string]any{"state": "exited", "exit_code": code})
		if exited.Seq != 0 {
			_ = conn.WriteJSON(exited)
		}
		return
	}

	eventsCh := sess.SubscribeEvents()
	defer sess.UnsubscribeEvents(eventsCh)

	_, _ = sess.PublishEvent(events.EventKindStatus, map[string]any{"state": "attached"})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var c clientMsg
			if err := json.Unmarshal(msg, &c); err != nil {
				continue
			}
			switch c.Type {
			case "input":
				_ = sess.WriteInput([]byte(c.Data))
			case "resize":
				_ = sess.Resize(c.Cols, c.Rows)
			case "ping":
				_ = c.TS
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case ev, ok := <-eventsCh:
			if !ok {
				return
			}
			_ = conn.WriteJSON(ev)
		}
	}
}
