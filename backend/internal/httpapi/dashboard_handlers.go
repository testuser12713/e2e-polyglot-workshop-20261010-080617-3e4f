package httpapi

import "net/http"

// GetDashboard handles GET /api/workshop/dashboard. Implemented by the dashboard
// ticket; this stub answers 501 until then.
func (s *Server) GetDashboard(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}
