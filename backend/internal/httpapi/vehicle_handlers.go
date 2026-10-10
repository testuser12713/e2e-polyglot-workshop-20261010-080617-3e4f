package httpapi

import "net/http"

// CreateVehicle handles POST /api/vehicles. Implemented by the public intake
// ticket; this stub answers 501 until then.
func (s *Server) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}
