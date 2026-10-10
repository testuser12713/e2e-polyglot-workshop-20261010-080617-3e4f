package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrOrderNotFound is returned when no order with the given number exists.
var ErrOrderNotFound = errors.New("store: order not found")

// ErrOrderNotChangeable is returned when the order's status no longer allows
// capturing positions (it has been picked up or rejected).
var ErrOrderNotChangeable = errors.New("store: order is not open for positions")

// ItemInput is one position the workshop wants to capture on an order.
type ItemInput struct {
	Kind           string
	Description    string
	Quantity       int
	UnitPriceCents int
}

// OrderItemsLine is one captured position as the API returns it. The total is
// always the integer-cent product of quantity and unit price.
type OrderItemsLine struct {
	Kind           string `json:"kind"`
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
	TotalCents     int    `json:"total_cents"`
}

// OrderItemsCustomer is the customer block of the OrderDetail response.
type OrderItemsCustomer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// OrderItemsVehicle is the vehicle block of the OrderDetail response.
type OrderItemsVehicle struct {
	LicensePlate string `json:"license_plate"`
	Make         string `json:"make"`
	Model        string `json:"model"`
	MileageKM    int    `json:"mileage_km"`
}

// OrderItemsHistory is one status-history entry.
type OrderItemsHistory struct {
	Status    string `json:"status"`
	ChangedAt string `json:"changed_at"`
}

// OrderItemsDetail is the updated OrderDetail returned after positions are
// captured. Its JSON matches the sprint-wide OrderDetail contract exactly.
type OrderItemsDetail struct {
	OrderNumber        string              `json:"order_number"`
	Status             string              `json:"status"`
	Customer           OrderItemsCustomer  `json:"customer"`
	Vehicle            OrderItemsVehicle   `json:"vehicle"`
	PreferredDate      string              `json:"preferred_date"`
	ProblemDescription string              `json:"problem_description"`
	Items              []OrderItemsLine    `json:"items"`
	StatusHistory      []OrderItemsHistory `json:"status_history"`
}

// blockedStatuses lists the statuses that forbid capturing new positions.
var blockedStatuses = map[string]bool{
	"abgeholt":  true,
	"abgelehnt": true,
}

// AddOrderItems appends the positions to the order and returns the order's fresh
// detail. It runs in a single transaction and locks the order row, so concurrent
// captures on the same order cannot interleave their totals. The order must
// exist and must not have been picked up or rejected; otherwise ErrOrderNotFound
// or ErrOrderNotChangeable is returned.
func AddOrderItems(ctx context.Context, pool *pgxpool.Pool, orderNumber string, items []ItemInput) (*OrderItemsDetail, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: begin add order items: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orderID int64
	var status string
	err = tx.QueryRow(ctx,
		`SELECT id, status FROM orders WHERE order_number = $1 FOR UPDATE`,
		orderNumber,
	).Scan(&orderID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: load order %s: %w", orderNumber, err)
	}
	if blockedStatuses[status] {
		return nil, ErrOrderNotChangeable
	}

	for _, item := range items {
		if _, err := tx.Exec(ctx,
			`INSERT INTO order_items
				(order_id, kind, description, quantity, unit_price_cents, total_cents)
			 VALUES ($1, $2, $3, $4::integer, $5::integer, $4::integer * $5::integer)`,
			orderID, item.Kind, item.Description, item.Quantity, item.UnitPriceCents,
		); err != nil {
			return nil, fmt.Errorf("store: insert order item: %w", err)
		}
	}

	detail, err := loadOrderItemsDetail(ctx, tx, orderNumber)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("store: commit add order items: %w", err)
	}
	return detail, nil
}

// loadOrderItemsDetail reads the full OrderDetail for one order number.
func loadOrderItemsDetail(ctx context.Context, q pgx.Tx, orderNumber string) (*OrderItemsDetail, error) {
	detail := &OrderItemsDetail{OrderNumber: orderNumber}
	var preferredDate *time.Time
	var orderID int64
	err := q.QueryRow(ctx,
		`SELECT o.id, o.status,
		        c.name, c.email, c.phone,
		        v.license_plate, v.make, v.model, v.mileage_km,
		        o.preferred_date, o.problem_description
		 FROM orders o
		 JOIN customers c ON c.id = o.customer_id
		 JOIN vehicles  v ON v.id = o.vehicle_id
		 WHERE o.order_number = $1`,
		orderNumber,
	).Scan(
		&orderID, &detail.Status,
		&detail.Customer.Name, &detail.Customer.Email, &detail.Customer.Phone,
		&detail.Vehicle.LicensePlate, &detail.Vehicle.Make, &detail.Vehicle.Model, &detail.Vehicle.MileageKM,
		&preferredDate, &detail.ProblemDescription,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: load order detail %s: %w", orderNumber, err)
	}
	if preferredDate != nil {
		detail.PreferredDate = preferredDate.Format("2006-01-02")
	}

	items, err := loadOrderItems(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	detail.Items = items

	history, err := loadStatusHistory(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	detail.StatusHistory = history
	return detail, nil
}

func loadOrderItems(ctx context.Context, q pgx.Tx, orderID int64) ([]OrderItemsLine, error) {
	rows, err := q.Query(ctx,
		`SELECT kind, description, quantity, unit_price_cents, total_cents
		 FROM order_items
		 WHERE order_id = $1
		 ORDER BY id`,
		orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: load order items: %w", err)
	}
	defer rows.Close()

	items := make([]OrderItemsLine, 0)
	for rows.Next() {
		var line OrderItemsLine
		if err := rows.Scan(&line.Kind, &line.Description, &line.Quantity, &line.UnitPriceCents, &line.TotalCents); err != nil {
			return nil, fmt.Errorf("store: scan order item: %w", err)
		}
		items = append(items, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate order items: %w", err)
	}
	return items, nil
}

func loadStatusHistory(ctx context.Context, q pgx.Tx, orderID int64) ([]OrderItemsHistory, error) {
	rows, err := q.Query(ctx,
		`SELECT status, changed_at
		 FROM status_history
		 WHERE order_id = $1
		 ORDER BY changed_at, id`,
		orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: load status history: %w", err)
	}
	defer rows.Close()

	history := make([]OrderItemsHistory, 0)
	for rows.Next() {
		var entry OrderItemsHistory
		var changedAt time.Time
		if err := rows.Scan(&entry.Status, &changedAt); err != nil {
			return nil, fmt.Errorf("store: scan status history: %w", err)
		}
		entry.ChangedAt = changedAt.UTC().Format(time.RFC3339)
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate status history: %w", err)
	}
	return history, nil
}
