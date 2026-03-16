package webapp

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// reservedPaths are paths that must not be treated as short codes.
var reservedPaths = map[string]bool{
	"app":     true,
	"api":     true,
	"healthz": true,
	"static":  true,
	"assets":  true,
	"favicon": true,
	"preview": true,
}

var codePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,100}$`)

// Routes sets up the HTTP routes.
// Priority: API proxy → short-code redirect → SPA (catch-all)
func (app *App) Routes() http.Handler {
	mux := http.NewServeMux()

	// 1. API reverse proxy → backend REST gateway
	if app.BackendURL != "" {
		mux.Handle("/api/", APIProxy(app.BackendURL))
	}

	// 2. Health check (frontend)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// 3. Everything else: short-code redirect or SPA
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		// Root → serve SPA index
		if path == "" {
			app.serveSPAIndex(w, r)
			return
		}

		// Strip trailing + for public preview (like bit.ly's code+ feature)
		if strings.HasSuffix(path, "+") {
			code := strings.TrimSuffix(path, "+")
			if codePattern.MatchString(code) && !reservedPaths[code] {
				http.Redirect(w, r, "/preview/"+code, http.StatusTemporaryRedirect)
				return
			}
		}

		// If it looks like a short code and is NOT a reserved path, try redirect
		if codePattern.MatchString(path) && !reservedPaths[path] {
			app.GetURL(w, r)
			return
		}

		// Everything else → serve SPA (handles client-side routing)
		app.serveSPA(w, r)
	})

	return mux
}

// serveSPA serves the React SPA. If the file exists on disk, serve it.
// Otherwise serve index.html for client-side routing.
func (app *App) serveSPA(w http.ResponseWriter, r *http.Request) {
	// Try to serve the file directly (JS, CSS, images, etc.)
	path := r.URL.Path
	if path != "/" {
		fullPath := filepath.Join(app.SPADir, filepath.Clean(path))
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, fullPath)
			return
		}
	}

	// Fallback to index.html for SPA client-side routing
	app.serveSPAIndex(w, r)
}

// serveSPAIndex serves the SPA index.html.
func (app *App) serveSPAIndex(w http.ResponseWriter, r *http.Request) {
	indexPath := filepath.Join(app.SPADir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		http.Error(w, "SPA not built. Run 'npm run build' in ui-src/", http.StatusServiceUnavailable)
		return
	}
	http.ServeFile(w, r, indexPath)
}
