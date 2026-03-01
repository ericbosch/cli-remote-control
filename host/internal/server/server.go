package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ericbosch/cli-remote-control/host/internal/codexrpc"
	"github.com/ericbosch/cli-remote-control/host/internal/session"
)

// Server is the HTTP and WebSocket server.
type Server struct {
	cfg     Config
	manager *session.Manager
	tickets *wsTicketManager
	mux     *http.ServeMux
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	mgr := session.NewManager(cfg.LogDir, 64, filepath.Join(".run", "sessions"))
	mux := http.NewServeMux()
	s := &Server{cfg: cfg, manager: mgr, tickets: newWSTicketManager(), mux: mux}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealthz)

	api := s.authMiddleware(false, http.HandlerFunc(s.handleAPI))
	s.mux.Handle("/api/", api)
	s.mux.Handle("/ws/events/", s.wsAuthMiddleware(http.HandlerFunc(s.handleWSEvents)))
	s.mux.Handle("/ws/", s.wsAuthMiddleware(http.HandlerFunc(s.handleWS)))
	if s.cfg.WebDir != "" {
		s.mux.Handle("/", spaFileServer(s.cfg.WebDir))
	} else {
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("rc-host: use /api/sessions and /ws/events/{id} (legacy: /ws/sessions/{id})\n"))
		})
	}
}

func (s *Server) authMiddleware(allowQueryToken bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(s.cfg.Token) == "" {
			http.Error(w, "server misconfigured", http.StatusInternalServerError)
			return
		}

		token := strings.TrimSpace(r.Header.Get("Authorization"))
		if token == "" && allowQueryToken {
			token = strings.TrimSpace(r.URL.Query().Get("token"))
		}
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = strings.TrimSpace(token[len("bearer "):])
		}
		if token != s.cfg.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) wsAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prefer Authorization header (non-browser clients). Browsers should use a short-lived ticket.
		token := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = strings.TrimSpace(token[len("bearer "):])
		}
		if token != "" {
			if token != s.cfg.Token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		ticket := strings.TrimSpace(r.URL.Query().Get("ticket"))
		if ticket == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !s.tickets.Consume(ticket, time.Now()) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	jsonEncoder(w).Encode(map[string]interface{}{
		"ok": true,
	})
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
	case path == "/api/engines" && r.Method == http.MethodGet:
		s.listEngines(w, r)
	case path == "/api/ws-ticket" && r.Method == http.MethodPost:
		s.issueWSTicket(w, r)
	case path == "/api/sessions" && r.Method == http.MethodGet:
		s.listSessions(w, r)
	case path == "/api/sessions" && r.Method == http.MethodPost:
		s.createSession(w, r)
	default:
		// /api/sessions/{id}/terminate
		if len(path) > len("/api/sessions/") && r.Method == http.MethodPost {
			rest := path[len("/api/sessions/"):]
			if strings.HasSuffix(rest, "/terminate") {
				id := strings.TrimSuffix(rest, "/terminate")
				if id != "" {
					s.terminateSession(w, r, id)
					return
				}
			}
		}
		http.NotFound(w, r)
	}
}

