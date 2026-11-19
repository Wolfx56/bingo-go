package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"bingo/internal/api"
	"bingo/internal/bingo"
	"bingo/internal/store"
)

func newServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(api.New(bingo.NewService(st)))
	t.Cleanup(func() {
		srv.Close()
		st.Close()
	})
	return srv
}

// call sends a request and decodes the JSON response into out (if not nil).
func call(t *testing.T, srv *httptest.Server, method, path, body string, out any) int {
	t.Helper()
	req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			t.Fatalf("%s %s: decoding response: %v", method, path, err)
		}
	}
	return res.StatusCode
}

const newBingo = `{"title":"Test","size":3,"cells":["a","b","c","d","e","f","g","h","i"]}`

func TestBingoLifecycle(t *testing.T) {
	srv := newServer(t)

	var created map[string]any
	if status := call(t, srv, "POST", "/api/bingos", newBingo, &created); status != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201", status)
	}
	// The JSON contract the front end relies on.
	for _, key := range []string{"id", "title", "description", "size", "createdAt", "updatedAt", "archivedAt", "checked", "total", "pattern", "lines", "cells"} {
		if _, ok := created[key]; !ok {
			t.Errorf("create: response has no %q field", key)
		}
	}
	cells := created["cells"].([]any)
	cellID := cells[0].(map[string]any)["id"]

	var validated struct {
		Checked int
		Cells   []struct {
			CheckedAt *string
			Note      string
		}
	}
	path := fmt.Sprintf("/api/bingos/1/cells/%v/validation", cellID)
	if status := call(t, srv, "PUT", path, `{"note":"done"}`, &validated); status != http.StatusOK {
		t.Fatalf("validate: status = %d, want 200", status)
	}
	if validated.Checked != 1 || validated.Cells[0].CheckedAt == nil || validated.Cells[0].Note != "done" {
		t.Errorf("validate: unexpected response %+v", validated)
	}

	// The list carries the pattern but not the cells.
	var list []map[string]any
	if status := call(t, srv, "GET", "/api/bingos", "", &list); status != http.StatusOK || len(list) != 1 {
		t.Fatalf("list: status = %d, %d bingos", status, len(list))
	}
	if _, ok := list[0]["cells"]; ok {
		t.Error("list: response should not include cells")
	}
	if pattern := list[0]["pattern"].([]any); len(pattern) != 9 || pattern[0] != true {
		t.Errorf("list: unexpected pattern %v", pattern)
	}

	var events []map[string]any
	if status := call(t, srv, "GET", "/api/history", "", &events); status != http.StatusOK || len(events) != 2 {
		t.Fatalf("history: status = %d, %d events, want 200 and 2", status, len(events))
	}
	if events[0]["kind"] != "validated" || events[0]["bingoTitle"] != "Test" {
		t.Errorf("history: unexpected latest event %v", events[0])
	}

	if status := call(t, srv, "POST", "/api/bingos/1/archive", "", nil); status != http.StatusOK {
		t.Fatalf("archive: status = %d, want 200", status)
	}
	if status := call(t, srv, "DELETE", path, "", nil); status != http.StatusConflict {
		t.Errorf("unvalidate on archived bingo: status = %d, want 409", status)
	}

	if status := call(t, srv, "DELETE", "/api/bingos/1", "", nil); status != http.StatusNoContent {
		t.Errorf("delete: status = %d, want 204", status)
	}
}

func TestErrors(t *testing.T) {
	srv := newServer(t)
	cases := []struct {
		name, method, path, body string
		want                     int
	}{
		{"invalid input", "POST", "/api/bingos", `{"title":"","size":3,"cells":[]}`, http.StatusUnprocessableEntity},
		{"malformed JSON", "POST", "/api/bingos", `not json`, http.StatusBadRequest},
		{"unknown bingo", "GET", "/api/bingos/999", "", http.StatusNotFound},
		{"non-numeric id", "GET", "/api/bingos/abc", "", http.StatusNotFound},
		{"unknown cell", "PUT", "/api/bingos/999/cells/1/validation", "", http.StatusNotFound},
		{"unknown route", "GET", "/api/nope", "", http.StatusNotFound},
	}
	for _, c := range cases {
		var body struct{ Error string }
		if status := call(t, srv, c.method, c.path, c.body, &body); status != c.want {
			t.Errorf("%s: status = %d, want %d", c.name, status, c.want)
		}
		if body.Error == "" {
			t.Errorf("%s: response has no error message", c.name)
		}
	}
}
