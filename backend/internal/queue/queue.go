// Package queue is the API's side of the Valkey invoice queue. The API pushes an
// order number with LPUSH; the Python worker BRPOPs it.
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// DefaultInvoiceQueueKey is the Valkey list name used when INVOICE_QUEUE_KEY is
// not set. The worker reads the same variable and must agree on this default.
const DefaultInvoiceQueueKey = "invoices"

// ErrNotConfigured is returned when Configure has not run (VALKEY_URL missing).
var ErrNotConfigured = errors.New("queue: not configured (VALKEY_URL)")

// Queue is a configured Valkey list client.
type Queue struct {
	client *redis.Client
	key    string
}

var defaultQueue *Queue

// Configure parses the Valkey address and installs the package-level queue used
// by EnqueueInvoice. It does not connect eagerly; the first command connects.
func Configure(valkeyURL, key string) error {
	options, err := redis.ParseURL(valkeyURL)
	if err != nil {
		return fmt.Errorf("queue: invalid VALKEY_URL: %w", err)
	}
	if key == "" {
		key = DefaultInvoiceQueueKey
	}
	defaultQueue = &Queue{client: redis.NewClient(options), key: key}
	return nil
}

// EnqueueInvoice pushes {"order_number":"..."} onto the invoice queue.
func EnqueueInvoice(ctx context.Context, orderNumber string) error {
	if defaultQueue == nil {
		return ErrNotConfigured
	}
	return defaultQueue.EnqueueInvoice(ctx, orderNumber)
}

// EnqueueInvoice pushes one order onto this queue.
func (q *Queue) EnqueueInvoice(ctx context.Context, orderNumber string) error {
	payload, err := json.Marshal(map[string]string{"order_number": orderNumber})
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, q.key, payload).Err()
}

// Close releases the underlying client, if any.
func Close() error {
	if defaultQueue == nil || defaultQueue.client == nil {
		return nil
	}
	return defaultQueue.client.Close()
}
