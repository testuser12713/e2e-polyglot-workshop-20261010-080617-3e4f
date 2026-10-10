package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/domain"
	"workshop/internal/queue"
	"workshop/internal/status"
)

// ErrOrderNotFound is returned when an order number does not exist.
var ErrOrderNotFound = errors.New("store: order not found")

// ErrInvalidTransition is returned when a status change is not a single legal
// step along the lifecycle. The stored status is left unchanged.
var ErrInvalidTransition = errors.New("store: invalid status transition")

// StatusOrderCustomer is the customer block of the workshop OrderDetail view.
type StatusOrderCustomer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// StatusOrderVehicle is the vehicle block of the workshop OrderDetail view.
type StatusOrderVehicle struct {
	LicensePlate string `json:"license_plate"`
	Make         string `json:"make"`
	Model        string `json:"model"`
	MileageKm    int    `json:"mileage_km"`
}

// StatusOrderItem is one invoice position of the workshop OrderDetail view.
type StatusOrderItem struct {
	Kind           string `json:"kind"`
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
	TotalCents     int    `json:"total_cents"`
}

// StatusOrderDetail is the full workshop order view returned after a status
// change. Its JSON shape matches the shared OrderDetail contract exactly.
type StatusOrderDetail struct {
	OrderNumber        string              `json:"order_number"`
	Status             domain.Status       `json:"status"`
	Customer           StatusOrderCustomer `json:"customer"`
	Vehicle            StatusOrderVehicle  `json:"vehicle"`
	PreferredDate      *string             `json:"preferred_date"`
	ProblemDescription string              `json:"problem_description"`
	Items              []StatusOrderItem   `json:"items"`
	StatusHistory      []StatusChange      `json:"status_history"`
}

// UpdateOrderStatus advances orderNumber to next inside a single transaction:
// it locks the order row, validates the transition against the lifecycle, then
// writes the new status and appends the change to the status protocol together
// (AC-05, AC-06). When the order reaches "fertig" it enqueues an invoice job on
// the Valkey queue (AC-08).
//
// It returns ErrOrderNotFound for an unknown order, ErrInvalidTransition for a
// jump or a backwards step (the status is never touched), and the new status on
// success.
func UpdateOrderStatus(ctx context.Context, pool *pgxpool.Pool, orderNumber string, next domain.Status) (domain.Status, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("store: begin status update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orderID int64
	var current domain.Status
	err = tx.QueryRow(ctx,
		`SELECT id, status FROM orders WHERE order_number = $1 FOR UPDATE`,
		orderNumber,
	).Scan(&orderID, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrOrderNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: load order %s: %w", orderNumber, err)
	}

	if current == next {
		return "", fmt.Errorf("%w: %s is already %s", ErrInvalidTransition, orderNumber, next)
	}
	if _, err := status.Transition(current, next); err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidTransition, err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE orders SET status = $1, updated_at = now() WHERE id = $2`,
		string(next), orderID,
	); err != nil {
		return "", fmt.Errorf("store: update order %s: %w", orderNumber, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO status_history (order_id, status, previous_status) VALUES ($1, $2, $3)`,
		orderID, string(next), string(current),
	); err != nil {
		return "", fmt.Errorf("store: record status history for %s: %w", orderNumber, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("store: commit status update %s: %w", orderNumber, err)
	}

	if next == domain.StatusDone {
		if err := queue.EnqueueInvoice(ctx, orderNumber); err != nil {
			return next, fmt.Errorf("store: enqueue invoice for %s: %w", orderNumber, err)
		}
	}
	return next, nil
}

// LoadStatusOrderDetail reads the full workshop order view (OrderDetail): the
// order with its customer, vehicle, positions and status protocol. It returns
// ErrOrderNotFound when the order number does not exist.
func LoadStatusOrderDetail(ctx context.Context, pool *pgxpool.Pool, orderNumber string) (*StatusOrderDetail, error) {
	var detail StatusOrderDetail
	var preferredDate *time.Time

	err := pool.QueryRow(ctx, `
		SELECT o.order_number, o.status,
		       c.name, c.email, c.phone,
		       v.license_plate, v.make, v.model, v.mileage_km,
		       o.preferred_date, o.problem_description
		FROM orders o
		JOIN customers c ON c.id = o.customer_id
		JOIN vehicles v ON v.id = o.vehicle_id
		WHERE o.order_number = $1`, orderNumber,
	).Scan(
		&detail.OrderNumber, &detail.Status,
		&detail.Customer.Name, &detail.Customer.Email, &detail.Customer.Phone,
		&detail.Vehicle.LicensePlate, &detail.Vehicle.Make, &detail.Vehicle.Model, &detail.Vehicle.MileageKm,
		&preferredDate, &detail.ProblemDescription,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: load order %s: %w", orderNumber, err)
	}
	if preferredDate != nil {
		formatted := preferredDate.Format("2006-01-02")
		detail.PreferredDate = &formatted
	}

	if detail.Items, err = loadStatusOrderItems(ctx, pool, orderNumber); err != nil {
		return nil, err
	}
	if detail.StatusHistory, err = LoadStatusHistory(ctx, pool, orderNumber); err != nil {
		return nil, err
	}
	return &detail, nil
}

func loadStatusOrderItems(ctx context.Context, pool *pgxpool.Pool, orderNumber string) ([]StatusOrderItem, error) {
	rows, err := pool.Query(ctx, `
		SELECT i.kind, i.description, i.quantity, i.unit_price_cents, i.total_cents
		FROM order_items i
		JOIN orders o ON o.id = i.order_id
		WHERE o.order_number = $1
		ORDER BY i.id ASC`, orderNumber)
	if err != nil {
		return nil, fmt.Errorf("store: load items for %s: %w", orderNumber, err)
	}
	defer rows.Close()

	items := make([]StatusOrderItem, 0)
	for rows.Next() {
		var item StatusOrderItem
		if err := rows.Scan(&item.Kind, &item.Description, &item.Quantity, &item.UnitPriceCents, &item.TotalCents); err != nil {
			return nil, fmt.Errorf("store: scan items for %s: %w", orderNumber, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: read items for %s: %w", orderNumber, err)
	}
	return items, nil
}
