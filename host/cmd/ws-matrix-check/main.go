package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type apiErrorEnvelope struct {
	Error *struct {
		Code      string `json:"code"`
		Message   string `json:"message"`
		Hint      string `json:"hint"`
		RequestID string `json:"request_id"`
	} `json:"error"`
}

type sessionInfo struct {
	ID string `json:"id"`
}

type wsTicketResp struct {
	Ticket string `json:"ticket"`
}

type wsClientMsg struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
	TS   int64  `json:"ts,omitempty"`
}

type wsEvent struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

type checkResult struct {
	Name   string
	Status string // PASS|FAIL|SKIP
	Detail string
}

func main() {
	var (
		tokenFile = flag.String("token-file", "", "Path to Bearer token file (required)")
		localBase = flag.String("local-base", "http://127.0.0.1:8787", "Local REST base URL")
		serveBase = flag.String("serve-base", "", "Serve HTTPS base URL (e.g. https://<taildns>:8443). If empty, Serve checks are skipped.")
		tailIP    = flag.String("tail-ip", "", "Optional tailnet IP to probe reachability for :8787 (informational)")
		timeout   = flag.Duration("timeout", 8*time.Second, "Timeout per check")
	)
	flag.Parse()

	if strings.TrimSpace(*tokenFile) == "" {
		fmt.Println("error: --token-file is required")
		os.Exit(2)
	}

	token, err := readToken(*tokenFile)
	if err != nil {
		fmt.Printf("error: read token file: %s\n", err)
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sessID, err := createShellSession(ctx, *localBase, token)
	if err != nil {
		fmt.Printf("local_rest_create_session=FAIL %s\n", scrubErr(err.Error()))
		os.Exit(1)
	}
	fmt.Println("local_rest_create_session=PASS")

	var results []checkResult

	wsLocalBase := strings.Replace(*localBase, "http://", "ws://", 1)
	wsLocalBase = strings.Replace(wsLocalBase, "https://", "wss://", 1)
	results = append(results, wsCheck(ctx, "local_ws_auth_header", wsLocalBase, *localBase, token, sessID, false, *timeout))
	results = append(results, wsCheck(ctx, "local_ws_ticket", wsLocalBase, *localBase, token, sessID, true, *timeout))

	if strings.TrimSpace(*tailIP) != "" {
		results = append(results, probeTailnet8787(*tailIP))
	} else {
		results = append(results, checkResult{Name: "tailnet_ws_direct", Status: "SKIP", Detail: "tail-ip not provided; host binds 127.0.0.1 by default"})
	}

	if strings.TrimSpace(*serveBase) == "" {
		results = append(results, checkResult{Name: "serve_wss_auth_header", Status: "SKIP", Detail: "serve-base not provided"})
		results = append(results, checkResult{Name: "serve_wss_ticket", Status: "SKIP", Detail: "serve-base not provided"})
	} else {
		wssBase := strings.Replace(*serveBase, "https://", "wss://", 1)
		results = append(results, wsCheck(ctx, "serve_wss_auth_header", wssBase, *localBase, token, sessID, false, *timeout))
		results = append(results, wsCheck(ctx, "serve_wss_ticket", wssBase, *localBase, token, sessID, true, *timeout))
	}

	exit := 0
	for _, r := range results {
		if r.Detail != "" {
			fmt.Printf("%s=%s %s\n", r.Name, r.Status, scrubErr(r.Detail))
		} else {
			fmt.Printf("%s=%s\n", r.Name, r.Status)
		}
		if r.Status == "FAIL" {
			exit = 1
		}
	}
	os.Exit(exit)
}

func readToken(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes.TrimSpace(b))), nil
}

func createShellSession(ctx context.Context, base string, token string) (string, error) {
	body := map[string]any{
		"engine": "shell",
		"name":   "ws-matrix-check",
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/sessions", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("http %d: %s", res.StatusCode, summarizeAPIError(raw))
	}
	var info sessionInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return "", err
	}
	if info.ID == "" {
		return "", fmt.Errorf("missing session id")
	}
	return info.ID, nil
}

func issueTicket(ctx context.Context, base string, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/api/ws-ticket", strings.NewReader("{}"))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("http %d: %s", res.StatusCode, summarizeAPIError(raw))
	}
	var out wsTicketResp
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Ticket) == "" {
		return "", fmt.Errorf("missing ticket")
	}
	return strings.TrimSpace(out.Ticket), nil
}

