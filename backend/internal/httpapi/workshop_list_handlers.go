package httpapi

import "net/http"

// ListWorkshopOrders handles GET /api/workshop/orders. Implemented by the
// workshop list ticket; this stub answers 501 until then.
func (s *Server) ListWorkshopOrders(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}

// GetWorkshopOrder handles GET /api/workshop/orders/{order_number}. Implemented
// by the workshop list ticket; this stub answers 501 until then.
func (s *Server) GetWorkshopOrder(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}
