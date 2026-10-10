package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"workshop/internal/domain"
	"workshop/internal/store"
)

// orderStatusRequest is the body of POST /api/workshop/orders/{order_number}/status.
type orderStatusRequest struct {
	Status domain.Status `json:"status"`
}

// ConfirmOrder handles POST /api/workshop/orders/{order_number}/confirm. It
// advances an "angefragt" order to "bestaetigt" and records the change in the
// status protocol. An order that is not in a confirmable state answers 409.
func (s *Server) ConfirmOrder(w http.ResponseWriter, r *http.Request) {
	orderNumber := r.PathValue("order_number")
	if orderNumber == "" {
		WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		return
	}
	if s.Pool == nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "database unavailable")
		return
	}

	if _, err := store.UpdateOrderStatus(r.Context(), s.Pool, orderNumber, domain.StatusConfirmed); err != nil {
		switch {
		case errors.Is(err, store.ErrOrderNotFound):
			WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		case errors.Is(err, store.ErrInvalidTransition):
			WriteError(w, http.StatusConflict, CodeConflict, "order cannot be confirmed from its current status")
		default:
			WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		}
		return
	}

	s.writeStatusOrderDetail(w, r, orderNumber)
}

// SetOrderStatus handles POST /api/workshop/orders/{order_number}/status. It
// moves the order to the requested status only when that is a single legal step
// along angefragt -> bestaetigt -> in_arbeit -> fertig -> abgeholt; every jump
// or backwards step answers 400 invalid_transition and leaves the status
// unchanged (AC-05). Reaching "fertig" also enqueues the invoice job (AC-08).
func (s *Server) SetOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderNumber := r.PathValue("order_number")
	if orderNumber == "" {
		WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		return
	}

	var body orderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "request body must be JSON with a status field")
		return
	}
	if !domain.IsValid(body.Status) {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "status must be one of angefragt, bestaetigt, in_arbeit, fertig, abgeholt")
		return
	}
	if s.Pool == nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "database unavailable")
		return
	}

	if _, err := store.UpdateOrderStatus(r.Context(), s.Pool, orderNumber, body.Status); err != nil {
		switch {
		case errors.Is(err, store.ErrOrderNotFound):
			WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		case errors.Is(err, store.ErrInvalidTransition):
			WriteError(w, http.StatusBadRequest, CodeInvalidTransition, "status change is not allowed")
		default:
			WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		}
		return
	}

	s.writeStatusOrderDetail(w, r, orderNumber)
}

// writeStatusOrderDetail loads and writes the full OrderDetail view. It is used by
// both status routes so they answer the same shape the workshop order endpoint
// serves.
func (s *Server) writeStatusOrderDetail(w http.ResponseWriter, r *http.Request, orderNumber string) {
	detail, err := store.LoadStatusOrderDetail(r.Context(), s.Pool, orderNumber)
	if errors.Is(err, store.ErrOrderNotFound) {
		WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, detail)
}
