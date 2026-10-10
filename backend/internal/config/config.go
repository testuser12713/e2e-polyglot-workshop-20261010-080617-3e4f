// Package config reads the API's runtime configuration from the environment.
//
// Configuration is read lazily: nothing here runs at import time, so a missing
// value never kills the process before it can log a helpful message. Load is
// called once from the startup path, which validates the whole settings object
// so a required-but-missing variable refuses to start and says which one.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Defaults for values that are safe to fall back on. Secrets never have a
// default here; they are either declared by the runner or the process refuses
// to boot with a message naming the variable.
const (
	DefaultPort            = "8080"
	DefaultHourlyRateCents = 9500
	// DefaultInvoiceQueueKey is the Valkey list the API pushes to and the
	// worker pops from. The worker's INVOICE_QUEUE_KEY must match this value.
	DefaultInvoiceQueueKey = "invoices"
	// DefaultCORSOrigin is used while the web service has not been declared yet
	// (its ${service:web.origin} stays unresolved then).
	DefaultCORSOrigin = "http://localhost:5173"
)

// Config is the fully resolved API configuration.
type Config struct {
	DatabaseURL             string
	ValkeyURL               string
	Port                    string
	HourlyRateCents         int
	InitialEmployeeEmail    string
	InitialEmployeePassword string
	CORSOrigin              string
	InvoiceQueueKey         string
}

// Load reads the configuration from the environment and validates it. It
// returns an error naming the missing variable when a required value is unset.
func Load() (*Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required but unset (see RUN.json)")
	}

	valkeyURL := strings.TrimSpace(os.Getenv("VALKEY_URL"))
	if valkeyURL == "" {
		return nil, fmt.Errorf("config: VALKEY_URL is required but unset (see RUN.json)")
	}

	rate, err := intEnv("HOURLY_RATE_CENTS", DefaultHourlyRateCents)
	if err != nil {
		return nil, err
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = DefaultPort
	}

	queueKey := strings.TrimSpace(os.Getenv("INVOICE_QUEUE_KEY"))
	if queueKey == "" {
		queueKey = DefaultInvoiceQueueKey
	}

	return &Config{
		DatabaseURL:             databaseURL,
		ValkeyURL:               valkeyURL,
		Port:                    port,
		HourlyRateCents:         rate,
		InitialEmployeeEmail:    strings.TrimSpace(os.Getenv("INITIAL_EMPLOYEE_EMAIL")),
		InitialEmployeePassword: os.Getenv("INITIAL_EMPLOYEE_PASSWORD"),
		CORSOrigin:              strings.TrimSpace(os.Getenv("CORS_ORIGIN")),
		InvoiceQueueKey:         queueKey,
	}, nil
}

func intEnv(name string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer, got %q", name, raw)
	}
	return value, nil
}