func (s *Server) listEngines(w http.ResponseWriter, _ *http.Request) {
	// Shell is always available.
	engines := []string{"shell"}

	if _, err := exec.LookPath("codex"); err == nil {
		engines = append(engines, "codex")
	}
	// Cursor support is best-effort; it may rely on cursor-agent/agent/cursor.
	if _, err := exec.LookPath("cursor-agent"); err == nil {
		engines = append(engines, "cursor")
	} else if _, err := exec.LookPath("agent"); err == nil {
		engines = append(engines, "cursor")
	} else if _, err := exec.LookPath("cursor"); err == nil {
		engines = append(engines, "cursor")
	}

	w.Header().Set("Content-Type", "application/json")
	jsonEncoder(w).Encode(engines)
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	list := s.manager.List()
	out := make([]map[string]interface{}, len(list))
	for i, sess := range list {
		out[i] = sess.Info()
	}
	w.Header().Set("Content-Type", "application/json")
	enc := jsonEncoder(w)
	enc.Encode(out)
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Engine        string                 `json:"engine"`
		Name          string                 `json:"name"`
		WorkspacePath string                 `json:"workspacePath"`
		Workspace     string                 `json:"workspace"` // backward-compatible alias
		Prompt        string                 `json:"prompt"`
		Mode          string                 `json:"mode"`
		Args          map[string]interface{} `json:"args"`
	}
	if err := jsonDecode(r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body", "")
		return
	}
	if body.Engine == "" {
		// Secure, reliable default: shell-first.
		body.Engine = "shell"
	}
	if body.WorkspacePath == "" && body.Workspace != "" {
		body.WorkspacePath = body.Workspace
	}
	switch body.Engine {
	case "shell", "codex", "cursor":
		// ok
	default:
		writeAPIError(w, http.StatusBadRequest, "invalid_engine", "Unknown engine", "Choose an engine from GET /api/engines (at minimum: shell).")
		return
	}

	if body.WorkspacePath != "" {
		if st, err := os.Stat(body.WorkspacePath); err != nil || !st.IsDir() {
			writeAPIError(w, http.StatusBadRequest, "invalid_workspace", "Workspace path does not exist or is not a directory", "Set Workspace to an existing directory, or leave it blank to use a safe default.")
			return
		}
	}

	// For engines that need a cwd, prefer a safe default when empty.
	if body.Engine == "codex" && body.WorkspacePath == "" {
		if cwd, err := os.Getwd(); err == nil && cwd != "" {
			body.WorkspacePath = cwd
		}
	}

	args := map[string]interface{}{}
	for k, v := range body.Args {
		args[k] = v
	}
	if body.WorkspacePath != "" {
		args["workspacePath"] = body.WorkspacePath
	}
	if body.Prompt != "" {
		args["prompt"] = body.Prompt
	}
	if body.Mode != "" {
		args["mode"] = body.Mode
	}

	sess, err := s.manager.Create(r.Context(), body.Engine, body.Name, args)
	if err != nil {
		// Codex errors should be actionable and never opaque 500s.
		if body.Engine == "codex" {
			code := "codex_failed"
			hint := "Ensure the 'codex' CLI is installed and authenticated on the host (e.g. run `codex` locally to complete login). This service does not use OPENAI_API_KEY / PAYG keys."
			if errors.Is(err, codexrpc.ErrCodexUnavailable) {
				code = "codex_unavailable"
				hint = "Install the 'codex' CLI on the host and ensure it is on PATH. This service does not use OPENAI_API_KEY / PAYG keys."
			}
			writeAPIError(w, http.StatusFailedDependency, code, "Codex session creation failed", err.Error()+"\n"+hint)
			return
		}
		log.Printf("create session: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "Internal error", "")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	jsonEncoder(w).Encode(sess.Info())
}

func (s *Server) terminateSession(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.manager.Terminate(id); err != nil {
		if err == session.ErrNotFound {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// /ws/sessions/{id}
	if len(path) < len("/ws/sessions/") || path[:len("/ws/sessions/")] != "/ws/sessions/" {
		http.NotFound(w, r)
		return
	}
	id := path[len("/ws/sessions/"):]
	if id == "" {
		http.NotFound(w, r)
		return
	}
	sess := s.manager.Get(id)
	if sess == nil {
		http.NotFound(w, r)
		return
	}
	// Upgrade and run WS handler
	upgrader := websocketUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	runSessionWS(r.Context(), conn, sess)
}

// Run starts the HTTP server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	addr := s.cfg.Bind + ":" + s.cfg.Port
	srv := &http.Server{Addr: addr, Handler: corsMiddleware(s.mux)}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	log.Printf("Listening on http://%s", addr)
	return srv.ListenAndServe()
}

// helpers for JSON
func jsonEncoder(w http.ResponseWriter) *json.Encoder {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc
}
