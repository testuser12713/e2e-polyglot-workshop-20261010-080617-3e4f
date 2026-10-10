package store

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmployeeNotFound is returned when no employee (or session) matches. The
// login handler maps it to 401 without revealing whether the e-mail exists.
var ErrEmployeeNotFound = errors.New("employee not found")

// Employee is a workshop employee. PasswordHash is read only by the login
// handler to verify a password; it must never be logged or returned to a client.
type Employee struct {
	ID           int64
	Name         string
	Email        string
	PasswordHash string
}

// FindEmployeeByEmail returns the employee with the given e-mail (matched
// case-insensitively) or ErrEmployeeNotFound. It includes the password hash so
// the caller can verify the login password.
func FindEmployeeByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (Employee, error) {
	var e Employee
	err := pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash
		   FROM employees
		  WHERE lower(email) = lower($1)`,
		email,
	).Scan(&e.ID, &e.Name, &e.Email, &e.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Employee{}, ErrEmployeeNotFound
	}
	if err != nil {
		return Employee{}, err
	}
	return e, nil
}

// CreateEmployeeSession stores a new session for employeeID. Only the hash of
// the session token is persisted; the raw token lives solely in the client.
func CreateEmployeeSession(ctx context.Context, pool *pgxpool.Pool, employeeID int64, tokenHash string, expiresAt time.Time) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO employee_sessions (employee_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		employeeID, tokenHash, expiresAt,
	)
	return err
}

// EmployeeBySessionToken returns the employee owning a non-expired session with
// the given token hash, or ErrEmployeeNotFound when the token is unknown or
// expired.
func EmployeeBySessionToken(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (Employee, error) {
	var e Employee
	err := pool.QueryRow(ctx,
		`SELECT e.id, e.name, e.email, e.password_hash
		   FROM employee_sessions s
		   JOIN employees e ON e.id = s.employee_id
		  WHERE s.token_hash = $1
		    AND s.expires_at > now()`,
		tokenHash,
	).Scan(&e.ID, &e.Name, &e.Email, &e.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Employee{}, ErrEmployeeNotFound
	}
	if err != nil {
		return Employee{}, err
	}
	return e, nil
}
