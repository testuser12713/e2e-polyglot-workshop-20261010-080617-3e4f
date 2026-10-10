package httpapi

import "net/http"

// ConfirmOrder handles POST /api/workshop/orders/{order_number}/confirm.
// Implemented by the status lifecycle ticket; this stub answers 501 until then.
func (s *Server) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}

// SetOrderStatus handles POST /api/workshop/orders/{order_number}/status.
// Implemented by the status lifecycle ticket; this stub answers 501 until then.
func (s *Server) SetOrderStatus(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}
