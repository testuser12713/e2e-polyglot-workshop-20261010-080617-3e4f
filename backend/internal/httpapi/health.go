package httpapi

import (
	"context"
	"net/http"
	"time"
)

// Health reports whether the API can serve requests. It pings the same
// PostgreSQL pool the real routes use, so a healthy answer means the database
// connection is genuinely usable.
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if s.Pool == nil {
		WriteError(w, http.StatusServiceUnavailable, CodeInternalError, "database unavailable")
		return
	}
	if err := s.Pool.Ping(ctx); err != nil {
		WriteError(w, http.StatusServiceUnavailable, CodeInternalError, "database unavailable")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
