package config

import (
	"strings"
	"testing"
)

func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://app@localhost:5432/app")
	t.Setenv("VALKEY_URL", "redis://localhost:6379/0")
	t.Setenv("PORT", "9090")
	t.Setenv("HOURLY_RATE_CENTS", "12345")
	t.Setenv("CORS_ORIGIN", "http://localhost:5173")
	t.Setenv("INVOICE_QUEUE_KEY", "invoice_queue")
	t.Setenv("INITIAL_EMPLOYEE_EMAIL", "chef@example.com")
	t.Setenv("INITIAL_EMPLOYEE_PASSWORD", "not-in-repo")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.DatabaseURL != "postgresql://app@localhost:5432/app" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.ValkeyURL != "redis://localhost:6379/0" {
		t.Errorf("ValkeyURL = %q", cfg.ValkeyURL)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q", cfg.Port)
	}
	if cfg.HourlyRateCents != 12345 {
		t.Errorf("HourlyRateCents = %d", cfg.HourlyRateCents)
	}
	if cfg.CORSOrigin != "http://localhost:5173" {
		t.Errorf("CORSOrigin = %q", cfg.CORSOrigin)
	}
	if cfg.InvoiceQueueKey != "invoice_queue" {
		t.Errorf("InvoiceQueueKey = %q", cfg.InvoiceQueueKey)
	}
	if cfg.InitialEmployeeEmail != "chef@example.com" {
		t.Errorf("InitialEmployeeEmail = %q", cfg.InitialEmployeeEmail)
	}
	if cfg.InitialEmployeePassword != "not-in-repo" {
		t.Errorf("InitialEmployeePassword = %q", cfg.InitialEmployeePassword)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://app@localhost:5432/app")
	t.Setenv("VALKEY_URL", "redis://localhost:6379/0")
	t.Setenv("PORT", "")
	t.Setenv("HOURLY_RATE_CENTS", "")
	t.Setenv("CORS_ORIGIN", "")
	t.Setenv("INVOICE_QUEUE_KEY", "")
	t.Setenv("INITIAL_EMPLOYEE_EMAIL", "")
	t.Setenv("INITIAL_EMPLOYEE_PASSWORD", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Port != DefaultPort {
		t.Errorf("Port = %q, want default %q", cfg.Port, DefaultPort)
	}
	if cfg.HourlyRateCents != DefaultHourlyRateCents {
		t.Errorf("HourlyRateCents = %d, want default %d", cfg.HourlyRateCents, DefaultHourlyRateCents)
	}
	if cfg.InvoiceQueueKey != DefaultInvoiceQueueKey {
		t.Errorf("InvoiceQueueKey = %q, want default %q", cfg.InvoiceQueueKey, DefaultInvoiceQueueKey)
	}
	if cfg.InitialEmployeeEmail != "" || cfg.InitialEmployeePassword != "" {
		t.Errorf("initial employee credentials should be empty when unset")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("VALKEY_URL", "redis://localhost:6379/0")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() succeeded without DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error should name DATABASE_URL, got: %v", err)
	}
}

func TestLoadRequiresValkeyURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://app@localhost:5432/app")
	t.Setenv("VALKEY_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() succeeded without VALKEY_URL")
	}
	if !strings.Contains(err.Error(), "VALKEY_URL") {
		t.Errorf("error should name VALKEY_URL, got: %v", err)
	}
}

func TestLoadRejectsNonIntegerRate(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://app@localhost:5432/app")
	t.Setenv("VALKEY_URL", "redis://localhost:6379/0")
	t.Setenv("HOURLY_RATE_CENTS", "cheap")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() accepted a non-integer HOURLY_RATE_CENTS")
	}
}
