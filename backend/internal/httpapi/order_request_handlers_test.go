package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// intakeOrderPayload builds a valid order request body.
func intakeOrderPayload(plate, email string) map[string]any {
	return map[string]any{
		"customer": map[string]any{
			"name":  "Max Muster",
			"email": email,
			"phone": "0170-1234567",
		},
		"vehicle": map[string]any{
			"license_plate": plate,
			"make":          "Audi",
			"model":         "A3",
			"mileage_km":    98765,
		},
		"preferred_date":      "2026-11-01",
		"problem_description": "Bremsen quietschen beim Bremsen",
	}
}

func TestCreateOrderReturnsRequestedOrder(t *testing.T) {
	router := newTestRouter(t)
	pool := newTestPool(t)
	suffix := intakeUniqueSuffix()

	rec := intakePostJSON(t, router, "/api/orders", intakeOrderPayload("HH-"+suffix, "order."+suffix+"@example.com"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/orders status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	var got struct {
		OrderNumber string    `json:"order_number"`
		Status      string    `json:"status"`
		CreatedAt   time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if got.OrderNumber == "" {
		t.Error("order_number is empty")
	}
	if got.Status != "angefragt" {
		t.Errorf("status = %q, want %q", got.Status, "angefragt")
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at is zero")
	}

	var storedStatus string
	if err := pool.QueryRow(context.Background(),
		`SELECT status FROM orders WHERE order_number = $1`, got.OrderNumber,
	).Scan(&storedStatus); err != nil {
		t.Fatalf("query order: %v", err)
	}
	if storedStatus != "angefragt" {
		t.Errorf("stored status = %q, want %q", storedStatus, "angefragt")
	}

	var historyCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM status_history WHERE order_id = (SELECT id FROM orders WHERE order_number = $1)`,
		got.OrderNumber,
	).Scan(&historyCount); err != nil {
		t.Fatalf("query status_history: %v", err)
	}
	if historyCount == 0 {
		t.Error("no initial status_history entry was recorded for the new order")
	}
}

func TestCreateOrderReusesCustomerAndVehicle(t *testing.T) {
	router := newTestRouter(t)
	pool := newTestPool(t)
	suffix := intakeUniqueSuffix()
	plate := "S-" + suffix
	email := "reuse." + suffix + "@example.com"

	for i := 0; i < 2; i++ {
		rec := intakePostJSON(t, router, "/api/orders", intakeOrderPayload(plate, email))
		if rec.Code != http.StatusCreated {
			t.Fatalf("attempt %d: POST /api/orders status = %d, want 201 (body: %s)", i+1, rec.Code, rec.Body.String())
		}
	}

	var vehicleCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM vehicles WHERE license_plate = $1`, plate,
	).Scan(&vehicleCount); err != nil {
		t.Fatalf("query vehicles: %v", err)
	}
	if vehicleCount != 1 {
		t.Errorf("vehicles with plate %q = %d, want 1 (vehicle was not reused)", plate, vehicleCount)
	}

	var customerCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM customers WHERE email = $1`, email,
	).Scan(&customerCount); err != nil {
		t.Fatalf("query customers: %v", err)
	}
	if customerCount != 1 {
		t.Errorf("customers with e-mail %q = %d, want 1 (customer was not reused)", email, customerCount)
	}
}

func TestCreateOrderRejectsMissingCustomer(t *testing.T) {
	router := newTestRouter(t)

	payload := intakeOrderPayload("F-"+intakeUniqueSuffix(), "missing."+intakeUniqueSuffix()+"@example.com")
	payload["customer"] = map[string]any{"email": "missing@example.com", "phone": "0170-000000"}

	rec := intakePostJSON(t, router, "/api/orders", payload)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing customer name status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := intakeErrorCode(t, rec); code != CodeValidationError {
		t.Errorf("error code = %q, want %q", code, CodeValidationError)
	}
}

func TestCreateOrderRejectsInvalidPreferredDate(t *testing.T) {
	router := newTestRouter(t)

	payload := intakeOrderPayload("F-"+intakeUniqueSuffix(), "date."+intakeUniqueSuffix()+"@example.com")
	payload["preferred_date"] = "01.11.2026"

	rec := intakePostJSON(t, router, "/api/orders", payload)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid preferred_date status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := intakeErrorCode(t, rec); code != CodeValidationError {
		t.Errorf("error code = %q, want %q", code, CodeValidationError)
	}
}
