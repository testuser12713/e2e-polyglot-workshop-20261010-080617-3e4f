package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"workshop/internal/store"
)

// orderRequestBody is the JSON body of POST /api/orders.
type orderRequestBody struct {
	Customer struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	} `json:"customer"`
	Vehicle struct {
		LicensePlate string `json:"license_plate"`
		Make         string `json:"make"`
		Model        string `json:"model"`
		MileageKM    int    `json:"mileage_km"`
	} `json:"vehicle"`
	PreferredDate      string `json:"preferred_date"`
	ProblemDescription string `json:"problem_description"`
}

// CreateOrder handles POST /api/orders. It creates or reuses the customer and
// vehicle, opens the order in status "angefragt" with a generated order number
// and answers 201. The only thing logged is the order number — never a name,
// e-mail, phone, plate or problem description.
func (s *Server) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var body orderRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "invalid JSON body")
		return
	}

	name := strings.TrimSpace(body.Customer.Name)
	email := strings.TrimSpace(body.Customer.Email)
	phone := strings.TrimSpace(body.Customer.Phone)
	if message, ok := validateCustomer(name, email, phone); !ok {
		WriteError(w, http.StatusBadRequest, CodeValidationError, message)
		return
	}

	licensePlate := strings.TrimSpace(body.Vehicle.LicensePlate)
	vehicleMake := strings.TrimSpace(body.Vehicle.Make)
	model := strings.TrimSpace(body.Vehicle.Model)
	if message, ok := validateVehicle(licensePlate, vehicleMake, model, body.Vehicle.MileageKM); !ok {
		WriteError(w, http.StatusBadRequest, CodeValidationError, message)
		return
	}

	var preferredDate *time.Time
	if raw := strings.TrimSpace(body.PreferredDate); raw != "" {
		parsed, err := time.Parse("2006-01-02", raw)
		if err != nil {
			WriteError(w, http.StatusBadRequest, CodeValidationError, "preferred_date must be in YYYY-MM-DD format")
			return
		}
		preferredDate = &parsed
	}

	order, err := store.CreateOrder(r.Context(), s.Pool, store.OrderRequest{
		Customer: store.CustomerInput{Name: name, Email: email, Phone: phone},
		Vehicle: store.VehicleInput{
			LicensePlate: licensePlate,
			Make:         vehicleMake,
			Model:        model,
			MileageKM:    body.Vehicle.MileageKM,
		},
		PreferredDate:      preferredDate,
		ProblemDescription: strings.TrimSpace(body.ProblemDescription),
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "could not create order")
		return
	}

	log.Printf("order created: %s", order.OrderNumber)
	WriteJSON(w, http.StatusCreated, order)
}
