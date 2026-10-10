package httpapi

import "net/http"

// CreateOrder handles POST /api/orders. Implemented by the public intake ticket;
// this stub answers 501 until then.
func (s *Server) CreateOrder(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}
