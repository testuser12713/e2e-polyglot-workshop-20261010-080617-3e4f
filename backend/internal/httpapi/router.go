package httpapi

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/config"
)

const defaultCORSOrigin = config.DefaultCORSOrigin

// Server carries the dependencies every handler shares.
type Server struct {
	Pool   *pgxpool.Pool
	Config *config.Config
}

// NewServer builds a Server. Config may be nil in tests that do not exercise
// configuration-dependent behaviour.
func NewServer(pool *pgxpool.Pool, cfg *config.Config) *Server {
	SetSessionPool(pool)
	return &Server{Pool: pool, Config: cfg}
}

// NewRouter wires every route of the API contract exactly once. Public routes
// are plain handlers; every /api/workshop/* route is wrapped in RequireEmployee.
func NewRouter(s *Server) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.Health)

	mux.HandleFunc("POST /api/customers", s.CreateCustomer)
	mux.HandleFunc("POST /api/vehicles", s.CreateVehicle)
	mux.HandleFunc("POST /api/orders", s.CreateOrder)

	mux.HandleFunc("GET /api/orders/status", s.GetOrderStatus)
	mux.HandleFunc("GET /api/orders/{order_number}/invoice", s.GetOrderInvoice)

	mux.HandleFunc("POST /api/auth/login", s.Login)

	mux.Handle("GET /api/workshop/orders", RequireEmployee(http.HandlerFunc(s.ListWorkshopOrders)))
	mux.Handle("GET /api/workshop/orders/{order_number}", RequireEmployee(http.HandlerFunc(s.GetWorkshopOrder)))
	mux.Handle("POST /api/workshop/orders/{order_number}/confirm", RequireEmployee(http.HandlerFunc(s.ConfirmOrder)))
	mux.Handle("POST /api/workshop/orders/{order_number}/items", RequireEmployee(http.HandlerFunc(s.AddOrderItems)))
	mux.Handle("POST /api/workshop/orders/{order_number}/status", RequireEmployee(http.HandlerFunc(s.SetOrderStatus)))
	mux.Handle("GET /api/workshop/dashboard", RequireEmployee(http.HandlerFunc(s.GetDashboard)))

	var handler http.Handler = mux
	handler = Recover(handler)
	handler = Logging(handler)
	handler = CORS(s.corsOrigin(), handler)
	return handler
}

func (s *Server) corsOrigin() string {
	if s.Config == nil {
		return ""
	}
	return s.Config.CORSOrigin
}
