package store

import (
	"context"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"

	"workshop/migrations"
)

// Migrate applies every embedded migration that has not been applied yet. It is
// safe to call on every startup: applied versions are recorded in
// schema_migrations and skipped, and each migration runs inside its own
// transaction so a failure leaves the schema unchanged.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("store: create schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("store: read migrations: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		if err := applyMigration(ctx, pool, name); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, name string) error {
	var applied bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name,
	).Scan(&applied); err != nil {
		return fmt.Errorf("store: check migration %s: %w", name, err)
	}
	if applied {
		return nil
	}

	sql, err := migrations.FS.ReadFile(name)
	if err != nil {
		return fmt.Errorf("store: read migration %s: %w", name, err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin migration %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("store: apply migration %s: %w", name, err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (version) VALUES ($1)`, name,
	); err != nil {
		return fmt.Errorf("store: record migration %s: %w", name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store: commit migration %s: %w", name, err)
	}
	return nil
}
