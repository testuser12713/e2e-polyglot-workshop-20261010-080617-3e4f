package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"workshop/internal/store"
)

// invoiceItemInput mirrors the ItemInput contract: an InvoiceItem without the
// derived total_cents. Pointers distinguish a missing field from a zero value so
// "quantity: 0" and a missing quantity are both rejected as invalid.
type invoiceItemInput struct {
	Kind           string `json:"kind"`
	Description    string `json:"description"`
	Quantity       *int   `json:"quantity"`
	UnitPriceCents *int   `json:"unit_price_cents"`
}

type addOrderItemsRequest struct {
	Items []invoiceItemInput `json:"items"`
}

// AddOrderItems handles POST /api/workshop/orders/{order_number}/items. It
// captures one or more labor/part positions and answers 200 with the updated
// OrderDetail. An empty or invalid list is a 400 validation_error; an order that
// has already been picked up (or rejected) is a 409 conflict; an unknown order
// is a 404 not_found. The route is registered behind RequireEmployee.
func (s *Server) AddOrderItems(w http.ResponseWriter, r *http.Request) {
	orderNumber := r.PathValue("order_number")

	var body addOrderItemsRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "request body must be a JSON object with an items array")
		return
	}

	items, ok := validateItems(body.Items)
	if !ok {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "items must be a non-empty array of valid labor or part positions")
		return
	}

	detail, err := store.AddOrderItems(r.Context(), s.Pool, orderNumber, items)
	switch {
	case errors.Is(err, store.ErrOrderNotFound):
		WriteError(w, http.StatusNotFound, CodeNotFound, "order not found")
		return
	case errors.Is(err, store.ErrOrderNotChangeable):
		WriteError(w, http.StatusConflict, CodeConflict, "positions cannot be added to a picked-up or rejected order")
		return
	case err != nil:
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "could not capture the positions")
		return
	}

	WriteJSON(w, http.StatusOK, detail)
}

// validateItems checks the array is non-empty and every entry follows the
// contract. It returns the normalised inputs or false when any entry is invalid.
func validateItems(raw []invoiceItemInput) ([]store.ItemInput, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	items := make([]store.ItemInput, 0, len(raw))
	for _, entry := range raw {
		kind := strings.TrimSpace(entry.Kind)
		if kind != "labor" && kind != "part" {
			return nil, false
		}
		description := strings.TrimSpace(entry.Description)
		if description == "" {
			return nil, false
		}
		if entry.Quantity == nil || *entry.Quantity <= 0 {
			return nil, false
		}
		if entry.UnitPriceCents == nil || *entry.UnitPriceCents < 0 {
			return nil, false
		}
		items = append(items, store.ItemInput{
			Kind:           kind,
			Description:    description,
			Quantity:       *entry.Quantity,
			UnitPriceCents: *entry.UnitPriceCents,
		})
	}
	return items, true
}
