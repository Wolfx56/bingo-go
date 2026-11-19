package api

import (
	"net/http"
	"strconv"

	"bingo/internal/bingo"
)

// replyBingo writes the outcome of a service call that returns one bingo.
func replyBingo(w http.ResponseWriter, r *http.Request, status int, b *bingo.Bingo, err error) {
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, status, toBingoJSON(b, true))
}

func replyEvents(w http.ResponseWriter, r *http.Request, events []bingo.Event, err error) {
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toEventsJSON(events))
}

func (h *handler) listBingos(w http.ResponseWriter, r *http.Request) {
	bingos, err := h.svc.List(r.Context())
	if err != nil {
		fail(w, r, err)
		return
	}
	out := make([]bingoJSON, len(bingos))
	for i := range bingos {
		out[i] = toBingoJSON(&bingos[i], false)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handler) createBingo(w http.ResponseWriter, r *http.Request) {
	var req bingoRequest
	if !decode(w, r, &req, false) {
		return
	}
	b, err := h.svc.Create(r.Context(), req.input())
	replyBingo(w, r, http.StatusCreated, b, err)
}

func (h *handler) getBingo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	b, err := h.svc.Get(r.Context(), id)
	replyBingo(w, r, http.StatusOK, b, err)
}

func (h *handler) updateBingo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req bingoRequest
	if !decode(w, r, &req, false) {
		return
	}
	b, err := h.svc.Update(r.Context(), id, req.input())
	replyBingo(w, r, http.StatusOK, b, err)
}

func (h *handler) deleteBingo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) setArchived(archived bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r, "id")
		if !ok {
			return
		}
		b, err := h.svc.SetArchived(r.Context(), id, archived)
		replyBingo(w, r, http.StatusOK, b, err)
	}
}

func (h *handler) duplicateBingo(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var req duplicateRequest
	if !decode(w, r, &req, true) {
		return
	}
	b, err := h.svc.Duplicate(r.Context(), id, req.Title, req.Shuffle)
	replyBingo(w, r, http.StatusCreated, b, err)
}

func (h *handler) validateCell(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	cellID, ok := pathID(w, r, "cellId")
	if !ok {
		return
	}
	var req validationRequest
	if !decode(w, r, &req, true) {
		return
	}
	b, err := h.svc.Validate(r.Context(), id, cellID, req.Note)
	replyBingo(w, r, http.StatusOK, b, err)
}

func (h *handler) unvalidateCell(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	cellID, ok := pathID(w, r, "cellId")
	if !ok {
		return
	}
	b, err := h.svc.Unvalidate(r.Context(), id, cellID)
	replyBingo(w, r, http.StatusOK, b, err)
}

func (h *handler) bingoHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	events, err := h.svc.History(r.Context(), id, 0)
	replyEvents(w, r, events, err)
}

func (h *handler) history(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := h.svc.History(r.Context(), 0, limit)
	replyEvents(w, r, events, err)
}
