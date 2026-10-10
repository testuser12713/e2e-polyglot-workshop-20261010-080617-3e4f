package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestCreateVehicleReturnsCreated(t *testing.T) {
	router := newTestRouter(t)
	suffix := intakeUniqueSuffix()

	rec := intakePostJSON(t, router, "/api/vehicles", map[string]any{
		"license_plate": "B-" + suffix,
		"make":          "Volkswagen",
		"model":         "Golf",
		"mileage_km":    123456,
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/vehicles status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	var got struct {
		ID           int64  `json:"id"`
		LicensePlate string `json:"license_plate"`
		Make         string `json:"make"`
		Model        string `json:"model"`
		MileageKM    int    `json:"mileage_km"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if got.ID == 0 {
		t.Error("vehicle id was not assigned")
	}
	if got.LicensePlate != "B-"+suffix {
		t.Errorf("license_plate = %q, want %q", got.LicensePlate, "B-"+suffix)
	}
	if got.Make != "Volkswagen" || got.Model != "Golf" {
		t.Errorf("make/model = %q/%q, want Volkswagen/Golf", got.Make, got.Model)
	}
	if got.MileageKM != 123456 {
		t.Errorf("mileage_km = %d, want 123456", got.MileageKM)
	}
}

func TestCreateVehicleRejectsDuplicatePlate(t *testing.T) {
	router := newTestRouter(t)
	plate := "M-" + intakeUniqueSuffix()

	first := intakePostJSON(t, router, "/api/vehicles", map[string]any{
		"license_plate": plate,
		"make":          "BMW",
		"model":         "320d",
		"mileage_km":    1000,
	})
	if first.Code != http.StatusCreated {
		t.Fatalf("first vehicle status = %d, want 201 (body: %s)", first.Code, first.Body.String())
	}

	second := intakePostJSON(t, router, "/api/vehicles", map[string]any{
		"license_plate": plate,
		"make":          "BMW",
		"model":         "320d",
		"mileage_km":    2000,
	})
	if second.Code != http.StatusConflict {
		t.Fatalf("duplicate plate status = %d, want 409 (body: %s)", second.Code, second.Body.String())
	}
	if code := intakeErrorCode(t, second); code != CodeConflict {
		t.Errorf("error code = %q, want %q", code, CodeConflict)
	}
}

func TestCreateVehicleRejectsMissingFields(t *testing.T) {
	router := newTestRouter(t)

	rec := intakePostJSON(t, router, "/api/vehicles", map[string]any{
		"license_plate": "K-" + intakeUniqueSuffix(),
		"model":         "Nur Modell",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing make status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := intakeErrorCode(t, rec); code != CodeValidationError {
		t.Errorf("error code = %q, want %q", code, CodeValidationError)
	}
}
