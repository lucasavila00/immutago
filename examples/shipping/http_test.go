package shipping

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lucasavila00/immutago/pkg/supervise"
	"github.com/lucasavila00/immutago/pkg/supervise/supervisetest"
)

// TestHTTPQuote checks the JSON contract through the running catalog service.
func TestHTTPQuote(t *testing.T) {
	loader := startLoader(t, "catalog.json", time.Hour)
	handler := NewHandler(loader)
	request := httptest.NewRequest(http.MethodPost, "/quotes", strings.NewReader(
		`{"destination":"US","parcels":[{"weight_grams":1000},{"weight_grams":1001}]}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("response = %d, %v, %s", recorder.Code, recorder.Header(), recorder.Body)
	}
	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.CatalogVersion != "2026-11" || len(response.Quotes) != 2 || response.Quotes[0].TotalCents != 1650 {
		t.Fatalf("response = %+v", response)
	}
}

// TestHTTPNoMatch returns a successful empty quote result for an unserved shipment.
func TestHTTPNoMatch(t *testing.T) {
	loader := startLoader(t, "catalog.json", time.Hour)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/quotes", strings.NewReader(
		`{"destination":"GB","parcels":[{"weight_grams":500}]}`))
	NewHandler(loader).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"quotes":[]`) ||
		!strings.Contains(recorder.Body.String(), `"no_match":"destination_not_served"`) {
		t.Fatalf("response = %d, %s", recorder.Code, recorder.Body)
	}
}

// TestHTTPRejectsInvalidRequests checks parsing, domain validation, routing, and methods.
func TestHTTPRejectsInvalidRequests(t *testing.T) {
	loader := startLoader(t, "catalog.json", time.Hour)
	handler := NewHandler(loader)
	cases := []struct {
		name, method, path, body string
		status                   int
	}{
		{"malformed", "POST", "/quotes", `{`, 400},
		{"empty", "POST", "/quotes", ``, 400},
		{"null", "POST", "/quotes", `null`, 400},
		{"unknown field", "POST", "/quotes", `{"country":"US"}`, 400},
		{"extra document", "POST", "/quotes", `{} {}`, 400},
		{"destination", "POST", "/quotes", `{"destination":"us","parcels":[{"weight_grams":1}]}`, 400},
		{"no parcels", "POST", "/quotes", `{"destination":"US","parcels":[]}`, 400},
		{"zero weight", "POST", "/quotes", `{"destination":"US","parcels":[{"weight_grams":0}]}`, 400},
		{"negative weight", "POST", "/quotes", `{"destination":"US","parcels":[{"weight_grams":-1}]}`, 400},
		{"fractional weight", "POST", "/quotes", `{"destination":"US","parcels":[{"weight_grams":1.5}]}`, 400},
		{"method", "GET", "/quotes", ``, 405},
		{"path", "POST", "/unknown", `{}`, 404},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(item.method, item.path, strings.NewReader(item.body)))
			if recorder.Code != item.status {
				t.Fatalf("status = %d; want %d: %s", recorder.Code, item.status, recorder.Body)
			}
			if item.status == 405 && recorder.Header().Get("Allow") != http.MethodPost {
				t.Fatal("method error did not name the allowed method")
			}
		})
	}
}

// startLoader starts the real service with cleanup that waits for its Run method.
func startLoader(t *testing.T, path string, every time.Duration) supervise.Ref[Loader] {
	t.Helper()
	root := supervisetest.NewRoot(t)
	loader, err := supervise.Start[Loader](root, Args{Path: path, Every: every})
	if err != nil {
		t.Fatal(err)
	}
	return loader
}
