package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"workshop/internal/domain"
	"workshop/internal/queue"
	"workshop/internal/store"
)

// statusErrorEnvelope mirrors the shared error contract for decoding in tests.
type statusErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// insertStatusFixtureOrder creates a customer, a vehicle and one order, all with
// unique values, and removes them again when the test ends. Only the rows this
// test created are touched.
func insertStatusFixtureOrder(t *testing.T, pool *pgxpool.Pool, initial domain.Status) string {
	t.Helper()
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())

	var customerID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Status Testkunde", "status-test-"+suffix+"@example.com", "0123456789",
	).Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	var vehicleID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (license_plate, make, model, mileage_km) VALUES ($1, $2, $3, $4) RETURNING id`,
		"ST-"+suffix, "VW", "Golf", 12345,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	orderNumber := "STATUS-" + suffix
	var orderID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO orders (order_number, customer_id, vehicle_id, problem_description, status)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		orderNumber, customerID, vehicleID, "Bremsen quietschen", string(initial),
	).Scan(&orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM orders WHERE id = $1`, orderID)
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, customerID)
		_, _ = pool.Exec(ctx, `DELETE FROM vehicles WHERE id = $1`, vehicleID)
	})
	return orderNumber
}

// configureTestQueue points the package-level queue at a fresh Valkey list so a
// test never reads another test's message.
func configureTestQueue(t *testing.T) string {
	t.Helper()
	valkeyURL := os.Getenv("VALKEY_URL")
	if valkeyURL == "" {
		t.Skip("VALKEY_URL is not set; the test needs a real Valkey")
	}
	key := fmt.Sprintf("test_invoices_%d", time.Now().UnixNano())
	if err := queue.Configure(valkeyURL, key); err != nil {
		t.Fatalf("configure queue: %v", err)
	}
	t.Cleanup(func() { _ = queue.Close() })
	return key
}

func readQueueHead(t *testing.T, key string) []string {
	t.Helper()
	options, err := redis.ParseURL(os.Getenv("VALKEY_URL"))
	if err != nil {
		t.Fatalf("parse VALKEY_URL: %v", err)
	}
	client := redis.NewClient(options)
	defer client.Close()
	items, err := client.LRange(context.Background(), key, 0, -1).Result()
	if err != nil {
		t.Fatalf("read queue %s: %v", key, err)
	}
	return items
}

func doSetStatus(t *testing.T, s *Server, orderNumber string, target domain.Status) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"status": string(target)})
	req := httptest.NewRequest(http.MethodPost, "/api/workshop/orders/"+orderNumber+"/status", bytes.NewReader(body))
	req.SetPathValue("order_number", orderNumber)
	rec := httptest.NewRecorder()
	s.SetOrderStatus(rec, req)
	return rec
}

func doConfirm(t *testing.T, s *Server, orderNumber string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/workshop/orders/"+orderNumber+"/confirm", nil)
	req.SetPathValue("order_number", orderNumber)
	rec := httptest.NewRecorder()
	s.ConfirmOrder(rec, req)
	return rec
}

func storedStatus(t *testing.T, pool *pgxpool.Pool, orderNumber string) domain.Status {
	t.Helper()
	var status domain.Status
	if err := pool.QueryRow(context.Background(),
		`SELECT status FROM orders WHERE order_number = $1`, orderNumber,
	).Scan(&status); err != nil {
		t.Fatalf("read stored status: %v", err)
	}
	return status
}

func decodeStatusDetail(t *testing.T, rec *httptest.ResponseRecorder) store.StatusOrderDetail {
	t.Helper()
	var detail store.StatusOrderDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode OrderDetail: %v (body: %s)", err, rec.Body.String())
	}
	return detail
}

func TestConfirmOrderRecordsHistory(t *testing.T) {
	pool := newTestPool(t)
	s := NewServer(pool, nil)
	orderNumber := insertStatusFixtureOrder(t, pool, domain.StatusRequested)

	rec := doConfirm(t, s, orderNumber)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirm status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	detail := decodeStatusDetail(t, rec)
	if detail.Status != domain.StatusConfirmed {
		t.Errorf("detail status = %q, want %q", detail.Status, domain.StatusConfirmed)
	}

	history, err := store.LoadStatusHistory(context.Background(), pool, orderNumber)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	if len(history) != 1 || history[0].Status != domain.StatusConfirmed {
		t.Fatalf("history = %+v, want exactly one bestaetigt entry", history)
	}
	if detail.Customer.Name == "" || detail.Vehicle.LicensePlate == "" {
		t.Errorf("detail is missing customer or vehicle data: %+v", detail)
	}
}

