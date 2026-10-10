package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/config"
	"workshop/internal/store"
)

var workshopSeedCounter atomic.Int64

// newWorkshopListPool connects to the real PostgreSQL from DATABASE_URL and
// applies the migrations. There is no SQLite anywhere (AC-21). When DATABASE_URL
// is unset the test is skipped; the pipeline always provides it.
func newWorkshopListPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; the test needs a real PostgreSQL")
	}
	ctx := context.Background()
	pool, err := store.NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := store.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

// seedWorkshopOrder inserts one customer, vehicle, order, order item and status
// history entry and returns the generated order number. Everything it created is
// removed again on cleanup, so it never depends on state from an earlier run and
// never deletes rows another ticket owns.
func seedWorkshopOrder(t *testing.T, pool *pgxpool.Pool, status, plate string, createdAt time.Time, itemTotalCents int64) string {
	t.Helper()
	ctx := context.Background()

	suffix := fmt.Sprintf("%d-%d", time.Now().UnixNano(), workshopSeedCounter.Add(1))
	orderNumber := "TEST-" + suffix
	email := "kunde-" + suffix + "@example.com"

	var customerID, vehicleID, orderID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Testkunde", email, "0123456789",
	).Scan(&customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (license_plate, make, model, mileage_km) VALUES ($1, $2, $3, $4) RETURNING id`,
		plate, "VW", "Golf", 12345,
	).Scan(&vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO orders (order_number, customer_id, vehicle_id, preferred_date, problem_description, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		orderNumber, customerID, vehicleID, "2026-01-02", "Bremsen quietschen", status, createdAt,
	).Scan(&orderID); err != nil {
		t.Fatalf("insert order: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO order_items (order_id, kind, description, quantity, unit_price_cents, total_cents)
		 VALUES ($1, 'labor', 'Arbeitszeit', 2, 9500, $2)`,
		orderID, itemTotalCents,
	); err != nil {
		t.Fatalf("insert order item: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO status_history (order_id, status, changed_at) VALUES ($1, $2, $3)`,
		orderID, status, createdAt,
	); err != nil {
		t.Fatalf("insert status history: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx := context.Background()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM orders WHERE order_number = $1`, orderNumber)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM vehicles WHERE license_plate = $1`, plate)
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM customers WHERE email = $1`, email)
	})
	return orderNumber
}

// workshopListMux registers the two routes under test without the employee
// middleware, so the handler response can be exercised directly. The full router
// (with RequireEmployee) is used in the authentication test.
func workshopListMux(s *Server) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/workshop/orders", s.ListWorkshopOrders)
	mux.HandleFunc("GET /api/workshop/orders/{order_number}", s.GetWorkshopOrder)
	return mux
}

func getWorkshopOrderList(t *testing.T, handler http.Handler, query url.Values) (int, []store.WorkshopOrderSummary) {
	t.Helper()
	path := "/api/workshop/orders"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want 200 (body: %s)", path, rec.Code, rec.Body.String())
	}
	var body struct {
		Orders []store.WorkshopOrderSummary `json:"orders"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("list body is not JSON: %v (body: %s)", err, rec.Body.String())
	}
	return rec.Code, body.Orders
}

func findOrder(orders []store.WorkshopOrderSummary, orderNumber string) (int, bool) {
	for i, o := range orders {
		if o.OrderNumber == orderNumber {
			return i, true
		}
	}
	return -1, false
}

func TestListWorkshopOrdersReturnsSummariesNewestFirst(t *testing.T) {
	pool := newWorkshopListPool(t)
	s := NewServer(pool, &config.Config{})
	handler := workshopListMux(s)

	newer := seedWorkshopOrder(t, pool, "bestaetigt", "B-AB 1234", time.Now(), 19000)
	older := seedWorkshopOrder(t, pool, "angefragt", "M-CD 5678", time.Now().Add(-2*time.Hour), 5000)

	_, orders := getWorkshopOrderList(t, handler, nil)

	newerIndex, ok := findOrder(orders, newer)
	if !ok {
		t.Fatalf("newer order %s missing from list: %+v", newer, orders)
	}
	olderIndex, ok := findOrder(orders, older)
	if !ok {
		t.Fatalf("older order %s missing from list: %+v", older, orders)
	}
	if newerIndex >= olderIndex {
		t.Errorf("orders not newest-first: newer at %d, older at %d", newerIndex, olderIndex)
	}

	got := orders[newerIndex]
	if got.Status != "bestaetigt" {
		t.Errorf("status = %q, want %q", got.Status, "bestaetigt")
	}
	if got.LicensePlate != "B-AB 1234" {
		t.Errorf("license_plate = %q, want %q", got.LicensePlate, "B-AB 1234")
	}
	if got.CustomerName != "Testkunde" {
		t.Errorf("customer_name = %q, want %q", got.CustomerName, "Testkunde")
	}
	if got.TotalCents != 19000 {
		t.Errorf("total_cents = %d, want %d", got.TotalCents, 19000)
	}
	if got.PreferredDate == nil || *got.PreferredDate != "2026-01-02" {
		t.Errorf("preferred_date = %v, want 2026-01-02", got.PreferredDate)
	}
}

func TestListWorkshopOrdersFiltersByStatus(t *testing.T) {
	pool := newWorkshopListPool(t)
	s := NewServer(pool, &config.Config{})
	handler := workshopListMux(s)

	confirmed := seedWorkshopOrder(t, pool, "bestaetigt", "B-EF 1111", time.Now(), 1000)
	requested := seedWorkshopOrder(t, pool, "angefragt", "B-GH 2222", time.Now(), 2000)

	_, orders := getWorkshopOrderList(t, handler, url.Values{"status": {"bestaetigt"}})

	if _, ok := findOrder(orders, confirmed); !ok {
		t.Errorf("confirmed order %s missing from filtered list: %+v", confirmed, orders)
	}
	if _, ok := findOrder(orders, requested); ok {
		t.Errorf("requested order %s should be filtered out: %+v", requested, orders)
	}
	for _, o := range orders {
		if o.Status != "bestaetigt" {
			t.Errorf("filtered list contains status %q", o.Status)
		}
	}
}

func TestListWorkshopOrdersSearchesPartialPlateCaseInsensitive(t *testing.T) {
	pool := newWorkshopListPool(t)
	s := NewServer(pool, &config.Config{})
	handler := workshopListMux(s)

	target := seedWorkshopOrder(t, pool, "angefragt", "B-AB 1234", time.Now(), 1000)
	other := seedWorkshopOrder(t, pool, "angefragt", "M-XY 9999", time.Now(), 2000)

	_, orders := getWorkshopOrderList(t, handler, url.Values{"license_plate": {"ab 12"}})
	if _, ok := findOrder(orders, target); !ok {
		t.Errorf("partial lowercase plate search missed order %s: %+v", target, orders)
	}
	if _, ok := findOrder(orders, other); ok {
		t.Errorf("partial plate search returned non-matching order %s: %+v", other, orders)
	}
}

func TestListWorkshopOrdersRejectsUnknownStatus(t *testing.T) {
	pool := newWorkshopListPool(t)
	s := NewServer(pool, &config.Config{})
	handler := workshopListMux(s)

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders?status=bogus", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown status filter status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if envelope.Error.Code != CodeValidationError {
		t.Errorf("error code = %q, want %q", envelope.Error.Code, CodeValidationError)
	}
}

