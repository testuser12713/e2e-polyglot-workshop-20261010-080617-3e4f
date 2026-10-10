package httpapi

import "net/http"

// Login handles POST /api/auth/login. Implemented by the employee login ticket;
// this stub answers 501 until then.
func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	NotYetImplemented(w, r)
}
