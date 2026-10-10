package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/config"
)

var lookupSeq int64

// seededLookupOrder is the row set one test provisions for the customer lookup.
// It deliberately includes contact and vehicle data so the tests can prove the
// lookup response never carries any of it (AC-31).
type seededLookupOrder struct {
	orderNumber string
	plate       string
	customerID  int64
	vehicleID   int64
	orderID     int64
	email       string
	make        string
	model       string
}

// seedLookupOrder inserts its own customer, vehicle, order, status history and —
// when withInvoice is true — an invoice with two items. Everything is namespaced
// by orderNumber so the test only ever touches the rows it created.
func seedLookupOrder(t *testing.T, pool *pgxpool.Pool, orderNumber, plate string, withInvoice bool) seededLookupOrder {
	t.Helper()
	ctx := context.Background()

	seeded := seededLookupOrder{
		orderNumber: orderNumber,
		plate:       plate,
		email:       "lookup-" + orderNumber + "@example.test",
		make:        "Testmarke",
		model:       "Testmodell",
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Test Kunde", seeded.email, "+49 000 000000",
	).Scan(&seeded.customerID); err != nil {
		t.Fatalf("seed customer: %v", err)
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (license_plate, make, model, mileage_km) VALUES ($1, $2, $3, $4) RETURNING id`,
		plate, seeded.make, seeded.model, 12345,
	).Scan(&seeded.vehicleID); err != nil {
		t.Fatalf("seed vehicle: %v", err)
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO orders (order_number, customer_id, vehicle_id, problem_description, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		orderNumber, seeded.customerID, seeded.vehicleID, "Testproblem", "bestaetigt",
	).Scan(&seeded.orderID); err != nil {
		t.Fatalf("seed order: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO status_history (order_id, status, previous_status, changed_at)
		 VALUES ($1, 'angefragt', NULL, now() - interval '1 hour'),
		        ($1, 'bestaetigt', 'angefragt', now())`,
		seeded.orderID,
	); err != nil {
		t.Fatalf("seed status history: %v", err)
	}

	if withInvoice {
		var invoiceID int64
		if err := pool.QueryRow(ctx,
			`INSERT INTO invoices (order_id, order_number, net_cents, vat_cents, gross_cents)
			 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			seeded.orderID, orderNumber, 10000, 1900, 11900,
		).Scan(&invoiceID); err != nil {
			t.Fatalf("seed invoice: %v", err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO invoice_items (invoice_id, kind, description, quantity, unit_price_cents, total_cents)
			 VALUES ($1, 'labor', 'Arbeitszeit', 2, 4000, 8000),
			        ($1, 'part', 'Bremsbelag', 1, 2000, 2000)`,
			invoiceID,
		); err != nil {
			t.Fatalf("seed invoice items: %v", err)
		}
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM orders WHERE id = $1`, seeded.orderID); err != nil {
			t.Errorf("cleanup order: %v", err)
		}
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM vehicles WHERE id = $1`, seeded.vehicleID); err != nil {
			t.Errorf("cleanup vehicle: %v", err)
		}
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM customers WHERE id = $1`, seeded.customerID); err != nil {
			t.Errorf("cleanup customer: %v", err)
		}
	})

	return seeded
}

func lookupRouter(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	pool := newTestPool(t)
	cfg := &config.Config{CORSOrigin: config.DefaultCORSOrigin}
	return NewRouter(NewServer(pool, cfg)), pool
}

func uniqueSuffix() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), atomic.AddInt64(&lookupSeq, 1))
}

func doGet(t *testing.T, router http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestOrderStatusLookupSuccess proves that a matching order number and plate
// return the order's status and its full status history.
func TestOrderStatusLookupSuccess(t *testing.T) {
	router, pool := lookupRouter(t)
	suffix := uniqueSuffix()
	orderNumber := "LOOKUP-" + suffix
	plate := "B-LK" + suffix
	seedLookupOrder(t, pool, orderNumber, plate, false)

	rec := doGet(t, router, "/api/orders/status?order_number="+orderNumber+"&license_plate="+plate)
	if rec.Code != http.StatusOK {
		t.Fatalf("status lookup = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var body struct {
		OrderNumber   string `json:"order_number"`
		Status        string `json:"status"`
		StatusHistory []struct {
			Status    string `json:"status"`
			ChangedAt string `json:"changed_at"`
		} `json:"status_history"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("status body is not JSON: %v", err)
	}
	if body.OrderNumber != orderNumber {
		t.Errorf("order_number = %q, want %q", body.OrderNumber, orderNumber)
	}
	if body.Status != "bestaetigt" {
		t.Errorf("status = %q, want %q", body.Status, "bestaetigt")
	}
	if len(body.StatusHistory) != 2 {
		t.Fatalf("status_history length = %d, want 2", len(body.StatusHistory))
	}
	if body.StatusHistory[0].Status != "angefragt" || body.StatusHistory[1].Status != "bestaetigt" {
		t.Errorf("status_history order = %q,%q, want angefragt,bestaetigt",
			body.StatusHistory[0].Status, body.StatusHistory[1].Status)
	}
	if body.StatusHistory[0].ChangedAt == "" {
		t.Error("status_history[0].changed_at is empty")
	}
}

