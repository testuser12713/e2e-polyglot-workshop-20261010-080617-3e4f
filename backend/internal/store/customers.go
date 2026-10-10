package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// querier is the read/execute surface shared by *pgxpool.Pool and pgx.Tx, so a
// helper can run either directly on the pool or inside a transaction. Every
// statement here is parameterised — no SQL is built from external input.
type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Customer is a customer record as returned to the public intake API.
type Customer struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// CreateCustomer inserts a new customer and returns it with its generated id.
func CreateCustomer(ctx context.Context, pool *pgxpool.Pool, name, email, phone string) (Customer, error) {
	return createCustomer(ctx, pool, name, email, phone)
}

func createCustomer(ctx context.Context, q querier, name, email, phone string) (Customer, error) {
	customer := Customer{Name: name, Email: email, Phone: phone}
	err := q.QueryRow(ctx,
		`INSERT INTO customers (name, email, phone) VALUES ($1, $2, $3) RETURNING id`,
		name, email, phone,
	).Scan(&customer.ID)
	if err != nil {
		return Customer{}, err
	}
	return customer, nil
}

// findCustomerByEmail returns the oldest customer with the given e-mail, so a
// repeated order request from the same person reuses one record. The second
// result is false when no such customer exists.
func findCustomerByEmail(ctx context.Context, q querier, email string) (Customer, bool, error) {
	customer := Customer{}
	err := q.QueryRow(ctx,
		`SELECT id, name, email, phone FROM customers WHERE email = $1 ORDER BY id LIMIT 1`,
		email,
	).Scan(&customer.ID, &customer.Name, &customer.Email, &customer.Phone)
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, false, nil
	}
	if err != nil {
		return Customer{}, false, err
	}
	return customer, true, nil
}
