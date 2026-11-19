// Package api is the HTTP layer of the JSON API: routing, request decoding,
// response encoding and error mapping. It holds no business logic; every
// handler delegates to a Service.
//
// Error messages returned to clients are displayed to the user as-is, so they
// are written in French like the rest of the UI.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"bingo/internal/bingo"
)

const maxBodyBytes = 1 << 20

// Service is what the handlers need from the business layer.
// *bingo.Service implements it.
type Service interface {
	List(ctx context.Context) ([]bingo.Bingo, error)
	Get(ctx context.Context, id int64) (*bingo.Bingo, error)
	Create(ctx context.Context, in bingo.Input) (*bingo.Bingo, error)
	Update(ctx context.Context, id int64, in bingo.Input) (*bingo.Bingo, error)
	Delete(ctx context.Context, id int64) error
	SetArchived(ctx context.Context, id int64, archived bool) (*bingo.Bingo, error)
	Duplicate(ctx context.Context, id int64, title string, shuffle bool) (*bingo.Bingo, error)
	Validate(ctx context.Context, bingoID, cellID int64, note string) (*bingo.Bingo, error)
	Unvalidate(ctx context.Context, bingoID, cellID int64) (*bingo.Bingo, error)
	History(ctx context.Context, bingoID int64, limit int) ([]bingo.Event, error)
}

type handler struct{ svc Service }

// New returns the handler for every route under /api/.
func New(svc Service) http.Handler {
	h := &handler{svc: svc}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/bingos", h.listBingos)
	mux.HandleFunc("POST /api/bingos", h.createBingo)
	mux.HandleFunc("GET /api/bingos/{id}", h.getBingo)
	mux.HandleFunc("PUT /api/bingos/{id}", h.updateBingo)
	mux.HandleFunc("DELETE /api/bingos/{id}", h.deleteBingo)
	mux.HandleFunc("POST /api/bingos/{id}/archive", h.setArchived(true))
	mux.HandleFunc("DELETE /api/bingos/{id}/archive", h.setArchived(false))
	mux.HandleFunc("POST /api/bingos/{id}/duplicate", h.duplicateBingo)
	mux.HandleFunc("GET /api/bingos/{id}/history", h.bingoHistory)
	mux.HandleFunc("PUT /api/bingos/{id}/cells/{cellId}/validation", h.validateCell)
	mux.HandleFunc("DELETE /api/bingos/{id}/cells/{cellId}/validation", h.unvalidateCell)
	mux.HandleFunc("GET /api/history", h.history)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "Route inconnue.")
	})
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
	writeJSON(w, status, errorJSON{Error: msg})
}

// fail maps a business error to an HTTP response.
func fail(w http.ResponseWriter, r *http.Request, err error) {
	var verr *bingo.ValidationError
	switch {
	case errors.Is(err, bingo.ErrNotFound):
		writeError(w, http.StatusNotFound, "Bingo ou case introuvable.")
	case errors.Is(err, bingo.ErrArchived):
		writeError(w, http.StatusConflict, "Ce bingo est archivé : désarchivez-le pour le modifier.")
	case errors.As(err, &verr):
		writeError(w, http.StatusUnprocessableEntity, verr.Msg)
	default:
		slog.Error("internal error", "method", r.Method, "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "Erreur interne du serveur.")
	}
}

// pathID reads a positive integer path parameter, replying 404 if it is not one.
func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "Bingo ou case introuvable.")
		return 0, false
	}
	return id, true
}

// decode reads the JSON body into v, replying 400 if it is malformed.
// An empty body is accepted when optional is true.
func decode(w http.ResponseWriter, r *http.Request, v any, optional bool) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	err := json.NewDecoder(r.Body).Decode(v)
	if err == nil || (optional && errors.Is(err, io.EOF)) {
		return true
	}
	writeError(w, http.StatusBadRequest, "Corps de requête JSON invalide.")
	return false
}
