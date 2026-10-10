package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/config"
	"workshop/internal/store"
)

// newItemsRouter wires only the route under test. The full router wraps it in
// RequireEmployee, which another ticket fills in; testing the handler directly
// keeps this slice independent of that one while still exercising the same
// path-pattern plumbing (PathValue).
func newItemsRouter(t *testing.T, pool *pgxpool.Pool) http.Handler {
	t.Helper()
	s := NewServer(pool, &config.Config{CORSOrigin: config.DefaultCORSOrigin})
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/workshop/orders/{order_number}/items", s.AddOrderItems)
	return mux
}

// insertOrderFixture creates the customer/vehicle/order rows an items test needs
// and removes them again afterwards. Only rows this test created are touched.
func insertOrderFixture(t *testing.T, pool *pgxpool.Pool, status string) string {
	t.Helper()
	ctx := context.Background()
	suffix := time.Now().UnixNano()

	var customerID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Testkunde", fmt.Sprintf("kunde-%d@example.test", suffix), "0123456789",
	).Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	var vehicleID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (license_plate, make, model, mileage_km) VALUES ($1, $2, $3, $4) RETURNING id`,
		fmt.Sprintf("TEST-%d", suffix), "VW", "Golf", 120000,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	orderNumber := fmt.Sprintf("AUF-%d", suffix)
	var orderID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO orders (order_number, customer_id, vehicle_id, preferred_date, problem_description, status)
		 VALUES ($1, $2, $3, CURRENT_DATE, $4, $5) RETURNING id`,
		orderNumber, customerID, vehicleID, "Bremsen quietschen", status,
	).Scan(&orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO status_history (order_id, status, previous_status) VALUES ($1, $2, NULL)`,
		orderID, status,
	); err != nil {
		t.Fatalf("insert status history: %v", err)
	}

	t.Cleanup(func() {
		// The order carries order_items and status_history via ON DELETE CASCADE.
		_, _ = pool.Exec(ctx, `DELETE FROM orders WHERE id = $1`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, customerID)
		_, _ = pool.Exec(ctx, `DELETE FROM vehicles WHERE id = $1`, vehicleID)
	})
	return orderNumber
}

func postItems(t *testing.T, router http.Handler, orderNumber, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/api/workshop/orders/"+orderNumber+"/items", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeErrorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error body is not the shared envelope: %v (body: %s)", err, rec.Body.String())
	}
	return envelope.Error.Code
}

func TestAddOrderItemsCapturesLaborAndPart(t *testing.T) {
	pool := newTestPool(t)
	router := newItemsRouter(t, pool)
	orderNumber := insertOrderFixture(t, pool, "angefragt")

	body := `{"items":[
		{"kind":"labor","description":"Bremsbelag wechseln","quantity":1,"unit_price_cents":9500},
		{"kind":"part","description":"Bremsbelag","quantity":2,"unit_price_cents":2500}
	]}`
	rec := postItems(t, router, orderNumber, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST items status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var detail store.OrderItemsDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("response is not OrderDetail JSON: %v", err)
	}
	if detail.OrderNumber != orderNumber {
		t.Errorf("order_number = %q, want %q", detail.OrderNumber, orderNumber)
	}
	if detail.Status != "angefragt" {
		t.Errorf("status = %q, want angefragt", detail.Status)
	}
	if len(detail.Items) != 2 {
		t.Fatalf("items length = %d, want 2 (body: %s)", len(detail.Items), rec.Body.String())
	}
	if detail.Items[0].Kind != "labor" || detail.Items[0].TotalCents != 9500 {
		t.Errorf("labor item = %+v, want kind labor total 9500", detail.Items[0])
	}
	if detail.Items[1].Kind != "part" || detail.Items[1].TotalCents != 5000 {
		t.Errorf("part item = %+v, want kind part total 5000", detail.Items[1])
	}

	// The positions must be durably stored, not only echoed back.
	var storedCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM order_items oi
		 JOIN orders o ON o.id = oi.order_id
		 WHERE o.order_number = $1`, orderNumber,
	).Scan(&storedCount); err != nil {
		t.Fatalf("count stored items: %v", err)
	}
	if storedCount != 2 {
		t.Errorf("stored items = %d, want 2", storedCount)
	}
}

