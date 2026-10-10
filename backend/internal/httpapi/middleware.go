package httpapi

import (
	"log"
	"net/http"
	"strings"
	"time"
)

// CORS answers browser requests only for the configured web origin. It never
// combines a wildcard origin with credentials.
//
// An unresolved placeholder (the web service is not declared on this branch yet)
// or an empty value falls back to the development default.
func CORS(origin string, next http.Handler) http.Handler {
	allowed := origin
	if allowed == "" || strings.Contains(allowed, "${") {
		allowed = defaultCORSOrigin
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requestOrigin := r.Header.Get("Origin"); requestOrigin != "" && requestOrigin == allowed {
			header := w.Header()
			header.Set("Access-Control-Allow-Origin", allowed)
			header.Set("Access-Control-Allow-Credentials", "true")
			header.Set("Vary", "Origin")
			header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			header.Set("Access-Control-Max-Age", "600")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Recover turns a panicking handler into a clean 500 in the shared envelope so
// the process keeps serving and the error still carries CORS headers (this
// middleware sits inside CORS).
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("panic handling %s %s", r.Method, r.URL.Path)
				WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Logging logs one line per request. It deliberately logs only the method and
// path — never the query string or body, which may contain a license plate,
// problem description or credentials (AC-30).
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, recorder.status, time.Since(start))
	})
}
