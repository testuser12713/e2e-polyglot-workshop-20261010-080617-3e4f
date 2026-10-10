package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/store"
)

// sessionPool is the database pool RequireEmployee uses to resolve a bearer
// token. It is set once when the server is built, which keeps RequireEmployee's
// agreed signature (no Server receiver) while still querying real sessions.
var sessionPool *pgxpool.Pool

// SetSessionPool connects the session lookup used by RequireEmployee. It is
// called by NewServer.
func SetSessionPool(pool *pgxpool.Pool) {
	sessionPool = pool
}

// RequireEmployee guards every /api/workshop/* route. A missing token, an
// unknown token and an expired token all answer 401 in the shared error
// envelope; only a live session reaches the wrapped handler (AC-13).
func RequireEmployee(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "missing bearer token")
			return
		}
		if sessionPool == nil {
			WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "invalid bearer token")
			return
		}
		if _, err := store.EmployeeBySessionToken(r.Context(), sessionPool, hashSessionToken(token)); err != nil {
			if errors.Is(err, store.ErrEmployeeNotFound) {
				WriteError(w, http.StatusUnauthorized, CodeUnauthorized, "invalid or expired bearer token")
				return
			}
			WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// bearerToken extracts the token from an "Authorization: Bearer <token>" header,
// returning "" when the header is absent or malformed.
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

// hashSessionToken is the hex SHA-256 of a raw session token. Only this hash is
// ever stored, so a leaked database still cannot be replayed against the API.
func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
