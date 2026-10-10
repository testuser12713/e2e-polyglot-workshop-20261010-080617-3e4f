package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"workshop/internal/store"
)

// vehicleRequest is the JSON body of POST /api/vehicles.
type vehicleRequest struct {
	LicensePlate string `json:"license_plate"`
	Make         string `json:"make"`
	Model        string `json:"model"`
	MileageKM    int    `json:"mileage_km"`
}

// CreateVehicle handles POST /api/vehicles. A second vehicle with the same
// license plate is rejected with 409 conflict.
func (s *Server) CreateVehicle(w http.ResponseWriter, r *http.Request) {
	var req vehicleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "invalid JSON body")
		return
	}
	req.LicensePlate = strings.TrimSpace(req.LicensePlate)
	req.Make = strings.TrimSpace(req.Make)
	req.Model = strings.TrimSpace(req.Model)

	if message, ok := validateVehicle(req.LicensePlate, req.Make, req.Model, req.MileageKM); !ok {
		WriteError(w, http.StatusBadRequest, CodeValidationError, message)
		return
	}

	vehicle, err := store.CreateVehicle(r.Context(), s.Pool, req.LicensePlate, req.Make, req.Model, req.MileageKM)
	if errors.Is(err, store.ErrDuplicatePlate) {
		WriteError(w, http.StatusConflict, CodeConflict, "a vehicle with this license plate already exists")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "could not create vehicle")
		return
	}
	WriteJSON(w, http.StatusCreated, vehicle)
}

// validateVehicle reports whether the required vehicle fields are present and
// the mileage is not negative.
func validateVehicle(licensePlate, make, model string, mileageKM int) (string, bool) {
	if licensePlate == "" {
		return "license_plate is required", false
	}
	if make == "" {
		return "make is required", false
	}
	if model == "" {
		return "model is required", false
	}
	if mileageKM < 0 {
		return "mileage_km must not be negative", false
	}
	return "", true
}