func TestListWorkshopOrdersRequiresAuthentication(t *testing.T) {
	pool := newWorkshopListPool(t)
	router := NewRouter(NewServer(pool, &config.Config{}))

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	// The employee middleware (another ticket) rejects an anonymous caller. The
	// only thing this ticket can assert is that the route is not served without
	// credentials; it never answers 2xx.
	if rec.Code >= 200 && rec.Code < 300 {
		t.Fatalf("unauthenticated GET /api/workshop/orders returned %d, want a rejection", rec.Code)
	}
}

func TestGetWorkshopOrderReturnsDetail(t *testing.T) {
	pool := newWorkshopListPool(t)
	s := NewServer(pool, &config.Config{})
	handler := workshopListMux(s)

	orderNumber := seedWorkshopOrder(t, pool, "in_arbeit", "B-IJ 3333", time.Now(), 27000)

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders/"+orderNumber, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET detail status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var detail store.WorkshopOrderDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("detail body is not JSON: %v", err)
	}
	if detail.OrderNumber != orderNumber {
		t.Errorf("order_number = %q, want %q", detail.OrderNumber, orderNumber)
	}
	if detail.Status != "in_arbeit" {
		t.Errorf("status = %q, want %q", detail.Status, "in_arbeit")
	}
	if detail.Vehicle.LicensePlate != "B-IJ 3333" || detail.Vehicle.Make != "VW" || detail.Vehicle.Model != "Golf" {
		t.Errorf("vehicle = %+v", detail.Vehicle)
	}
	if detail.Customer.Name != "Testkunde" {
		t.Errorf("customer name = %q", detail.Customer.Name)
	}
	if detail.ProblemDescription != "Bremsen quietschen" {
		t.Errorf("problem_description = %q", detail.ProblemDescription)
	}
	if len(detail.Items) != 1 {
		t.Fatalf("items = %+v, want 1 entry", detail.Items)
	}
	if detail.Items[0].Kind != "labor" || detail.Items[0].TotalCents != 27000 {
		t.Errorf("item = %+v", detail.Items[0])
	}
	if len(detail.StatusHistory) != 1 || detail.StatusHistory[0].Status != "in_arbeit" {
		t.Errorf("status_history = %+v", detail.StatusHistory)
	}
}

func TestGetWorkshopOrderUnknownReturnsNotFound(t *testing.T) {
	pool := newWorkshopListPool(t)
	s := NewServer(pool, &config.Config{})
	handler := workshopListMux(s)

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders/DOES-NOT-EXIST", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown order status = %d, want 404 (body: %s)", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("error body is not JSON: %v", err)
	}
	if envelope.Error.Code != CodeNotFound {
		t.Errorf("error code = %q, want %q", envelope.Error.Code, CodeNotFound)
	}
}
