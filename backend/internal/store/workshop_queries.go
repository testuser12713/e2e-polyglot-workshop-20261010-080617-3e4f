package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrWorkshopOrderNotFound is returned by GetWorkshopOrderDetail when no order
// carries the requested order number. Handlers map it to a 404 not_found.
var ErrWorkshopOrderNotFound = errors.New("store: workshop order not found")

// WorkshopOrderSummary is one row of the workshop order list. The JSON shape is
// the one the API contract declares for GET /api/workshop/orders.
type WorkshopOrderSummary struct {
	OrderNumber   string  `json:"order_number"`
	Status        string  `json:"status"`
	LicensePlate  string  `json:"license_plate"`
	CustomerName  string  `json:"customer_name"`
	PreferredDate *string `json:"preferred_date"`
	TotalCents    int64   `json:"total_cents"`
}

// WorkshopOrderCustomer is the customer block of an order detail.
type WorkshopOrderCustomer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// WorkshopOrderVehicle is the vehicle block of an order detail.
type WorkshopOrderVehicle struct {
	LicensePlate string `json:"license_plate"`
	Make         string `json:"make"`
	Model        string `json:"model"`
	MileageKM    int    `json:"mileage_km"`
}

// WorkshopOrderItem is one captured labor/part position on an order.
type WorkshopOrderItem struct {
	Kind           string `json:"kind"`
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	TotalCents     int64  `json:"total_cents"`
}

// WorkshopOrderStatusChange is one entry of an order's status history.
type WorkshopOrderStatusChange struct {
	Status    string    `json:"status"`
	ChangedAt time.Time `json:"changed_at"`
}

// WorkshopOrderDetail is the full order shape returned by
// GET /api/workshop/orders/{order_number}.
type WorkshopOrderDetail struct {
	OrderNumber        string                      `json:"order_number"`
	Status             string                      `json:"status"`
	Customer           WorkshopOrderCustomer       `json:"customer"`
	Vehicle            WorkshopOrderVehicle        `json:"vehicle"`
	PreferredDate      *string                     `json:"preferred_date"`
	ProblemDescription string                      `json:"problem_description"`
	Items              []WorkshopOrderItem         `json:"items"`
	StatusHistory      []WorkshopOrderStatusChange `json:"status_history"`
}

// ListWorkshopOrders returns the order summaries ordered by creation date
// descending. An empty status matches every status; an empty licensePlate
// matches every vehicle. The license plate filter is a case-insensitive partial
// match. Every query is parameterised (AC-25).
func ListWorkshopOrders(ctx context.Context, pool *pgxpool.Pool, status, licensePlate string) ([]WorkshopOrderSummary, error) {
	const query = `
		SELECT o.order_number,
		       o.status,
		       v.license_plate,
		       c.name,
		       to_char(o.preferred_date, 'YYYY-MM-DD'),
		       COALESCE((
		           SELECT SUM(oi.total_cents)
		           FROM order_items oi
		           WHERE oi.order_id = o.id
		       ), 0)
		FROM orders o
		JOIN customers c ON c.id = o.customer_id
		JOIN vehicles v ON v.id = o.vehicle_id
		WHERE ($1 = '' OR o.status = $1)
		  AND ($2 = '' OR v.license_plate ILIKE '%' || $2 || '%')
		ORDER BY o.created_at DESC, o.id DESC`

	rows, err := pool.Query(ctx, query, status, licensePlate)
	if err != nil {
		return nil, fmt.Errorf("store: list workshop orders: %w", err)
	}
	defer rows.Close()

	orders := make([]WorkshopOrderSummary, 0)
	for rows.Next() {
		var summary WorkshopOrderSummary
		if err := rows.Scan(
			&summary.OrderNumber,
			&summary.Status,
			&summary.LicensePlate,
			&summary.CustomerName,
			&summary.PreferredDate,
			&summary.TotalCents,
		); err != nil {
			return nil, fmt.Errorf("store: scan workshop order: %w", err)
		}
		orders = append(orders, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate workshop orders: %w", err)
	}
	return orders, nil
}

// GetWorkshopOrderDetail returns one order with its customer, vehicle, items and
// status history. It returns ErrWorkshopOrderNotFound when no order matches.
func GetWorkshopOrderDetail(ctx context.Context, pool *pgxpool.Pool, orderNumber string) (*WorkshopOrderDetail, error) {
	const orderQuery = `
		SELECT o.id,
		       o.order_number,
		       o.status,
		       c.name,
		       c.email,
		       c.phone,
		       v.license_plate,
		       v.make,
		       v.model,
		       v.mileage_km,
		       to_char(o.preferred_date, 'YYYY-MM-DD'),
		       o.problem_description
		FROM orders o
		JOIN customers c ON c.id = o.customer_id
		JOIN vehicles v ON v.id = o.vehicle_id
		WHERE o.order_number = $1`

	var (
		orderID int64
		detail  WorkshopOrderDetail
	)
	err := pool.QueryRow(ctx, orderQuery, orderNumber).Scan(
		&orderID,
		&detail.OrderNumber,
		&detail.Status,
		&detail.Customer.Name,
		&detail.Customer.Email,
		&detail.Customer.Phone,
		&detail.Vehicle.LicensePlate,
		&detail.Vehicle.Make,
		&detail.Vehicle.Model,
		&detail.Vehicle.MileageKM,
		&detail.PreferredDate,
		&detail.ProblemDescription,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWorkshopOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get workshop order: %w", err)
	}

	items, err := listOrderItems(ctx, pool, orderID)
	if err != nil {
		return nil, err
	}
	history, err := listOrderStatusHistory(ctx, pool, orderID)
	if err != nil {
		return nil, err
	}
	detail.Items = items
	detail.StatusHistory = history
	return &detail, nil
}

func listOrderItems(ctx context.Context, pool *pgxpool.Pool, orderID int64) ([]WorkshopOrderItem, error) {
	const query = `
		SELECT kind, description, quantity, unit_price_cents, total_cents
		FROM order_items
		WHERE order_id = $1
		ORDER BY id`

	rows, err := pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("store: list order items: %w", err)
	}
	defer rows.Close()

	items := make([]WorkshopOrderItem, 0)
	for rows.Next() {
		var item WorkshopOrderItem
		if err := rows.Scan(
			&item.Kind,
			&item.Description,
			&item.Quantity,
			&item.UnitPriceCents,
			&item.TotalCents,
		); err != nil {
			return nil, fmt.Errorf("store: scan order item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate order items: %w", err)
	}
	return items, nil
}

func listOrderStatusHistory(ctx context.Context, pool *pgxpool.Pool, orderID int64) ([]WorkshopOrderStatusChange, error) {
	const query = `
		SELECT status, changed_at
		FROM status_history
		WHERE order_id = $1
		ORDER BY changed_at, id`

	rows, err := pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("store: list order status history: %w", err)
	}
	defer rows.Close()

	history := make([]WorkshopOrderStatusChange, 0)
	for rows.Next() {
		var change WorkshopOrderStatusChange
		if err := rows.Scan(&change.Status, &change.ChangedAt); err != nil {
			return nil, fmt.Errorf("store: scan order status change: %w", err)
		}
		history = append(history, change)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate order status history: %w", err)
	}
	return history, nil
}
