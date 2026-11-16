package search

import (
	"encoding/json"
	"net/http"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type handler struct{ indexer *Indexer }

// HTTPHandler serves document queries and index status.
func HTTPHandler(indexer supervise.Ref[Indexer]) http.Handler {
	h := handler{indexer: indexer.Borrow()}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search", h.search)
	mux.HandleFunc("GET /status", h.status)
	return mux
}

// search validates the request before it selects one index version.
func (h handler) search(w http.ResponseWriter, r *http.Request) {
	request, err := parseQuery(r.URL.Query().Get("q"), r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	index := h.indexer.Read()
	response := searchIndex(index, request)
	writeJSON(w, http.StatusOK, response)
}

// status gives the JSON encoder temporary read access to the error text.
func (h handler) status(w http.ResponseWriter, _ *http.Request) {
	status := h.indexer.Status()
	response := struct {
		Version   int    `json:"version"`
		Documents int    `json:"documents"`
		Problem   string `json:"problem"`
	}{Version: status.Version, Documents: status.Documents, Problem: status.Problem.Unchecked()}
	writeJSON(w, http.StatusOK, response)
}

// writeJSON writes an ordinary response value.
func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
