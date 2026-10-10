package httpapi

import (
	"encoding/json"
	"net/http"
	"net/mail"
	"strings"

	"workshop/internal/store"
)

// customerRequest is the JSON body of POST /api/customers.
type customerRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// CreateCustomer handles POST /api/customers. It validates name, e-mail and
// phone and answers 201 with the stored customer.
func (s *Server) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	var req customerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Phone = strings.TrimSpace(req.Phone)

	if message, ok := validateCustomer(req.Name, req.Email, req.Phone); !ok {
		WriteError(w, http.StatusBadRequest, CodeValidationError, message)
		return
	}

	customer, err := store.CreateCustomer(r.Context(), s.Pool, req.Name, req.Email, req.Phone)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "could not create customer")
		return
	}
	WriteJSON(w, http.StatusCreated, customer)
}

// validateCustomer reports whether name, e-mail and phone are present and the
// e-mail is a plain, valid address.
func validateCustomer(name, email, phone string) (string, bool) {
	if name == "" {
		return "name is required", false
	}
	if email == "" {
		return "email is required", false
	}
	if !validEmail(email) {
		return "email is invalid", false
	}
	if phone == "" {
		return "phone is required", false
	}
	return "", true
}

// validEmail accepts a bare address (no display name), so "Max <a@b>" is
// rejected like any other malformed input.
func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return addr.Address == email
}
