package httpapi

import (
	"log"
	"net/http"

	"workshop/internal/store"
)

// dashboardResponse is the JSON body of GET /api/workshop/dashboard.
type dashboardResponse struct {
	OpenOrders        int64 `json:"open_orders"`
	CompletedToday    int64 `json:"completed_today"`
	RevenueMonthCents int64 `json:"revenue_month_cents"`
}

// GetDashboard handles GET /api/workshop/dashboard. It returns the number of
// open orders, the orders finished today and the current month's revenue in
// integer cents, all read in one query round.
func (s *Server) GetDashboard(w http.ResponseWriter, r *http.Request) {
	metrics, err := store.GetDashboardMetrics(r.Context(), s.Pool)
	if err != nil {
		log.Printf("GET /api/workshop/dashboard: %v", err)
		WriteError(w, http.StatusInternalServerError, CodeInternalError, "internal server error")
		return
	}
	WriteJSON(w, http.StatusOK, dashboardResponse{
		OpenOrders:        metrics.OpenOrders,
		CompletedToday:    metrics.CompletedToday,
		RevenueMonthCents: metrics.RevenueMonthCents,
	})
}
