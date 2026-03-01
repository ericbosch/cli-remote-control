package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

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
	if os.Getenv("RC_DEBUG_WS") == "1" {
		u := r.Header.Get("Upgrade")
		c := r.Header.Get("Connection")
		log.Printf("ws/events upgrade attempt: path=%s remote=%s has_upgrade=%t has_connection=%t upgrade=%q connection=%q",
			r.URL.Path, r.RemoteAddr, u != "", c != "", u, c)
	}
	upgrader := websocketUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	runSessionWSEvents(r.Context(), conn, sess, r.RemoteAddr, r.URL.Query().Get("from_seq"), r.URL.Query().Get("last_n"))
}

func runSessionWSEvents(ctx context.Context, conn *websocket.Conn, sess *session.Session, remoteAddr string, fromSeqRaw string, lastNRaw string) {
	debug := os.Getenv("RC_DEBUG_WS") == "1"
	started := time.Now()
	// Keepalive: proxies (including Serve) may drop idle WS connections.
	// Use Ping/Pong to detect half-open connections and keep them warm.
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})

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

	if debug {
		log.Printf("ws/events connected: session=%s remote=%s replay=%d from_seq=%s last_n=%d", sess.ID, remoteAddr, len(replay), fromSeqRaw, lastN)
	}

	pingTicker := time.NewTicker(25 * time.Second)
	defer pingTicker.Stop()

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
				if debug {
					log.Printf("ws/events input: session=%s bytes=%d", sess.ID, len(c.Data))
				}
				_ = sess.WriteInput([]byte(c.Data))
			case "resize":
				if debug {
					log.Printf("ws/events resize: session=%s cols=%d rows=%d", sess.ID, c.Cols, c.Rows)
				}
				_ = sess.Resize(c.Cols, c.Rows)
			case "ping":
				_ = c.TS
			}
		}
	}()

	sent := 0
	for {
		select {
		case <-ctx.Done():
			if debug {
				log.Printf("ws/events disconnected: session=%s remote=%s reason=ctx duration=%s sent=%d", sess.ID, remoteAddr, time.Since(started).Truncate(time.Millisecond), sent)
			}
			return
		case <-done:
			if debug {
				log.Printf("ws/events disconnected: session=%s remote=%s reason=client duration=%s sent=%d", sess.ID, remoteAddr, time.Since(started).Truncate(time.Millisecond), sent)
			}
			return
		case <-pingTicker.C:
			_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(2*time.Second))
		case ev, ok := <-eventsCh:
			if !ok {
				if debug {
					log.Printf("ws/events disconnected: session=%s remote=%s reason=session_closed duration=%s sent=%d", sess.ID, remoteAddr, time.Since(started).Truncate(time.Millisecond), sent)
				}
				return
			}
			sent++
			_ = conn.WriteJSON(ev)
		}
	}
}
