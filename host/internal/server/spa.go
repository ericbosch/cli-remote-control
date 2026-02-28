package server

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// spaFileServer serves files from webDir and falls back to index.html for
// history-based SPA routes.
//
// Security notes:
// - Uses http.Dir + http.FileServer which rejects path traversal.
// - Only falls back for GET/HEAD, for paths without an extension, and when the
//   request looks like an HTML navigation.
func spaFileServer(webDir string) http.Handler {
	fs := http.Dir(webDir)
	fileServer := http.FileServer(fs)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "" {
			p = "/"
		}
		p = path.Clean(p)
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}

		// If the path exists (file or directory), defer to the normal file server.
		if f, err := fs.Open(p); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Do not fall back for likely asset requests.
		if ext := filepath.Ext(p); ext != "" {
			http.NotFound(w, r)
			return
		}

		// Only fall back for typical navigations.
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if !acceptsHTML(r.Header.Get("Accept")) {
			http.NotFound(w, r)
			return
		}

		// Serve index.html for SPA routes.
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}

func acceptsHTML(accept string) bool {
	a := strings.ToLower(accept)
	// Default browser navigations include text/html. Some clients omit Accept;
	// treat that as HTML-friendly for robustness.
	if strings.TrimSpace(a) == "" {
		return true
	}
	return strings.Contains(a, "text/html")
}

