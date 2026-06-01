package httpapi

import (
	"context"
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
//
//	@Summary		Service health
//	@Description	Liveness probe. The database field is disabled, ok, or unreachable.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	healthResponse
//	@Router			/health [get]
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
