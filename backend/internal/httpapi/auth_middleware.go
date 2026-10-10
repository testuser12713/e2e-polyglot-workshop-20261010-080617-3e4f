package httpapi

import "net/http"

// RequireEmployee guards every /api/workshop/* route with a Bearer token. The
// employee login ticket fills in the verification; until then this stub answers
// 501 (never a panic, never a 500).
func RequireEmployee(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = next
		NotYetImplemented(w, r)
	})
}