func TestSetOrderStatusWalksToDoneAndEnqueuesInvoice(t *testing.T) {
	pool := newTestPool(t)
	key := configureTestQueue(t)
	s := NewServer(pool, nil)
	orderNumber := insertStatusFixtureOrder(t, pool, domain.StatusRequested)

	for _, target := range []domain.Status{domain.StatusConfirmed, domain.StatusInProgress, domain.StatusDone} {
		rec := doSetStatus(t, s, orderNumber, target)
		if rec.Code != http.StatusOK {
			t.Fatalf("status -> %s returned %d, want 200 (body: %s)", target, rec.Code, rec.Body.String())
		}
		if got := decodeStatusDetail(t, rec).Status; got != target {
			t.Fatalf("detail status = %q, want %q", got, target)
		}
	}

	history, err := store.LoadStatusHistory(context.Background(), pool, orderNumber)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	want := []domain.Status{domain.StatusConfirmed, domain.StatusInProgress, domain.StatusDone}
	if len(history) != len(want) {
		t.Fatalf("history length = %d, want %d (%+v)", len(history), len(want), history)
	}
	for i, expected := range want {
		if history[i].Status != expected {
			t.Errorf("history[%d] = %q, want %q", i, history[i].Status, expected)
		}
		if history[i].ChangedAt.IsZero() {
			t.Errorf("history[%d] has no timestamp", i)
		}
	}

	head := readQueueHead(t, key)
	expectedJob := fmt.Sprintf(`{"order_number":%q}`, orderNumber)
	if len(head) != 1 {
		t.Fatalf("queue length = %d, want exactly 1 job: %v", len(head), head)
	}
	if head[0] != expectedJob {
		t.Errorf("queue head = %s, want %s", head[0], expectedJob)
	}
}

func TestSetOrderStatusRejectsJumpAndKeepsStatus(t *testing.T) {
	pool := newTestPool(t)
	s := NewServer(pool, nil)
	orderNumber := insertStatusFixtureOrder(t, pool, domain.StatusRequested)

	rec := doSetStatus(t, s, orderNumber, domain.StatusInProgress)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("jump status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	var envelope statusErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if envelope.Error.Code != CodeInvalidTransition {
		t.Errorf("error code = %q, want %q", envelope.Error.Code, CodeInvalidTransition)
	}
	if got := storedStatus(t, pool, orderNumber); got != domain.StatusRequested {
		t.Errorf("stored status = %q, want unchanged %q", got, domain.StatusRequested)
	}
	history, err := store.LoadStatusHistory(context.Background(), pool, orderNumber)
	if err != nil {
		t.Fatalf("load history: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("history = %+v, want empty after a rejected transition", history)
	}
}

func TestSetOrderStatusRejectsBackwardsStep(t *testing.T) {
	pool := newTestPool(t)
	s := NewServer(pool, nil)
	orderNumber := insertStatusFixtureOrder(t, pool, domain.StatusRequested)

	if rec := doSetStatus(t, s, orderNumber, domain.StatusConfirmed); rec.Code != http.StatusOK {
		t.Fatalf("forward step status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}
	rec := doSetStatus(t, s, orderNumber, domain.StatusRequested)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("backwards step status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	if got := storedStatus(t, pool, orderNumber); got != domain.StatusConfirmed {
		t.Errorf("stored status = %q, want unchanged %q", got, domain.StatusConfirmed)
	}
}

func TestSetOrderStatusRejectsUnknownOrder(t *testing.T) {
	pool := newTestPool(t)
	s := NewServer(pool, nil)

	rec := doSetStatus(t, s, "DOES-NOT-EXIST", domain.StatusConfirmed)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown order status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestSetOrderStatusRejectsUnknownStatusValue(t *testing.T) {
	pool := newTestPool(t)
	s := NewServer(pool, nil)
	orderNumber := insertStatusFixtureOrder(t, pool, domain.StatusRequested)

	req := httptest.NewRequest(http.MethodPost, "/api/workshop/orders/"+orderNumber+"/status",
		bytes.NewReader([]byte(`{"status":"galaxy"}`)))
	req.SetPathValue("order_number", orderNumber)
	rec := httptest.NewRecorder()
	s.SetOrderStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown status value = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	var envelope statusErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if envelope.Error.Code != CodeValidationError {
		t.Errorf("error code = %q, want %q", envelope.Error.Code, CodeValidationError)
	}
}

func TestConfirmOrderRejectsAlreadyConfirmed(t *testing.T) {
	pool := newTestPool(t)
	s := NewServer(pool, nil)
	orderNumber := insertStatusFixtureOrder(t, pool, domain.StatusConfirmed)

	rec := doConfirm(t, s, orderNumber)
	if rec.Code != http.StatusConflict {
		t.Fatalf("confirm on confirmed order = %d, want 409 (body: %s)", rec.Code, rec.Body.String())
	}
	var envelope statusErrorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if envelope.Error.Code != CodeConflict {
		t.Errorf("error code = %q, want %q", envelope.Error.Code, CodeConflict)
	}
}
