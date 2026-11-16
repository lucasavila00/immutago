package shipping

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/lucasavila00/immutago/pkg/supervise"
)

// NewHandler serves POST /quotes with ordinary JSON request and response data.
func NewHandler(loader supervise.Ref[Loader]) http.Handler {
	worker := loader.Borrow()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/quotes" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST /quotes"})
			return
		}
		var request Request
		if err := decodeJSON(r.Body, &request); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		catalog := worker.Catalog()
		response := Quote(catalog, request)
		if response.Error != "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": response.Error})
			return
		}
		writeJSON(w, http.StatusOK, response)
	})
}

// decodeJSON reads one complete document and reports unknown fields.
func decodeJSON(input io.Reader, destination any) error {
	decoder := json.NewDecoder(input)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected one JSON document")
	}
	return nil
}

// writeJSON sets the response type and reports a failed network write.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("shipping response: %v", err)
	}
}
