package httpapi

import "net/http"

// AddOrderItems handles POST /api/workshop/orders/{order_number}/items.
// Implemented by the items ticket; this stub answers 501 until then.
func (s *Server) AddOrderItems(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}
