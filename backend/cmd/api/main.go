// Command api runs the workshop HTTP API. It loads configuration, connects to
// PostgreSQL, applies migrations, bootstraps the first employee when credentials
// are configured and serves the router on PORT.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"workshop/internal/auth"
	"workshop/internal/config"
	"workshop/internal/httpapi"
	"workshop/internal/queue"
	"workshop/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	pool, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("api: database connection failed: %w", err)
	}
	defer pool.Close()

	if err := store.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("api: migration failed: %w", err)
	}

	if err := auth.BootstrapInitialEmployee(ctx, pool, cfg.InitialEmployeeEmail, cfg.InitialEmployeePassword); err != nil {
		return fmt.Errorf("api: initial employee bootstrap failed: %w", err)
	}

	if err := queue.Configure(cfg.ValkeyURL, cfg.InvoiceQueueKey); err != nil {
		return fmt.Errorf("api: queue configuration failed: %w", err)
	}
	defer func() { _ = queue.Close() }()

	server := httpapi.NewServer(pool, cfg)
	addr := ":" + cfg.Port
	log.Printf("api listening on %s", addr)
	if err := http.ListenAndServe(addr, httpapi.NewRouter(server)); err != nil {
		return fmt.Errorf("api: server stopped: %w", err)
	}
	return nil
}
