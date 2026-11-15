package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

// TestHTTP checks request errors, methods, status, and JSON output.
func TestHTTP(t *testing.T) {
	root := supervisetest.NewRoot(t)
	files := fstest.MapFS{"guide.md": {Data: []byte("# Café recovery\nRestore the service.")}}
	indexer := startIndexer(t, root, files)
	handler := HTTPHandler(indexer)
	response := readResponse(t, handler, "/search?q=CAF%C3%89")
	if response.Total != 1 || response.Results[0].Title != "Café recovery" {
		t.Fatalf("Unicode query response = %+v", response)
	}
	for _, test := range []struct {
		method string
		target string
		code   int
	}{
		{http.MethodGet, "/search?q=caf%C3%A9", http.StatusOK},
		{http.MethodGet, "/search", http.StatusBadRequest},
		{http.MethodGet, "/search?q=recovery&limit=30", http.StatusBadRequest},
		{http.MethodPost, "/search?q=recovery", http.StatusMethodNotAllowed},
		{http.MethodGet, "/missing", http.StatusNotFound},
	} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(test.method, test.target, nil))
		if recorder.Code != test.code {
			t.Errorf("%s %s: status %d, want %d", test.method, test.target, recorder.Code, test.code)
		}
		if test.code == http.StatusOK || test.code == http.StatusBadRequest {
			if recorder.Header().Get("Content-Type") != "application/json" || !json.Valid(recorder.Body.Bytes()) {
				t.Errorf("%s did not return JSON: %s", test.target, recorder.Body.String())
			}
		}
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))
	var status struct {
		Version   int    `json:"version"`
		Documents int    `json:"documents"`
		Problem   string `json:"problem"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Version != 1 || status.Documents != 1 || status.Problem != "" {
		t.Fatalf("status response = %+v", status)
	}
}
