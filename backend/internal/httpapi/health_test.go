package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/config"
	"workshop/internal/store"
)

// newTestPool connects to the real PostgreSQL from DATABASE_URL and applies the
// migrations. There is no SQLite anywhere (AC-21). When DATABASE_URL is unset
// the test is skipped; the pipeline always provides it.
func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
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

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	pool := newTestPool(t)
	cfg := &config.Config{CORSOrigin: config.DefaultCORSOrigin}
	return NewRouter(NewServer(pool, cfg))
}

func TestHealthReturnsOK(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/health status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("health body is not JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("health body status = %q, want %q", body["status"], "ok")
	}
}

// TestContractRoutesAreRegistered checks that the router wires the contract's
// paths (a missing route answers 404). It deliberately does not assert what a
// stub body answers — only that the route is reachable.
func TestContractRoutesAreRegistered(t *testing.T) {
	router := newTestRouter(t)

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/health"},
		{http.MethodPost, "/api/customers"},
		{http.MethodPost, "/api/vehicles"},
		{http.MethodPost, "/api/orders"},
		{http.MethodGet, "/api/orders/status"},
		{http.MethodGet, "/api/orders/ABC-123/invoice"},
		{http.MethodPost, "/api/auth/login"},
		{http.MethodGet, "/api/workshop/orders"},
		{http.MethodGet, "/api/workshop/orders/ABC-123"},
		{http.MethodPost, "/api/workshop/orders/ABC-123/confirm"},
		{http.MethodPost, "/api/workshop/orders/ABC-123/items"},
		{http.MethodPost, "/api/workshop/orders/ABC-123/status"},
		{http.MethodGet, "/api/workshop/dashboard"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code == http.StatusNotFound {
			t.Errorf("%s %s is not registered (got 404)", tc.method, tc.path)
		}
	}
}

func TestUnknownRouteIsNotFound(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown route status = %d, want 404", rec.Code)
	}
}
