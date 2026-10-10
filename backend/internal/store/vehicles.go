package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrDuplicatePlate is returned when a vehicle with the same license plate
// already exists. The HTTP layer maps it to a 409 conflict.
var ErrDuplicatePlate = errors.New("store: a vehicle with this license plate already exists")

// Vehicle is a vehicle record as returned to the public intake API.
type Vehicle struct {
	ID           int64  `json:"id"`
	LicensePlate string `json:"license_plate"`
	Make         string `json:"make"`
	Model        string `json:"model"`
	MileageKM    int    `json:"mileage_km"`
}

// CreateVehicle inserts a new vehicle and returns it with its generated id. It
// returns ErrDuplicatePlate when the license plate is already taken.
func CreateVehicle(ctx context.Context, pool *pgxpool.Pool, licensePlate, make, model string, mileageKM int) (Vehicle, error) {
	return createVehicle(ctx, pool, licensePlate, make, model, mileageKM)
}

func createVehicle(ctx context.Context, q querier, licensePlate, make, model string, mileageKM int) (Vehicle, error) {
	vehicle := Vehicle{
		LicensePlate: licensePlate,
		Make:         make,
		Model:        model,
		MileageKM:    mileageKM,
	}
	err := q.QueryRow(ctx,
		`INSERT INTO vehicles (license_plate, make, model, mileage_km)
		 VALUES ($1, $2, $3, $4) RETURNING id`,
		licensePlate, make, model, mileageKM,
	).Scan(&vehicle.ID)
	if err != nil {
		if isUniqueViolation(err) {
			return Vehicle{}, ErrDuplicatePlate
		}
		return Vehicle{}, err
	}
	return vehicle, nil
}

// findVehicleByPlate returns the vehicle with the given plate. The second
// result is false when no such vehicle exists.
func findVehicleByPlate(ctx context.Context, q querier, licensePlate string) (Vehicle, bool, error) {
	vehicle := Vehicle{}
	err := q.QueryRow(ctx,
		`SELECT id, license_plate, make, model, mileage_km
		 FROM vehicles WHERE license_plate = $1`,
		licensePlate,
	).Scan(&vehicle.ID, &vehicle.LicensePlate, &vehicle.Make, &vehicle.Model, &vehicle.MileageKM)
	if errors.Is(err, pgx.ErrNoRows) {
		return Vehicle{}, false, nil
	}
	if err != nil {
		return Vehicle{}, false, err
	}
	return vehicle, true, nil
}

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
