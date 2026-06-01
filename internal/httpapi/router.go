// Package httpapi wires HTTP routes to their handlers.
package httpapi

import "net/http"

// NewRouter builds the HTTP handler for the API. The db argument may be nil in
// environments without a configured database; in that case the health endpoint
// reports the database as disabled.
func NewRouter(db Pinger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(db))
	return mux
}
