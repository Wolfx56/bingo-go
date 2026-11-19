package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// Handler serves the embedded front end. Any path that is not a built file
// falls back to index.html so the Vue router can take over.
func Handler() http.Handler {
	return spaHandler(distFS())
}

func spaHandler(dist fs.FS) http.Handler {
	files := http.FileServerFS(dist)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if info, err := fs.Stat(dist, name); name == "" || name == "index.html" || err != nil || info.IsDir() {
			// A missing static asset must not come back as HTML with a 200.
			if strings.HasPrefix(name, "assets/") {
				http.NotFound(w, r)
				return
			}
			serveIndex(w, r, dist)
			return
		}
		if strings.HasPrefix(name, "assets/") {
			// Vite asset file names are content-hashed.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		files.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, dist fs.FS) {
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		http.Error(w, "front end not built: run `make build`", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	if r.Method == http.MethodHead {
		return
	}
	w.Write(index)
}