func TestAddOrderItemsComputesIntegerCentTotal(t *testing.T) {
	pool := newTestPool(t)
	router := newItemsRouter(t, pool)
	orderNumber := insertOrderFixture(t, pool, "bestaetigt")

	body := `{"items":[{"kind":"part","description":"Ölfilter","quantity":3,"unit_price_cents":1999}]}`
	rec := postItems(t, router, orderNumber, body)

	if rec.Code != http.StatusOK {
		t.Fatalf("POST items status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var detail store.OrderItemsDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("response is not OrderDetail JSON: %v", err)
	}
	if len(detail.Items) != 1 {
		t.Fatalf("items length = %d, want 1", len(detail.Items))
	}
	if got := detail.Items[0].TotalCents; got != 3*1999 {
		t.Errorf("total_cents = %d, want %d", got, 3*1999)
	}
}

func TestAddOrderItemsRejectsPickedUpOrder(t *testing.T) {
	pool := newTestPool(t)
	router := newItemsRouter(t, pool)
	orderNumber := insertOrderFixture(t, pool, "abgeholt")

	body := `{"items":[{"kind":"labor","description":"Nacharbeit","quantity":1,"unit_price_cents":1000}]}`
	rec := postItems(t, router, orderNumber, body)

	if rec.Code != http.StatusConflict {
		t.Fatalf("POST items on picked-up order status = %d, want 409 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := decodeErrorCode(t, rec); code != CodeConflict {
		t.Errorf("error code = %q, want %q", code, CodeConflict)
	}

	var storedCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM order_items oi
		 JOIN orders o ON o.id = oi.order_id
		 WHERE o.order_number = $1`, orderNumber,
	).Scan(&storedCount); err != nil {
		t.Fatalf("count stored items: %v", err)
	}
	if storedCount != 0 {
		t.Errorf("stored items = %d, want 0 after a rejected capture", storedCount)
	}
}

func TestAddOrderItemsRejectsEmptyArray(t *testing.T) {
	pool := newTestPool(t)
	router := newItemsRouter(t, pool)
	orderNumber := insertOrderFixture(t, pool, "angefragt")

	rec := postItems(t, router, orderNumber, `{"items":[]}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST empty items status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := decodeErrorCode(t, rec); code != CodeValidationError {
		t.Errorf("error code = %q, want %q", code, CodeValidationError)
	}
}

func TestAddOrderItemsRejectsInvalidEntries(t *testing.T) {
	pool := newTestPool(t)
	router := newItemsRouter(t, pool)
	orderNumber := insertOrderFixture(t, pool, "angefragt")

	cases := map[string]string{
		"unknown kind":      `{"items":[{"kind":"flight","description":"x","quantity":1,"unit_price_cents":100}]}`,
		"empty description": `{"items":[{"kind":"labor","description":"","quantity":1,"unit_price_cents":100}]}`,
		"zero quantity":     `{"items":[{"kind":"labor","description":"x","quantity":0,"unit_price_cents":100}]}`,
		"negative price":    `{"items":[{"kind":"part","description":"x","quantity":1,"unit_price_cents":-1}]}`,
		"missing quantity":  `{"items":[{"kind":"labor","description":"x","unit_price_cents":100}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			rec := postItems(t, router, orderNumber, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
			}
			if code := decodeErrorCode(t, rec); code != CodeValidationError {
				t.Errorf("error code = %q, want %q", code, CodeValidationError)
			}
		})
	}
}

func TestAddOrderItemsUnknownOrderIsNotFound(t *testing.T) {
	pool := newTestPool(t)
	router := newItemsRouter(t, pool)

	body := `{"items":[{"kind":"labor","description":"x","quantity":1,"unit_price_cents":100}]}`
	rec := postItems(t, router, "AUF-DOES-NOT-EXIST", body)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := decodeErrorCode(t, rec); code != CodeNotFound {
		t.Errorf("error code = %q, want %q", code, CodeNotFound)
	}
}
