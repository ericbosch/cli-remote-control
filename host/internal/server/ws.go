package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/ericbosch/cli-remote-control/host/internal/session"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func websocketUpgrader() websocket.Upgrader {
	return upgrader
}

// WS protocol: client sends input, resize, ping; server sends output, replay, status, pong.

type clientMsg struct {
	Type string          `json:"type"`
	Data string          `json:"data"`
	Cols int             `json:"cols"`
	Rows int             `json:"rows"`
	TS   int64           `json:"ts"`
}

type serverMsg struct {
	Type   string `json:"type"`
	Stream string `json:"stream,omitempty"`
	Data   string `json:"data,omitempty"`
	State  string `json:"state,omitempty"`
	Code   int    `json:"code,omitempty"`
	TS     int64  `json:"ts,omitempty"`
}

func runSessionWS(ctx context.Context, conn *websocket.Conn, sess *session.Session) {
	// Send replay first
	replay := sess.Replay(64 * 1024)
	if len(replay) > 0 {
		conn.WriteJSON(serverMsg{Type: "replay", Data: string(replay)})
	}
	conn.WriteJSON(serverMsg{Type: "status", State: "attached"})

	state, code := sess.State()
	if state == "exited" {
		conn.WriteJSON(serverMsg{Type: "status", State: "exited", Code: code})
		conn.Close()
		return
	}

	ch := sess.Subscribe()
	defer sess.Unsubscribe(ch)

	// Ping ticker
	pingTicker := time.NewTicker(30 * time.Second)
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
				sess.WriteInput([]byte(c.Data))
			case "resize":
				sess.Resize(c.Cols, c.Rows)
			case "ping":
				conn.WriteJSON(serverMsg{Type: "pong", TS: c.TS})
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case data, ok := <-ch:
			if !ok {
				state, code := sess.State()
				conn.WriteJSON(serverMsg{Type: "status", State: state, Code: code})
				return
			}
			if err := conn.WriteJSON(serverMsg{Type: "output", Stream: "stdout", Data: string(data)}); err != nil {
				log.Printf("ws write: %v", err)
				return
			}
		case <-pingTicker.C:
			if err := conn.WriteJSON(serverMsg{Type: "pong", TS: time.Now().UnixMilli()}); err != nil {
				return
			}
		}
	}
}
