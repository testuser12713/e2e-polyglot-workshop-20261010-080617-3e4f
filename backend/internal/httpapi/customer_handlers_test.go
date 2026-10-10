package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// intakePostJSON sends body as JSON to path through router and returns the
// recorder.
func intakePostJSON(t *testing.T, router http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// intakeErrorCode decodes the machine-readable code from the shared error
// envelope.
func intakeErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error body is not JSON: %v (body: %s)", err, rec.Body.String())
	}
	if envelope.Error.Message == "" {
		t.Errorf("error envelope has no message (body: %s)", rec.Body.String())
	}
	return envelope.Error.Code
}

// intakeUniqueSuffix returns a random token used to keep test rows apart across
// runs and parallel tests.
func intakeUniqueSuffix() string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return strings.ToUpper(hex.EncodeToString(buf))
}

func TestCreateCustomerReturnsCreated(t *testing.T) {
	router := newTestRouter(t)
	suffix := intakeUniqueSuffix()

	rec := intakePostJSON(t, router, "/api/customers", map[string]any{
		"name":  "Anna Beispiel",
		"email": "anna." + suffix + "@example.com",
		"phone": "0151-1234567",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/customers status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	var got struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if got.ID == 0 {
		t.Error("customer id was not assigned")
	}
	if got.Name != "Anna Beispiel" {
		t.Errorf("name = %q, want %q", got.Name, "Anna Beispiel")
	}
	if got.Email != "anna."+suffix+"@example.com" {
		t.Errorf("email = %q, want %q", got.Email, "anna."+suffix+"@example.com")
	}
	if got.Phone != "0151-1234567" {
		t.Errorf("phone = %q, want %q", got.Phone, "0151-1234567")
	}
}

func TestCreateCustomerRejectsMissingField(t *testing.T) {
	router := newTestRouter(t)

	rec := intakePostJSON(t, router, "/api/customers", map[string]any{
		"name":  "Ohne Telefon",
		"email": "ohne.telefon@example.com",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing phone status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := intakeErrorCode(t, rec); code != CodeValidationError {
		t.Errorf("error code = %q, want %q", code, CodeValidationError)
	}
}

func TestCreateCustomerRejectsInvalidEmail(t *testing.T) {
	router := newTestRouter(t)

	rec := intakePostJSON(t, router, "/api/customers", map[string]any{
		"name":  "Falsche Mail",
		"email": "not-an-email",
		"phone": "0170-000000",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid email status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := intakeErrorCode(t, rec); code != CodeValidationError {
		t.Errorf("error code = %q, want %q", code, CodeValidationError)
	}
}

// TestCreateCustomerIsPersisted confirms the customer is really written to
// PostgreSQL, not answered from memory.
func TestCreateCustomerIsPersisted(t *testing.T) {
	router := newTestRouter(t)
	pool := newTestPool(t)
	suffix := intakeUniqueSuffix()
	email := "persist." + suffix + "@example.com"

	rec := intakePostJSON(t, router, "/api/customers", map[string]any{
		"name":  "Persistenz",
		"email": email,
		"phone": "0170-999999",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /api/customers status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM customers WHERE email = $1`, email,
	).Scan(&count); err != nil {
		t.Fatalf("query customer: %v", err)
	}
	if count != 1 {
		t.Errorf("customers with e-mail %q = %d, want 1", email, count)
	}
}
