package httpapi

import "net/http"

// CreateCustomer handles POST /api/customers. Implemented by the public intake
// ticket; this stub answers 501 until then.
func (s *Server) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}
