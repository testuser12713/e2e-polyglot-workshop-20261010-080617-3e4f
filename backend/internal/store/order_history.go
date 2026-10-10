package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/domain"
)

// StatusChange is one entry of an order's status protocol (AC-06): the status
// that was reached and the instant it was reached.
type StatusChange struct {
	Status    domain.Status `json:"status"`
	ChangedAt time.Time     `json:"changed_at"`
}

// LoadStatusHistory returns the order's status protocol, oldest first. Every
// legal status change recorded by UpdateOrderStatus appears here. An unknown
// order number yields an empty, non-nil history.
func LoadStatusHistory(ctx context.Context, pool *pgxpool.Pool, orderNumber string) ([]StatusChange, error) {
	rows, err := pool.Query(ctx, `
		SELECT h.status, h.changed_at
		FROM status_history h
		JOIN orders o ON o.id = h.order_id
		WHERE o.order_number = $1
		ORDER BY h.changed_at ASC, h.id ASC`, orderNumber)
	if err != nil {
		return nil, fmt.Errorf("store: load status history for %s: %w", orderNumber, err)
	}
	defer rows.Close()

	history := make([]StatusChange, 0)
	for rows.Next() {
		var change StatusChange
		if err := rows.Scan(&change.Status, &change.ChangedAt); err != nil {
			return nil, fmt.Errorf("store: scan status history for %s: %w", orderNumber, err)
		}
		history = append(history, change)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: read status history for %s: %w", orderNumber, err)
	}
	return history, nil
}
