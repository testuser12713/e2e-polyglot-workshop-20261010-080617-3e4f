package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"workshop/internal/auth"
	"workshop/internal/store"
)

const (
	// maxLoginFailures is how many failed attempts a single client may make
	// before the endpoint answers 429 (AC-23).
	maxLoginFailures = 5
	// loginFailureWindow is the sliding window the failures are counted in.
	loginFailureWindow = 15 * time.Minute
	// sessionTTL is how long a login token stays valid.
	sessionTTL = 12 * time.Hour
)

// loginLimiter is the process-wide rate limiter for POST /api/auth/login. It is
// keyed by client address so one attacker cannot lock out every employee.
var loginLimiter = auth.NewRateLimiter(maxLoginFailures, loginFailureWindow)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type employeePayload struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type loginResponse struct {
	Token    string          `json:"token"`
	Employee employeePayload `json:"employee"`
}

// Login handles POST /api/auth/login. It verifies the password against the
// stored bcrypt hash and, on success, returns a session token plus the public
// employee fields. The password and its hash never appear in the response, and
// nothing sensitive is logged (AC-24, AC-30).
func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	client := clientAddress(r)

	if !loginLimiter.Allow(client) {
		WriteError(w, http.StatusTooManyRequests, CodeRateLimited, "too many failed login attempts, try again later")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "invalid request body")
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		WriteError(w, http.StatusBadRequest, CodeValidationError, "email and password are required")
		return
	}

	if s.Pool == nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}

	employee, err := store.FindEmployeeByEmail(r.Context(), s.Pool, req.Email)
	if err != nil {
		if errors.Is(err, store.ErrEmployeeNotFound) {
			loginLimiter.RecordFailure(client)
			WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "invalid email or password")
			return
		}
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}

	if !auth.VerifyPassword(employee.PasswordHash, req.Password) {
		loginLimiter.RecordFailure(client)
		WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "invalid email or password")
		return
	}

	token, err := newSessionToken()
	if err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}
	if err := store.CreateEmployeeSession(r.Context(), s.Pool, employee.ID, hashSessionToken(token), time.Now().Add(sessionTTL)); err != nil {
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}

	loginLimiter.Reset(client)
	WriteJSON(w, http.StatusOK, loginResponse{
		Token: token,
		Employee: employeePayload{
			ID:    employee.ID,
			Name:  employee.Name,
			Email: employee.Email,
		},
	})
}

// newSessionToken returns a 256-bit random token rendered as hex. It is the raw
// value handed to the client; only its hash is stored server-side.
func newSessionToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// clientAddress is the rate-limit key: the client IP without its ephemeral port.
func clientAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
