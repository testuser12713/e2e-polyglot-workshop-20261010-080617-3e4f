package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/store"
)

// dashboardFixture holds the ids of the rows a dashboard test created so the
// cleanup can remove exactly those rows and nothing another ticket owns.
type dashboardFixture struct {
	orderNumbers []string
	vehicleID    int64
	customerID   int64
}

func uniqueSuffix(t *testing.T) string {
	t.Helper()
	replacer := strings.NewReplacer("/", "-", " ", "-", "_", "-")
	return fmt.Sprintf("%s-%d", replacer.Replace(t.Name()), time.Now().UnixNano())
}

// seedDashboard writes a known set of orders, status-history entries and
// invoices: two more open orders, two orders finished today and one invoice for
// the current month plus one for the previous month.
func seedDashboard(t *testing.T, pool *pgxpool.Pool, suffix string) dashboardFixture {
	t.Helper()
	ctx := context.Background()
	fixture := dashboardFixture{}

	if err := pool.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		"Dashboard Test", "dash-"+suffix+"@example.test", "000000",
	).Scan(&fixture.customerID); err != nil {
		t.Fatalf("insert customer: %v", err)
	}

	if err := pool.QueryRow(ctx,
		`INSERT INTO vehicles (license_plate, make, model, mileage_km) VALUES ($1, $2, $3, $4) RETURNING id`,
		"DASH-"+suffix, "Test", "Model", 1000,
	).Scan(&fixture.vehicleID); err != nil {
		t.Fatalf("insert vehicle: %v", err)
	}

	insertOrder := func(label, status string) int64 {
		number := "DASH-" + suffix + "-" + label
		var id int64
		if err := pool.QueryRow(ctx,
			`INSERT INTO orders (order_number, customer_id, vehicle_id, status)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			number, fixture.customerID, fixture.vehicleID, status,
		).Scan(&id); err != nil {
			t.Fatalf("insert order %s: %v", label, err)
		}
		fixture.orderNumbers = append(fixture.orderNumbers, number)
		return id
	}

	insertHistory := func(orderID int64, status string, changedAt time.Time) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO status_history (order_id, status, changed_at) VALUES ($1, $2, $3)`,
			orderID, status, changedAt,
		); err != nil {
			t.Fatalf("insert status history: %v", err)
		}
	}

	insertInvoice := func(orderID int64, number string, grossCents int64, createdAt time.Time) {
		if _, err := pool.Exec(ctx,
			`INSERT INTO invoices (order_id, order_number, gross_cents, created_at)
			 VALUES ($1, $2, $3, $4)`,
			orderID, number, grossCents, createdAt,
		); err != nil {
			t.Fatalf("insert invoice: %v", err)
		}
	}

	now := time.Now()
	firstOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	previousMonth := firstOfMonth.AddDate(0, -1, 0)

	// Open orders: in_arbeit and fertig count, abgeholt does not.
	openOrder := insertOrder("open", "in_arbeit")
	doneOrder := insertOrder("done", "fertig")
	pickedUpYesterday := insertOrder("picked-yesterday", "abgeholt")
	pickedUpToday := insertOrder("picked-today", "abgeholt")
	_ = openOrder

	// Finished today: doneOrder and pickedUpToday. pickedUpYesterday was
	// finished yesterday and must not be counted.
	insertHistory(doneOrder, "fertig", now)
	insertHistory(pickedUpToday, "fertig", now)
	insertHistory(pickedUpToday, "abgeholt", now)
	insertHistory(pickedUpYesterday, "fertig", now.Add(-24*time.Hour))

	// Revenue: the current month's invoice counts, the previous month's does not.
	insertInvoice(doneOrder, "DASH-"+suffix+"-inv-current", 11900, now)
	insertInvoice(pickedUpYesterday, "DASH-"+suffix+"-inv-previous", 5000, previousMonth)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM orders WHERE order_number = ANY($1)`, fixture.orderNumbers)
		_, _ = pool.Exec(ctx, `DELETE FROM vehicles WHERE id = $1`, fixture.vehicleID)
		_, _ = pool.Exec(ctx, `DELETE FROM customers WHERE id = $1`, fixture.customerID)
	})

	return fixture
}

func TestGetDashboardMetricsCounts(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	before, err := store.GetDashboardMetrics(ctx, pool)
	if err != nil {
		t.Fatalf("baseline metrics: %v", err)
	}

	seedDashboard(t, pool, uniqueSuffix(t))

	after, err := store.GetDashboardMetrics(ctx, pool)
	if err != nil {
		t.Fatalf("metrics after seeding: %v", err)
	}

	if got, want := after.OpenOrders-before.OpenOrders, int64(2); got != want {
		t.Errorf("open_orders delta = %d, want %d", got, want)
	}
	if got, want := after.CompletedToday-before.CompletedToday, int64(2); got != want {
		t.Errorf("completed_today delta = %d, want %d", got, want)
	}
	if got, want := after.RevenueMonthCents-before.RevenueMonthCents, int64(11900); got != want {
		t.Errorf("revenue_month_cents delta = %d, want %d", got, want)
	}
}

func TestGetDashboardHandlerResponds(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	srv := NewServer(pool, nil)

	before, err := store.GetDashboardMetrics(ctx, pool)
	if err != nil {
		t.Fatalf("baseline metrics: %v", err)
	}

	seedDashboard(t, pool, uniqueSuffix(t))

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/dashboard", nil)
	rec := httptest.NewRecorder()
	srv.GetDashboard(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/workshop/dashboard status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var body map[string]int64
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("dashboard body is not JSON: %v", err)
	}
	for _, key := range []string{"open_orders", "completed_today", "revenue_month_cents"} {
		if _, ok := body[key]; !ok {
			t.Errorf("dashboard body is missing %q (body: %s)", key, rec.Body.String())
		}
	}

	if got, want := body["open_orders"]-before.OpenOrders, int64(2); got != want {
		t.Errorf("open_orders delta = %d, want %d", got, want)
	}
	if got, want := body["completed_today"]-before.CompletedToday, int64(2); got != want {
		t.Errorf("completed_today delta = %d, want %d", got, want)
	}
	if got, want := body["revenue_month_cents"]-before.RevenueMonthCents, int64(11900); got != want {
		t.Errorf("revenue_month_cents delta = %d, want %d", got, want)
	}
}
