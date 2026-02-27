package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/ericbosch/cli-remote-control/host/internal/session"
)

// Server is the HTTP and WebSocket server.
type Server struct {
	cfg     Config
	manager *session.Manager
	mux     *http.ServeMux
}

// New creates a new server.
func New(cfg Config) (*Server, error) {
	mgr := session.NewManager(cfg.LogDir, 64)
	mux := http.NewServeMux()
	s := &Server{cfg: cfg, manager: mgr, mux: mux}
	s.routes()
	return s, nil
}

func (s *Server) routes() {
	api := s.authMiddleware(http.HandlerFunc(s.handleAPI))
	s.mux.Handle("/api/", api)
	s.mux.Handle("/ws/", s.authMiddleware(http.HandlerFunc(s.handleWS)))
	if s.cfg.WebDir != "" {
		s.mux.Handle("/", http.FileServer(http.Dir(s.cfg.WebDir)))
	} else {
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("rc-host: use /api/sessions and /ws/sessions/{id}\n"))
		})
	}
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token != "Bearer "+s.cfg.Token && token != s.cfg.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	switch {
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
		Prompt        string                 `json:"prompt"`
		Mode          string                 `json:"mode"`
		Args          map[string]interface{} `json:"args"`
	}
	if err := jsonDecode(r, &body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Engine == "" {
		body.Engine = "cursor"
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
		log.Printf("create session: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
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
