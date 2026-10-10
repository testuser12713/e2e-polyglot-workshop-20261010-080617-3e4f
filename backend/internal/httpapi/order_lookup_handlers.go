package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"workshop/internal/store"
)

// GetOrderStatus handles GET /api/orders/status?order_number=&license_plate=.
//
// It answers 200 with the current status and the status history only when both
// the order number and the license plate match the same order. A missing
// parameter is a validation error; any non-match answers the same 404 for a
// wrong plate, an unknown order number and a mismatched pair alike, so the
// response never reveals whether an order exists (AC-31).
func (s *Server) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderNumber := strings.TrimSpace(r.URL.Query().Get("order_number"))
	licensePlate := strings.TrimSpace(r.URL.Query().Get("license_plate"))
	if orderNumber == "" || licensePlate == "" {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "order_number and license_plate are required")
		return
	}
	if s.Pool == nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}

	result, err := store.LookupOrderStatus(r.Context(), s.Pool, orderNumber, licensePlate)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

// GetOrderInvoice handles GET /api/orders/{order_number}/invoice?license_plate=.
//
// It answers 200 with the invoice, its items and the net, VAT and gross totals
// only when the path order number and the license plate query both match the
// same order. Every non-match answers the same 404 (AC-31).
func (s *Server) GetOrderInvoice(w http.ResponseWriter, r *http.Request) {
	orderNumber := strings.TrimSpace(r.PathValue("order_number"))
	licensePlate := strings.TrimSpace(r.URL.Query().Get("license_plate"))
	if orderNumber == "" || licensePlate == "" {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "order_number and license_plate are required")
		return
	}
	if s.Pool == nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}

	result, err := store.LookupInvoice(r.Context(), s.Pool, orderNumber, licensePlate)
	if errors.Is(err, store.ErrNotFound) {
		WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, result)
}