// TestOrderStatusLookupWrongPlateIsNotFound proves a wrong plate yields 404.
func TestOrderStatusLookupWrongPlateIsNotFound(t *testing.T) {
	router, pool := lookupRouter(t)
	suffix := uniqueSuffix()
	orderNumber := "LOOKUP-" + suffix
	plate := "B-LK" + suffix
	seedLookupOrder(t, pool, orderNumber, plate, false)

	rec := doGet(t, router, "/api/orders/status?order_number="+orderNumber+"&license_plate=B-WRONG1")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("wrong plate = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
	assertNotFoundEnvelope(t, rec)
}

// TestOrderStatusLookupUnknownOrderIsNotFound proves an unknown order yields 404.
func TestOrderStatusLookupUnknownOrderIsNotFound(t *testing.T) {
	router, _ := lookupRouter(t)

	rec := doGet(t, router, "/api/orders/status?order_number=DOES-NOT-EXIST&license_plate=B-XXXX1")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown order = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
	assertNotFoundEnvelope(t, rec)
}

// TestOrderStatusLookupMissingParamsIsValidationError proves a missing parameter
// is rejected as a validation error rather than exposing any lookup result.
func TestOrderStatusLookupMissingParamsIsValidationError(t *testing.T) {
	router, _ := lookupRouter(t)

	for _, target := range []string{
		"/api/orders/status",
		"/api/orders/status?order_number=ABC",
		"/api/orders/status?license_plate=B-LK123",
	} {
		rec := doGet(t, router, target)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400 (body: %s)", target, rec.Code, rec.Body.String())
		}
		assertErrorCode(t, rec, CodeValidationError)
	}
}

// TestOrderStatusLookupLeaksNoPersonalData is the AC-31 guard: the response
// carries exclusively the lookup shape — no email, phone or vehicle data.
func TestOrderStatusLookupLeaksNoPersonalData(t *testing.T) {
	router, pool := lookupRouter(t)
	suffix := uniqueSuffix()
	orderNumber := "LOOKUP-" + suffix
	plate := "B-LK" + suffix
	seeded := seedLookupOrder(t, pool, orderNumber, plate, false)

	rec := doGet(t, router, "/api/orders/status?order_number="+orderNumber+"&license_plate="+plate)
	if rec.Code != http.StatusOK {
		t.Fatalf("status lookup = %d, want 200", rec.Code)
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("status body is not JSON: %v", err)
	}
	assertExactKeys(t, body, "order_number", "status", "status_history")

	bodyText := rec.Body.String()
	for _, forbidden := range []string{seeded.email, "phone", "customer", "vehicle", seeded.make, seeded.model, seeded.plate} {
		if strings.Contains(bodyText, forbidden) {
			t.Errorf("status response contains %q, which must never be exposed", forbidden)
		}
	}
}

