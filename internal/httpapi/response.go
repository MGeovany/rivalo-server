package httpapi

import (
	"encoding/json"
	"net/http"
)

// maxRequestBytes caps the size of a JSON request body to protect the server
// from oversized payloads.
const maxRequestBytes = 1 << 20 // 1 MiB

// errorResponse is the JSON body returned for error responses.
type errorResponse struct {
	Error string `json:"error"`
}

// decodeJSON reads and decodes a JSON request body into dst, enforcing a size
// limit. It returns an error on malformed or oversized input.
func decodeJSON[T any](w http.ResponseWriter, r *http.Request, dst *T) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	return json.NewDecoder(r.Body).Decode(dst)
}

// writeJSON serializes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error body with the given status code.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
