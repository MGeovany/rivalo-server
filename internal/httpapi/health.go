package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger reports whether a backing dependency (e.g. the database) is reachable.
// It is an interface so handlers can be tested without a real database.
type Pinger interface {
	Ping(ctx context.Context) error
}

// healthResponse is the JSON body returned by the health endpoint.
type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

// healthHandler reports the liveness of the service and, when a database is
// configured, its reachability. The endpoint always returns 200 as long as the
// process is serving requests; the database field communicates dependency state
// without failing the liveness probe.
func healthHandler(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "disabled"
		if db != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := db.Ping(ctx); err != nil {
				dbStatus = "unreachable"
			} else {
				dbStatus = "ok"
			}
		}

		writeJSON(w, http.StatusOK, healthResponse{
			Status:   "ok",
			Database: dbStatus,
		})
	}
}

// writeJSON serializes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
