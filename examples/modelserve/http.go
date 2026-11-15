package modelserve

import (
	"encoding/json"
	"net/http"
)

// HTTPHandler exposes predictions, status, and challenger lifetime controls.
func HTTPHandler(router *Router) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /predict", func(w http.ResponseWriter, r *http.Request) {
		var input Features
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Area <= 0 || input.Rooms <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supply positive area_m2 and rooms"})
			return
		}
		name := r.URL.Query().Get("arm")
		if name == "" {
			name = "primary"
		}
		model, err := router.Read(name)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, Predict(model, input))
	})
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, router.Status())
	})
	mux.HandleFunc("PUT /challenger", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Manifest string `json:"manifest"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if err := router.SetChallenger(input.Manifest); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, router.Status())
	})
	mux.HandleFunc("DELETE /challenger", func(w http.ResponseWriter, _ *http.Request) {
		router.RemoveChallenger()
		w.WriteHeader(http.StatusNoContent)
	})
	return mux
}

// writeJSON sends ordinary response values to the JSON encoder.
func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
