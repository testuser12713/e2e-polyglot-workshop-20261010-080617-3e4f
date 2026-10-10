package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/auth"
	"workshop/internal/config"
)

// testRouterWithPool builds the real router over the PostgreSQL test pool so
// login, session storage and the middleware all run against the database.
func testRouterWithPool(t *testing.T) (*pgxpool.Pool, http.Handler) {
	t.Helper()
	pool := newTestPool(t)
	cfg := &config.Config{CORSOrigin: config.DefaultCORSOrigin}
	return pool, NewRouter(NewServer(pool, cfg))
}

// seedEmployee inserts an employee with a real bcrypt hash and removes it again
// (sessions cascade) when the test ends. It touches only this ticket's tables.
func seedEmployee(t *testing.T, pool *pgxpool.Pool, email, password string) {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	_, err = pool.Exec(context.Background(),
		`INSERT INTO employees (name, email, password_hash)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash`,
		"Test Employee", email, hash)
	if err != nil {
		t.Fatalf("seed employee: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM employees WHERE email = $1`, email)
	})
}

func uniqueEmail() string {
	return fmt.Sprintf("emp-%d@example.com", time.Now().UnixNano())
}

func postLogin(handler http.Handler, remoteAddr, email, password string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("response body is not the error envelope: %v (body: %s)", err, rec.Body.String())
	}
	return envelope.Error.Code
}

func TestLoginReturnsTokenForConfiguredEmployee(t *testing.T) {
	pool, router := testRouterWithPool(t)
	email := uniqueEmail()
	const password = "correct-horse-battery"
	seedEmployee(t, pool, email, password)

	rec := postLogin(router, "198.51.100.10:5555", email, password)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Token    string `json:"token"`
		Employee struct {
			ID    int64  `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"employee"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("login body is not JSON: %v", err)
	}
	if resp.Token == "" {
		t.Error("login returned an empty token")
	}
	if resp.Employee.ID == 0 {
		t.Error("login returned no employee id")
	}
	if resp.Employee.Email != email {
		t.Errorf("employee email = %q, want %q", resp.Employee.Email, email)
	}

	body := rec.Body.String()
	if strings.Contains(body, password) {
		t.Error("login response leaked the plaintext password")
	}
	if strings.Contains(body, "$2a$") || strings.Contains(body, "$2b$") || strings.Contains(body, "$2y$") {
		t.Error("login response leaked a bcrypt password hash")
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	pool, router := testRouterWithPool(t)
	email := uniqueEmail()
	seedEmployee(t, pool, email, "right-password")

	rec := postLogin(router, "198.51.100.11:5555", email, "wrong-password")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := errorCode(t, rec); code != CodeUnauthorized {
		t.Errorf("error code = %q, want %q", code, CodeUnauthorized)
	}
}

func TestLoginRejectsUnknownEmail(t *testing.T) {
	_, router := testRouterWithPool(t)

	rec := postLogin(router, "198.51.100.12:5555", uniqueEmail(), "whatever")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body: %s)", rec.Code, rec.Body.String())
	}
}

func TestLoginRateLimitsRepeatedFailures(t *testing.T) {
	pool, router := testRouterWithPool(t)
	email := uniqueEmail()
	seedEmployee(t, pool, email, "right-password")

	const addr = "198.51.100.77:4000"
	loginLimiter.Reset("198.51.100.77")
	t.Cleanup(func() { loginLimiter.Reset("198.51.100.77") })

	var last *httptest.ResponseRecorder
	for attempt := 0; attempt <= maxLoginFailures; attempt++ {
		last = postLogin(router, addr, email, "wrong-password")
		if attempt < maxLoginFailures && last.Code != http.StatusUnauthorized {
			t.Fatalf("failure %d status = %d, want 401", attempt+1, last.Code)
		}
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("status after repeated failures = %d, want 429 (body: %s)", last.Code, last.Body.String())
	}
	if code := errorCode(t, last); code != CodeRateLimited {
		t.Errorf("error code = %q, want %q", code, CodeRateLimited)
	}
}

func TestWorkshopOrderWithoutTokenIsUnauthorized(t *testing.T) {
	_, router := testRouterWithPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body: %s)", rec.Code, rec.Body.String())
	}
	if code := errorCode(t, rec); code != CodeUnauthorized {
		t.Errorf("error code = %q, want %q", code, CodeUnauthorized)
	}
}

func TestWorkshopOrderRejectsUnknownToken(t *testing.T) {
	_, router := testRouterWithPool(t)

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestWorkshopOrderAcceptsValidToken proves the middleware passes a freshly
// issued token through to the wrapped handler. It asserts only that the guard
// opened (no 401, no 5xx) — the handler body belongs to the workshop list
// ticket and may still be a 501 placeholder here.
func TestWorkshopOrderAcceptsValidToken(t *testing.T) {
	pool, router := testRouterWithPool(t)
	email := uniqueEmail()
	seedEmployee(t, pool, email, "valid-password")

	login := postLogin(router, "198.51.100.20:1", email, "valid-password")
	if login.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (body: %s)", login.Code, login.Body.String())
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &resp); err != nil {
		t.Fatalf("login body is not JSON: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/workshop/orders", nil)
	req.Header.Set("Authorization", "Bearer "+resp.Token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("valid token was rejected with 401 (body: %s)", rec.Body.String())
	}
	if rec.Code == http.StatusInternalServerError {
		t.Fatalf("workshop handler crashed with 500 (body: %s)", rec.Body.String())
	}
}
