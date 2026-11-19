package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSPAHandler(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"favicon.svg":   {Data: []byte("<svg/>")},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
	h := spaHandler(dist)

	cases := []struct {
		name, method, path string
		wantStatus         int
		wantBody           string
		wantCache          string
	}{
		{"root", "GET", "/", http.StatusOK, "<html>app</html>", "no-cache"},
		{"client-side route", "GET", "/bingos/3/edit", http.StatusOK, "<html>app</html>", "no-cache"},
		{"hashed asset", "GET", "/assets/app.js", http.StatusOK, "console.log(1)", "public, max-age=31536000, immutable"},
		{"other file", "GET", "/favicon.svg", http.StatusOK, "<svg/>", ""},
		{"missing asset", "GET", "/assets/gone.js", http.StatusNotFound, "", ""},
		{"wrong method", "POST", "/", http.StatusMethodNotAllowed, "", ""},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code != c.wantStatus {
			t.Errorf("%s: status = %d, want %d", c.name, rec.Code, c.wantStatus)
		}
		if c.wantBody != "" && !strings.Contains(rec.Body.String(), c.wantBody) {
			t.Errorf("%s: body = %q, want it to contain %q", c.name, rec.Body.String(), c.wantBody)
		}
		if got := rec.Header().Get("Cache-Control"); got != c.wantCache {
			t.Errorf("%s: Cache-Control = %q, want %q", c.name, got, c.wantCache)
		}
	}
}