func wsCheck(ctx context.Context, name string, wsBase string, restBase string, token string, sessionID string, useTicket bool, timeout time.Duration) checkResult {
	wsBase = strings.TrimRight(wsBase, "/")
	path := "/ws/events/" + sessionID
	wsURL := wsBase + path

	var header http.Header
	if !useTicket {
		header = make(http.Header)
		header.Set("Authorization", "Bearer "+token)
	} else {
		ticket, err := issueTicket(ctx, restBase, token)
		if err != nil {
			return checkResult{Name: name, Status: "FAIL", Detail: "issue ticket: " + err.Error()}
		}
		wsURL = wsURL + "?ticket=" + ticket
	}

	d := websocket.Dialer{
		HandshakeTimeout: timeout,
		Proxy:            http.ProxyFromEnvironment,
		TLSClientConfig:  &tls.Config{MinVersion: tls.VersionTLS12},
	}
	conn, _, err := d.DialContext(ctx, wsURL, header)
	if err != nil {
		return checkResult{Name: name, Status: "FAIL", Detail: "dial failed: " + err.Error()}
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(timeout))

	marker := "__WS_MATRIX_OK__"
	send := wsClientMsg{Type: "input", Data: "echo " + marker + "\n"}
	if err := conn.WriteJSON(send); err != nil {
		return checkResult{Name: name, Status: "FAIL", Detail: "write input failed: " + err.Error()}
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err) {
				return checkResult{Name: name, Status: "FAIL", Detail: "closed before marker: " + err.Error()}
			}
			return checkResult{Name: name, Status: "FAIL", Detail: "read failed: " + err.Error()}
		}
		var ev wsEvent
		if err := json.Unmarshal(msg, &ev); err != nil {
			continue
		}
		if ev.Kind != "assistant" || len(ev.Payload) == 0 {
			continue
		}
		var p struct {
			Data string `json:"data"`
		}
		if err := json.Unmarshal(ev.Payload, &p); err != nil {
			continue
		}
		if strings.Contains(p.Data, marker) {
			return checkResult{Name: name, Status: "PASS"}
		}
	}
}

func probeTailnet8787(tailIP string) checkResult {
	// Informational only: secure default is bind 127.0.0.1, so this is expected to fail.
	d := net.Dialer{Timeout: 800 * time.Millisecond}
	conn, err := d.Dial("tcp", net.JoinHostPort(strings.TrimSpace(tailIP), "8787"))
	if err != nil {
		return checkResult{Name: "tailnet_ws_direct", Status: "SKIP", Detail: "expected unreachable (bind=127.0.0.1): " + err.Error()}
	}
	_ = conn.Close()
	return checkResult{Name: "tailnet_ws_direct", Status: "FAIL", Detail: "unexpectedly reachable; host may be bound to non-localhost"}
}

func summarizeAPIError(raw []byte) string {
	var env apiErrorEnvelope
	if err := json.Unmarshal(raw, &env); err == nil && env.Error != nil {
		parts := []string{}
		if env.Error.Code != "" {
			parts = append(parts, env.Error.Code)
		}
		if env.Error.Message != "" {
			parts = append(parts, env.Error.Message)
		}
		if env.Error.Hint != "" {
			parts = append(parts, "hint: "+env.Error.Hint)
		}
		if env.Error.RequestID != "" {
			parts = append(parts, "request_id="+env.Error.RequestID)
		}
		if len(parts) > 0 {
			return strings.Join(parts, " · ")
		}
	}
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return "(empty body)"
	}
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

func scrubErr(s string) string {
	// Ensure we never print tickets or bearer strings in errors.
	out := s
	out = strings.ReplaceAll(out, "Authorization: Bearer", "Authorization: Bearer REDACTED")
	out = scrubQuery(out, "ticket")
	out = scrubQuery(out, "token")
	out = scrubQuery(out, "access_token")
	out = scrubQuery(out, "refresh_token")
	return out
}

func scrubQuery(s, key string) string {
	// Replace occurrences like "key=...." with "key=REDACTED".
	needle := key + "="
	for {
		i := strings.Index(s, needle)
		if i < 0 {
			return s
		}
		j := i + len(needle)
		k := j
		for k < len(s) && s[k] != '&' && s[k] != ' ' && s[k] != '"' && s[k] != '\n' && s[k] != '\r' {
			k++
		}
		s = s[:j] + "REDACTED" + s[k:]
	}
}
