package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/domain"
)

// maxOrderNumberAttempts bounds the retry loop that allocates a unique order
// number, in case of the (astronomically unlikely) collision.
const maxOrderNumberAttempts = 5

// Order is a workshop order as returned by the public intake API.
type Order struct {
	ID                 int64      `json:"-"`
	OrderNumber        string     `json:"order_number"`
	Status             string     `json:"status"`
	PreferredDate      *time.Time `json:"-"`
	ProblemDescription string     `json:"-"`
	CreatedAt          time.Time  `json:"created_at"`
}

// CustomerInput is the customer description carried by an order request.
type CustomerInput struct {
	Name  string
	Email string
	Phone string
}

// VehicleInput is the vehicle description carried by an order request.
type VehicleInput struct {
	LicensePlate string
	Make         string
	Model        string
	MileageKM    int
}

// OrderRequest is the public intake payload that creates (or reuses) a customer
// and a vehicle and then opens the order in status "angefragt".
type OrderRequest struct {
	Customer           CustomerInput
	Vehicle            VehicleInput
	PreferredDate      *time.Time
	ProblemDescription string
}

// CreateOrder creates or reuses the customer and vehicle, then inserts the
// order in status "angefragt" together with its initial status-history entry.
// It retries when the generated order number collides.
func CreateOrder(ctx context.Context, pool *pgxpool.Pool, req OrderRequest) (Order, error) {
	var lastErr error
	for attempt := 0; attempt < maxOrderNumberAttempts; attempt++ {
		order, err := createOrderOnce(ctx, pool, req)
		if err == nil {
			return order, nil
		}
		if !isUniqueViolation(err) {
			return Order{}, err
		}
		lastErr = err
	}
	return Order{}, fmt.Errorf("store: could not allocate a unique order number: %w", lastErr)
}

func createOrderOnce(ctx context.Context, pool *pgxpool.Pool, req OrderRequest) (Order, error) {
	customer, err := findOrCreateCustomer(ctx, pool, req.Customer)
	if err != nil {
		return Order{}, err
	}
	vehicle, err := findOrCreateVehicle(ctx, pool, req.Vehicle)
	if err != nil {
		return Order{}, err
	}

	orderNumber, err := generateOrderNumber()
	if err != nil {
		return Order{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	order := Order{
		OrderNumber:        orderNumber,
		Status:             string(domain.StatusRequested),
		PreferredDate:      req.PreferredDate,
		ProblemDescription: req.ProblemDescription,
	}
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (order_number, customer_id, vehicle_id, preferred_date, problem_description, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, created_at`,
		order.OrderNumber, customer.ID, vehicle.ID, req.PreferredDate, req.ProblemDescription, order.Status,
	).Scan(&order.ID, &order.CreatedAt)
	if err != nil {
		return Order{}, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO status_history (order_id, status, previous_status) VALUES ($1, $2, NULL)`,
		order.ID, order.Status,
	); err != nil {
		return Order{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}
	return order, nil
}

// findOrCreateCustomer reuses the customer with the same e-mail or creates one.
func findOrCreateCustomer(ctx context.Context, q querier, in CustomerInput) (Customer, error) {
	if existing, found, err := findCustomerByEmail(ctx, q, in.Email); err != nil {
		return Customer{}, err
	} else if found {
		return existing, nil
	}
	return createCustomer(ctx, q, in.Name, in.Email, in.Phone)
}

// findOrCreateVehicle reuses the vehicle with the same license plate or creates
// one. It tolerates losing a race to a concurrent insert with the same plate.
func findOrCreateVehicle(ctx context.Context, q querier, in VehicleInput) (Vehicle, error) {
	if existing, found, err := findVehicleByPlate(ctx, q, in.LicensePlate); err != nil {
		return Vehicle{}, err
	} else if found {
		return existing, nil
	}

	vehicle, err := createVehicle(ctx, q, in.LicensePlate, in.Make, in.Model, in.MileageKM)
	if errors.Is(err, ErrDuplicatePlate) {
		if existing, found, lookupErr := findVehicleByPlate(ctx, q, in.LicensePlate); lookupErr != nil {
			return Vehicle{}, lookupErr
		} else if found {
			return existing, nil
		}
	}
	return vehicle, err
}

// generateOrderNumber returns a unique-looking order number of the form
// "AU-YYYYMMDD-XXXXXXXX" where the suffix is 32 bits of randomness.
func generateOrderNumber() (string, error) {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	suffix := strings.ToUpper(hex.EncodeToString(buf))
	return fmt.Sprintf("AU-%s-%s", time.Now().UTC().Format("20060102"), suffix), nil
}