// TestOrderInvoiceLookupSuccess proves a matching pair returns the invoice with
// its items and totals.
func TestOrderInvoiceLookupSuccess(t *testing.T) {
	router, pool := lookupRouter(t)
	suffix := uniqueSuffix()
	orderNumber := "INV-" + suffix
	plate := "B-IV" + suffix
	seedLookupOrder(t, pool, orderNumber, plate, true)

	rec := doGet(t, router, "/api/orders/"+orderNumber+"/invoice?license_plate="+plate)
	if rec.Code != http.StatusOK {
		t.Fatalf("invoice lookup = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var body struct {
		OrderNumber string `json:"order_number"`
		Items       []struct {
			Kind           string `json:"kind"`
			Description    string `json:"description"`
			Quantity       int    `json:"quantity"`
			UnitPriceCents int    `json:"unit_price_cents"`
			TotalCents     int    `json:"total_cents"`
		} `json:"items"`
		NetCents   int    `json:"net_cents"`
		VatCents   int    `json:"vat_cents"`
		GrossCents int    `json:"gross_cents"`
		CreatedAt  string `json:"created_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invoice body is not JSON: %v", err)
	}
	if body.OrderNumber != orderNumber {
		t.Errorf("order_number = %q, want %q", body.OrderNumber, orderNumber)
	}
	if body.NetCents != 10000 || body.VatCents != 1900 || body.GrossCents != 11900 {
		t.Errorf("totals = %d/%d/%d, want 10000/1900/11900", body.NetCents, body.VatCents, body.GrossCents)
	}
	if body.CreatedAt == "" {
		t.Error("created_at is empty")
	}
	if len(body.Items) != 2 {
		t.Fatalf("items length = %d, want 2", len(body.Items))
	}
	if body.Items[0].Kind != "labor" || body.Items[0].Quantity != 2 || body.Items[0].TotalCents != 8000 {
		t.Errorf("item[0] = %+v, want labor qty 2 total 8000", body.Items[0])
	}
	if body.Items[1].Kind != "part" || body.Items[1].TotalCents != 2000 {
		t.Errorf("item[1] = %+v, want part total 2000", body.Items[1])
	}
}

// TestOrderInvoiceLookupWrongPlateIsNotFound proves a wrong plate yields 404.
func TestOrderInvoiceLookupWrongPlateIsNotFound(t *testing.T) {
	router, pool := lookupRouter(t)
	suffix := uniqueSuffix()
	orderNumber := "INV-" + suffix
	plate := "B-IV" + suffix
	seedLookupOrder(t, pool, orderNumber, plate, true)

	rec := doGet(t, router, "/api/orders/"+orderNumber+"/invoice?license_plate=B-WRONG1")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("wrong plate = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
	assertNotFoundEnvelope(t, rec)
}

// TestOrderInvoiceLookupUnknownOrderIsNotFound proves an unknown order yields 404.
func TestOrderInvoiceLookupUnknownOrderIsNotFound(t *testing.T) {
	router, _ := lookupRouter(t)

	rec := doGet(t, router, "/api/orders/DOES-NOT-EXIST/invoice?license_plate=B-XXXX1")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown order = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
	assertNotFoundEnvelope(t, rec)
}

// TestOrderInvoiceLookupMissingPlateIsValidationError proves the absent plate is
// a validation error, not a lookup result.
func TestOrderInvoiceLookupMissingPlateIsValidationError(t *testing.T) {
	router, _ := lookupRouter(t)

	rec := doGet(t, router, "/api/orders/ABC-123/invoice")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing plate = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	assertErrorCode(t, rec, CodeValidationError)
}

// TestLookupWrongPlateAndUnknownOrderAreIndistinguishable is the "no hint"
// guarantee: a wrong plate and an unknown order number produce byte-identical
// 404 bodies, so neither reveals whether the order exists.
func TestLookupWrongPlateAndUnknownOrderAreIndistinguishable(t *testing.T) {
	router, pool := lookupRouter(t)
	suffix := uniqueSuffix()
	orderNumber := "LOOKUP-" + suffix
	plate := "B-LK" + suffix
	seedLookupOrder(t, pool, orderNumber, plate, false)

	wrongPlate := doGet(t, router, "/api/orders/status?order_number="+orderNumber+"&license_plate=B-WRONG1")
	unknownOrder := doGet(t, router, "/api/orders/status?order_number=DOES-NOT-EXIST&license_plate="+plate)

	if wrongPlate.Code != unknownOrder.Code {
		t.Errorf("status codes differ: wrong plate %d, unknown order %d", wrongPlate.Code, unknownOrder.Code)
	}
	if wrongPlate.Body.String() != unknownOrder.Body.String() {
		t.Errorf("404 bodies differ:\n wrong plate: %s\n unknown order: %s", wrongPlate.Body.String(), unknownOrder.Body.String())
	}
}

func assertNotFoundEnvelope(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assertErrorCode(t, rec, CodeNotFound)
}

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if envelope.Error.Code != want {
		t.Errorf("error code = %q, want %q (body: %s)", envelope.Error.Code, want, rec.Body.String())
	}
	if envelope.Error.Message == "" {
		t.Error("error message is empty")
	}
}

func assertExactKeys(t *testing.T, body map[string]json.RawMessage, want ...string) {
	t.Helper()
	if len(body) != len(want) {
		t.Errorf("response has %d keys, want %d (%v)", len(body), len(want), keysOf(body))
	}
	for _, key := range want {
		if _, ok := body[key]; !ok {
			t.Errorf("response is missing key %q (%v)", key, keysOf(body))
		}
	}
}

func keysOf(body map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(body))
	for key := range body {
		keys = append(keys, key)
	}
	return keys
}
