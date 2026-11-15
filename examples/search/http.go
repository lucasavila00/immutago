package search

import (
	"encoding/json"
	"net/http"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

type handler struct{ indexer supervise.Ref[Indexer] }

// HTTPHandler serves document queries and index status.
func HTTPHandler(indexer supervise.Ref[Indexer]) http.Handler {
	h := handler{indexer: indexer}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search", h.search)
	mux.HandleFunc("GET /status", h.status)
	return mux
}

func (h handler) search(w http.ResponseWriter, r *http.Request) {
	request, err := parseQuery(r.URL.Query().Get("q"), r.URL.Query().Get("limit"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	index := h.indexer.Current().Read()
	response := searchIndex(index, request)
	writeJSON(w, http.StatusOK, response)
}

func (h handler) status(w http.ResponseWriter, _ *http.Request) {
	status := h.indexer.Current().Status()
	response := struct {
		Version   int    `json:"version"`
		Documents int    `json:"documents"`
		Problem   string `json:"problem"`
	}{Version: status.Version, Documents: status.Documents, Problem: status.Problem.Clone()}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
