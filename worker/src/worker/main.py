"""Worker entry point.

The worker connects to PostgreSQL and to the Valkey queue, logs one ready line
and then blocks on ``BRPOP`` of the invoice queue. Every message is a JSON object
with an ``order_number``: the worker loads the order, checks whether an invoice
already exists for it, and otherwise stores the invoice with its items plus one
outbox notification for the customer. An empty queue is the normal state and must
not crash the worker; a job that cannot be processed is logged and skipped.
"""

from __future__ import annotations

import json
import logging
import time
from typing import Any

import psycopg
import redis

from worker import store
from worker.config import WorkerConfig
from worker.invoice import Invoice, build_invoice

logger = logging.getLogger("worker")

READY_MESSAGE = "worker ready"

# A finite blocking timeout: the queue is polled in short blocking windows so an
# idle connection is never held open long enough for the server to drop it, and
# the loop simply waits again. It still blocks rather than spin-waits.
BRPOP_TIMEOUT_SECONDS = 3
RECONNECT_DELAY_SECONDS = 1.0

NOTIFICATION_CHANNEL = "email"
NOTIFICATION_STATUS = "queued"


def configure_logging() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )


def handle_job(
    conn: psycopg.Connection[Any],
    config: WorkerConfig,
    payload: str,
) -> Invoice | None:
    """Turn one queue payload into an invoice and its outbox notification.

    Returns the stored invoice, or ``None`` when the job was skipped (an order
    that is unknown, an already invoiced order, or a malformed payload). The
    order number is the only identifier ever logged; no customer data, plate or
    problem description is written to the log.
    """
    try:
        data = json.loads(payload)
    except (TypeError, ValueError):
        logger.warning("discarding an invoice job with an unreadable payload")
        return None

    if not isinstance(data, dict):
        logger.warning("discarding an invoice job whose payload is not an object")
        return None

    order_number = data.get("order_number")
    if not order_number:
        logger.warning("discarding an invoice job without an order_number")
        return None
    order_number = str(order_number)

    if store.invoice_exists(conn, order_number):
        logger.info("invoice for order %s already exists; skipping", order_number)
        return None

    order = store.load_order(conn, order_number)
    if order is None:
        logger.warning("no order found for %s; skipping the invoice job", order_number)
        return None

    invoice = build_invoice(order, config.hourly_rate_cents, config.vat_percent)
    notification = build_notification(order, invoice)
    stored = store.save_invoice_and_notification(conn, invoice, notification)
    if stored:
        logger.info("stored invoice for order %s", order_number)
    else:
        logger.info("invoice for order %s was written concurrently; skipping", order_number)
        return None
    return invoice


def build_notification(order: dict[str, Any], invoice: Invoice) -> dict[str, Any]:
    """Build the outbox notification row for a customer's finished invoice.

    This only records the notification in the outbox; it never sends a real
    e-mail.
    """
    gross = f"{invoice.gross_cents / 100:.2f}".replace(".", ",")
    return {
        "order_id": order.get("order_id"),
        "order_number": invoice.order_number,
        "recipient": order.get("customer_email"),
        "subject": f"Ihre Rechnung zum Auftrag {invoice.order_number}",
        "body": (
            f"Guten Tag,\n\n"
            f"die Rechnung zu Ihrem Auftrag {invoice.order_number} ist fertig. "
            f"Der Gesamtbetrag beträgt {gross} EUR.\n\n"
            f"Mit freundlichen Grüßen\nIhre Werkstatt"
        ),
        "channel": NOTIFICATION_CHANNEL,
        "status": NOTIFICATION_STATUS,
    }


def process_one(
    conn: psycopg.Connection[Any],
    client: redis.Redis,
    config: WorkerConfig,
) -> bool:
    """Pop at most one job and process it; return whether a job was handled."""
    job = client.brpop(config.invoice_queue_key, timeout=BRPOP_TIMEOUT_SECONDS)
    if job is None:
        return False
    _key, payload = job
    logger.info("received an invoice job from queue '%s'", config.invoice_queue_key)
    try:
        handle_job(conn, config, payload)
    except Exception as exc:
        conn.rollback()
        logger.error(
            "failed to process an invoice job (%s); the job was skipped",
            type(exc).__name__,
        )
    return True


def consume_queue(
    conn: psycopg.Connection[Any],
    client: redis.Redis,
    config: WorkerConfig,
) -> None:
    """Block on the invoice queue forever; an empty queue is not an error."""
    while True:
        try:
            process_one(conn, client, config)
        except (redis.exceptions.ConnectionError, redis.exceptions.TimeoutError):
            logger.warning("lost the Valkey connection while waiting; reconnecting")
            time.sleep(RECONNECT_DELAY_SECONDS)
            client = redis.Redis.from_url(config.valkey_url, decode_responses=True)
            continue


def run(config: WorkerConfig) -> None:
    """Connect to both stores, announce readiness and consume the queue."""
    with psycopg.connect(config.database_url, autocommit=True) as conn:
        client = redis.Redis.from_url(config.valkey_url, decode_responses=True)
        try:
            client.ping()
        except redis.exceptions.ConnectionError as exc:
            raise RuntimeError(f"cannot reach Valkey at VALKEY_URL: {exc}") from exc
        logger.info(
            "%s: connected to PostgreSQL and Valkey, waiting for jobs on '%s'",
            READY_MESSAGE,
            config.invoice_queue_key,
        )
        consume_queue(conn, client, config)


def main() -> int:
    configure_logging()
    try:
        config = WorkerConfig.from_env()
        run(config)
    except KeyboardInterrupt:
        logger.info("worker stopped")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
