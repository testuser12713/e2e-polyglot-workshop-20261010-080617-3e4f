package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/domain"
)

// ErrNotFound signals that no resource matches the caller's input. The customer
// lookup queries answer it identically whether the order number or the license
// plate was the wrong half, so a caller can never learn which part matched
// (AC-31: no hint whether an order exists).
var ErrNotFound = errors.New("store: not found")

// LookupStatusChange is one entry of an order's status history.
type LookupStatusChange struct {
	Status    domain.Status `json:"status"`
	ChangedAt time.Time     `json:"changed_at"`
}

// OrderStatusLookup is the customer-facing status view. It deliberately carries
// only the contract's lookup shape — the order number, the current status and
// its history — and never customer, contact or vehicle data (AC-31).
type OrderStatusLookup struct {
	OrderNumber   string               `json:"order_number"`
	Status        domain.Status        `json:"status"`
	StatusHistory []LookupStatusChange `json:"status_history"`
}

// LookupInvoiceItem is one position of an invoice, matching the contract's
// InvoiceItem shape. Money is integer cents.
type LookupInvoiceItem struct {
	Kind           string `json:"kind"`
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	UnitPriceCents int    `json:"unit_price_cents"`
	TotalCents     int    `json:"total_cents"`
}

// InvoiceLookup is the customer-facing invoice view. Like OrderStatusLookup it
// exposes only the invoice of the matched order, no personal data (AC-31).
type InvoiceLookup struct {
	OrderNumber string              `json:"order_number"`
	Items       []LookupInvoiceItem `json:"items"`
	NetCents    int                 `json:"net_cents"`
	VatCents    int                 `json:"vat_cents"`
	GrossCents  int                 `json:"gross_cents"`
	CreatedAt   time.Time           `json:"created_at"`
}

// LookupOrderStatus returns the status and status history of the order whose
// order number AND license plate both match. If no single order satisfies both
// it returns ErrNotFound. Every query is parameterised (AC-25).
func LookupOrderStatus(ctx context.Context, pool *pgxpool.Pool, orderNumber, licensePlate string) (OrderStatusLookup, error) {
	var (
		orderID int64
		result  OrderStatusLookup
	)
	err := pool.QueryRow(ctx, `
		SELECT o.id, o.order_number, o.status
		FROM orders o
		JOIN vehicles v ON v.id = o.vehicle_id
		WHERE o.order_number = $1 AND v.license_plate = $2`,
		orderNumber, licensePlate,
	).Scan(&orderID, &result.OrderNumber, &result.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrderStatusLookup{}, ErrNotFound
	}
	if err != nil {
		return OrderStatusLookup{}, fmt.Errorf("store: lookup order status: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT status, changed_at
		FROM status_history
		WHERE order_id = $1
		ORDER BY changed_at ASC, id ASC`,
		orderID,
	)
	if err != nil {
		return OrderStatusLookup{}, fmt.Errorf("store: lookup status history: %w", err)
	}
	defer rows.Close()

	result.StatusHistory = []LookupStatusChange{}
	for rows.Next() {
		var change LookupStatusChange
		if err := rows.Scan(&change.Status, &change.ChangedAt); err != nil {
			return OrderStatusLookup{}, fmt.Errorf("store: scan status history: %w", err)
		}
		result.StatusHistory = append(result.StatusHistory, change)
	}
	if err := rows.Err(); err != nil {
		return OrderStatusLookup{}, fmt.Errorf("store: iterate status history: %w", err)
	}
	return result, nil
}

// LookupInvoice returns the invoice of the order whose order number AND license
// plate both match, with its items and the net, VAT and gross totals. If no
// such invoice exists it returns ErrNotFound.
func LookupInvoice(ctx context.Context, pool *pgxpool.Pool, orderNumber, licensePlate string) (InvoiceLookup, error) {
	var (
		invoiceID int64
		result    InvoiceLookup
	)
	err := pool.QueryRow(ctx, `
		SELECT i.id, i.order_number, i.net_cents, i.vat_cents, i.gross_cents, i.created_at
		FROM invoices i
		JOIN orders o ON o.id = i.order_id
		JOIN vehicles v ON v.id = o.vehicle_id
		WHERE o.order_number = $1 AND v.license_plate = $2`,
		orderNumber, licensePlate,
	).Scan(&invoiceID, &result.OrderNumber, &result.NetCents, &result.VatCents, &result.GrossCents, &result.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return InvoiceLookup{}, ErrNotFound
	}
	if err != nil {
		return InvoiceLookup{}, fmt.Errorf("store: lookup invoice: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT kind, description, quantity, unit_price_cents, total_cents
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY id ASC`,
		invoiceID,
	)
	if err != nil {
		return InvoiceLookup{}, fmt.Errorf("store: lookup invoice items: %w", err)
	}
	defer rows.Close()

	result.Items = []LookupInvoiceItem{}
	for rows.Next() {
		var item LookupInvoiceItem
		if err := rows.Scan(&item.Kind, &item.Description, &item.Quantity, &item.UnitPriceCents, &item.TotalCents); err != nil {
			return InvoiceLookup{}, fmt.Errorf("store: scan invoice item: %w", err)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return InvoiceLookup{}, fmt.Errorf("store: iterate invoice items: %w", err)
	}
	return result, nil
}
