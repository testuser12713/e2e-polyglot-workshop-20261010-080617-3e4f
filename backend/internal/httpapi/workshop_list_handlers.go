package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"workshop/internal/domain"
	"workshop/internal/store"
)

// ListWorkshopOrders handles GET /api/workshop/orders. It returns the order
// summaries ordered by creation date descending, optionally filtered by status
// and searched by a case-insensitive partial license plate. The route is
// wrapped in RequireEmployee by the router.
func (s *Server) ListWorkshopOrders(w http.ResponseWriter, r *http.Request) {
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	licensePlate := strings.TrimSpace(r.URL.Query().Get("license_plate"))

	if status != "" && !domain.IsValid(domain.Status(status)) {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "unknown status filter")
		return
	}
	if s.Pool == nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "database unavailable")
		return
	}

	orders, err := store.ListWorkshopOrders(r.Context(), s.Pool, status, licensePlate)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "could not load orders")
		return
	}
	if orders == nil {
		orders = []store.WorkshopOrderSummary{}
	}
	WriteJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

// GetWorkshopOrder handles GET /api/workshop/orders/{order_number}. It returns
// the OrderDetail shape with customer, vehicle, items and status history, or a
// 404 not_found when no order carries that number.
func (s *Server) GetWorkshopOrder(w http.ResponseWriter, r *http.Request) {
	orderNumber := strings.TrimSpace(r.PathValue("order_number"))
	if orderNumber == "" {
		WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		return
	}
	if s.Pool == nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "database unavailable")
		return
	}

	detail, err := store.GetWorkshopOrderDetail(r.Context(), s.Pool, orderNumber)
	if errors.Is(err, store.ErrWorkshopOrderNotFound) {
		WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "could not load order")
		return
	}
	WriteJSON(w, http.StatusOK, detail)
}
