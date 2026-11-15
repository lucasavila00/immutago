package modelserve

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
)

// TestHTTPWorkflow exercises scoring and dynamic challenger controls.
func TestHTTPWorkflow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		router, root := newTestRouter(t, sampleFiles(), new(eventLog))
		defer func() { <-root.Stop() }()
		handler := HTTPHandler(router)
		response := request(handler, "POST", "/predict", `{"area_m2":90,"rooms":3}`)
		var prediction Prediction
		if err := json.Unmarshal(response.Body.Bytes(), &prediction); err != nil {
			t.Fatal(err)
		}
		if response.Code != 200 || prediction.Estimate != 247000 || prediction.Version != "baseline-v1" {
			t.Fatal(response.Body)
		}
		if response := request(handler, "PUT", "/challenger", `{"manifest":"challenger.json"}`); response.Code != 200 {
			t.Fatal(response.Body)
		}
		response = request(handler, "POST", "/predict?arm=challenger", `{"area_m2":90,"rooms":3}`)
		if err := json.Unmarshal(response.Body.Bytes(), &prediction); err != nil {
			t.Fatal(err)
		}
		if response.Code != 200 || prediction.Estimate != 252000 {
			t.Fatal(response.Body)
		}
		if response := request(handler, "GET", "/status", ""); response.Code != 200 ||
			!strings.Contains(response.Body.String(), "candidate-v1") {
			t.Fatal(response.Body)
		}
		if response := request(handler, "DELETE", "/challenger", ""); response.Code != 204 {
			t.Fatal(response.Code)
		}
		if response := request(handler, "POST", "/predict?arm=challenger", `{"area_m2":90,"rooms":3}`); response.Code != 404 {
			t.Fatal(response.Code)
		}
	})
}

// TestHTTPBadInputs checks bad requests, missing models, and numeric overflow.
func TestHTTPBadInputs(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		router, root := newTestRouter(t, sampleFiles(), new(eventLog))
		defer func() { <-root.Stop() }()
		handler := HTTPHandler(router)
		for _, test := range []struct {
			method, path, body string
			code               int
		}{
			{"POST", "/predict", "{", 400},
			{"POST", "/predict", `{"area_m2":0,"rooms":3}`, 400},
			{"POST", "/predict", `{"area_m2":1e308,"rooms":3}`, 422},
			{"POST", "/predict?arm=missing", `{"area_m2":90,"rooms":3}`, 404},
			{"PUT", "/challenger", "{", 400},
			{"PUT", "/challenger", `{"manifest":"missing.json"}`, 400},
		} {
			response := request(handler, test.method, test.path, test.body)
			if response.Code != test.code || !strings.Contains(response.Body.String(), "error") {
				t.Errorf("%s %s: status=%d body=%s", test.method, test.path, response.Code, response.Body)
			}
		}
		if got := readPrediction(t, router, "primary"); got.Estimate != 247000 {
			t.Fatal(got)
		}
	})
}

// request sends one HTTP request without opening a listener.
func request(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
	return response
}
