package httpapi

import "net/http"

// GetOrderStatus handles GET /api/orders/status. Implemented by the customer
// lookup ticket; this stub answers 501 until then.
func (s *Server) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}

// GetOrderInvoice handles GET /api/orders/{order_number}/invoice. Implemented by
// the customer lookup ticket; this stub answers 501 until then.
func (s *Server) GetOrderInvoice(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}
