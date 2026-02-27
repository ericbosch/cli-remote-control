package codexrpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/ericbosch/cli-remote-control/host/internal/policy"
)

type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type Message struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	nextID  atomic.Int64
	mu      sync.Mutex
	pending map[int64]chan Message
	onNotif func(method string, params json.RawMessage)
}

func Start(ctx context.Context) (*Client, error) {
	cmd := exec.CommandContext(ctx, "codex", "app-server", "--listen", "stdio://")
	if env, _ := policy.EngineEnv(os.Environ()); len(env) > 0 {
		cmd.Env = env
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	c := &Client{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		pending: make(map[int64]chan Message),
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	go c.readLoop()
	go c.drainStderr()
	return c, nil
}

func (c *Client) SetNotificationHandler(fn func(method string, params json.RawMessage)) {
	c.mu.Lock()
	c.onNotif = fn
	c.mu.Unlock()
}

func (c *Client) Call(ctx context.Context, method string, params any, out any) error {
	id := c.nextID.Add(1)
	ch := make(chan Message, 1)

	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	b, err := json.Marshal(req)
	if err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return err
	}
	b = append(b, '\n')
	if _, err := c.stdin.Write(b); err != nil {
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case msg := <-ch:
		if msg.Error != nil {
			return fmt.Errorf("rpc error %d: %s", msg.Error.Code, msg.Error.Message)
		}
		if out == nil {
			return nil
		}
		return json.Unmarshal(msg.Result, out)
	}
}

func (c *Client) Wait() error {
	return c.cmd.Wait()
}

func (c *Client) Cmd() *exec.Cmd {
	return c.cmd
}

func (c *Client) readLoop() {
	sc := bufio.NewScanner(c.stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if len(msg.ID) > 0 && msg.Method != "" {
			c.replyUnsupportedRequest(msg)
			continue
		}

		if msg.Method != "" && len(msg.ID) == 0 {
			c.mu.Lock()
			fn := c.onNotif
			c.mu.Unlock()
			if fn != nil {
				fn(msg.Method, msg.Params)
			}
			continue
		}

		if len(msg.ID) > 0 {
			id, ok := parseID(msg.ID)
			if !ok {
				continue
			}
			c.mu.Lock()
			ch := c.pending[id]
			delete(c.pending, id)
			c.mu.Unlock()
			if ch != nil {
				ch <- msg
				close(ch)
			}
			continue
		}
	}
}

func parseID(raw json.RawMessage) (int64, bool) {
	var n int64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n, true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false
		}
		return v, true
	}
	return 0, false
}

func (c *Client) replyUnsupportedRequest(msg Message) {
	if len(msg.ID) == 0 {
		return
	}
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      json.RawMessage(msg.ID),
		"error": map[string]any{
			"code":    -32601,
			"message": "method not implemented by client",
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_, _ = c.stdin.Write(append(b, '\n'))
}

func (c *Client) drainStderr() {
	_, _ = io.Copy(io.Discard, c.stderr)
}

var ErrCodexUnavailable = errors.New("codex app-server unavailable")
