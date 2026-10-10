package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/internal/domain"
)

// DashboardMetrics is the workshop dashboard snapshot: how many orders are still
// open, how many were finished today and the revenue invoiced in the current
// month. Counts are absolute and revenue is integer cents.
type DashboardMetrics struct {
	OpenOrders        int64
	CompletedToday    int64
	RevenueMonthCents int64
}

// GetDashboardMetrics reads all three numbers in a single query round.
//
//   - open_orders: every order that has not been picked up ("abgeholt"), i.e.
//     still needs workshop attention.
//   - completed_today: orders that moved to "fertig" since the start of the
//     current day, taken from the status history. An order that was finished
//     today and picked up again the same day still counts.
//   - revenue_month_cents: the gross total of every invoice created in the
//     current calendar month.
//
// All time windows use the database clock (now()) so the dashboard agrees with
// the timestamps the rest of the API writes.
func GetDashboardMetrics(ctx context.Context, pool *pgxpool.Pool) (DashboardMetrics, error) {
	const query = `
		SELECT
			(SELECT count(*) FROM orders WHERE status <> $1) AS open_orders,
			(SELECT count(*) FROM status_history
			  WHERE status = $2
			    AND changed_at >= date_trunc('day', now())
			    AND changed_at <  date_trunc('day', now()) + interval '1 day') AS completed_today,
			(SELECT COALESCE(sum(gross_cents), 0) FROM invoices
			  WHERE created_at >= date_trunc('month', now())
			    AND created_at <  date_trunc('month', now()) + interval '1 month') AS revenue_month_cents`

	var metrics DashboardMetrics
	if err := pool.QueryRow(ctx, query,
		string(domain.StatusPickedUp),
		string(domain.StatusDone),
	).Scan(
		&metrics.OpenOrders,
		&metrics.CompletedToday,
		&metrics.RevenueMonthCents,
	); err != nil {
		return DashboardMetrics{}, fmt.Errorf("store: read dashboard metrics: %w", err)
	}
	return metrics, nil
}
