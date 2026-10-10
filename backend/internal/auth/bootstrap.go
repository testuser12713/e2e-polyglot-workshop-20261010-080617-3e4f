// Package auth owns employee authentication: the initial-employee bootstrap
// here, the password hashing and the login rate limit added by the login ticket.
package auth

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// BootstrapInitialEmployee creates the first employee from configuration when
// none exists yet. It is idempotent: if any employee exists it does nothing.
//
// When email or password is empty (the variables are not configured) the
// bootstrap is skipped and the API still boots — that is the intended behaviour
// for a developer who has not declared initial credentials.
func BootstrapInitialEmployee(ctx context.Context, pool *pgxpool.Pool, email, password string) error {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil
	}

	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM employees)`,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	name := email
	if at := strings.Index(email, "@"); at > 0 {
		name = email[:at]
	}

	_, err = pool.Exec(ctx,
		`INSERT INTO employees (name, email, password_hash) VALUES ($1, $2, $3)`,
		name, email, string(hash),
	)
	return err
}
