// Package api exposes the JSON API and serves the embedded front end.
//
// Error messages returned to clients are displayed to the user as-is, so they
// are written in French like the rest of the UI.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"

	"bingo/internal/store"
)

const maxBodyBytes = 1 << 20

type server struct{ store *store.Store }

// New returns the full HTTP handler: the API under /api/, the SPA for everything else.
func New(s *store.Store, dist fs.FS) http.Handler {
	srv := &server{store: s}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/bingos", srv.listBingos)
	mux.HandleFunc("POST /api/bingos", srv.createBingo)
	mux.HandleFunc("GET /api/bingos/{id}", srv.getBingo)
	mux.HandleFunc("PUT /api/bingos/{id}", srv.updateBingo)
	mux.HandleFunc("DELETE /api/bingos/{id}", srv.deleteBingo)
	mux.HandleFunc("POST /api/bingos/{id}/archive", srv.setArchived(true))
	mux.HandleFunc("DELETE /api/bingos/{id}/archive", srv.setArchived(false))
	mux.HandleFunc("POST /api/bingos/{id}/duplicate", srv.duplicateBingo)
	mux.HandleFunc("GET /api/bingos/{id}/history", srv.bingoHistory)
	mux.HandleFunc("PUT /api/bingos/{id}/cells/{cellId}/validation", srv.validateCell)
	mux.HandleFunc("DELETE /api/bingos/{id}/cells/{cellId}/validation", srv.unvalidateCell)
	mux.HandleFunc("GET /api/history", srv.history)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "Route inconnue.")
	})

	mux.Handle("/", spaHandler(dist))
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writing response", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// fail maps a store error to an HTTP response.
func fail(w http.ResponseWriter, r *http.Request, err error) {
	var verr *store.ValidationError
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "Bingo ou case introuvable.")
	case errors.Is(err, store.ErrArchived):
		writeError(w, http.StatusConflict, "Ce bingo est archivé : désarchivez-le pour le modifier.")
	case errors.As(err, &verr):
		writeError(w, http.StatusUnprocessableEntity, verr.Msg)
	default:
		slog.Error("internal error", "method", r.Method, "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "Erreur interne du serveur.")
	}
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "Bingo ou case introuvable.")
		return 0, false
	}
	return id, true
}

// decode reads the JSON body into v. An empty body is accepted when optional is true.
func decode(w http.ResponseWriter, r *http.Request, v any, optional bool) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	err := json.NewDecoder(r.Body).Decode(v)
	if err == nil || (optional && errors.Is(err, io.EOF)) {
		return true
	}
	writeError(w, http.StatusBadRequest, "Corps de requête JSON invalide.")
	return false
}

func (s *server) listBingos(w http.ResponseWriter, r *http.Request) {
	bingos, err := s.store.List(r.Context())
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, bingos)
}

func (s *server) createBingo(w http.ResponseWriter, r *http.Request) {
	var in store.BingoInput
	if !decode(w, r, &in, false) {
		return
	}
	b, err := s.store.Create(r.Context(), in)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *server) getBingo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	b, err := s.store.Get(r.Context(), id)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *server) updateBingo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var in store.BingoInput
	if !decode(w, r, &in, false) {
		return
	}
	b, err := s.store.Update(r.Context(), id, in)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *server) deleteBingo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := s.store.Delete(r.Context(), id); err != nil {
		fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) setArchived(archived bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, "id")
		if !ok {
			return
		}
		b, err := s.store.SetArchived(r.Context(), id, archived)
		if err != nil {
			fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, b)
	}
}

func (s *server) duplicateBingo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var in struct {
		Title   string `json:"title"`
		Shuffle bool   `json:"shuffle"`
	}
	if !decode(w, r, &in, true) {
		return
	}
	b, err := s.store.Duplicate(r.Context(), id, in.Title, in.Shuffle)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *server) validateCell(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	cellID, ok := pathID(w, r, "cellId")
	if !ok {
		return
	}
	var in struct {
		Note string `json:"note"`
	}
	if !decode(w, r, &in, true) {
		return
	}
	b, err := s.store.Validate(r.Context(), id, cellID, in.Note)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *server) unvalidateCell(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	cellID, ok := pathID(w, r, "cellId")
	if !ok {
		return
	}
	b, err := s.store.Unvalidate(r.Context(), id, cellID)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *server) bingoHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	events, err := s.store.History(r.Context(), id, 0)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (s *server) history(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := s.store.History(r.Context(), 0, limit)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// spaHandler serves the front-end files; any unknown path falls back to index.html
// so the Vue router can take over.
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
