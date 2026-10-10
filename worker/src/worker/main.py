"""Worker entry point.

The worker connects to PostgreSQL and to the Valkey queue, logs one ready line
and then blocks on ``BRPOP`` of the invoice queue. An empty queue is the normal
state and must not crash it.
"""

from __future__ import annotations

import logging
import time
from typing import Any

import psycopg
import redis

from worker.config import WorkerConfig

logger = logging.getLogger("worker")

READY_MESSAGE = "worker ready"

# A finite blocking timeout: the queue is polled in short blocking windows so an
# idle connection is never held open long enough for the server to drop it, and
# the loop simply waits again. It still blocks rather than spin-waits.
BRPOP_TIMEOUT_SECONDS = 3
RECONNECT_DELAY_SECONDS = 1.0


def configure_logging() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )


def consume_queue(
    conn: psycopg.Connection[Any],
    client: redis.Redis,
    config: WorkerConfig,
) -> None:
    """Block on the invoice queue forever; an empty queue is not an error."""
    while True:
        try:
            job = client.brpop(config.invoice_queue_key, timeout=BRPOP_TIMEOUT_SECONDS)
        except (redis.exceptions.ConnectionError, redis.exceptions.TimeoutError):
            logger.warning("lost the Valkey connection while waiting; reconnecting")
            time.sleep(RECONNECT_DELAY_SECONDS)
            client = redis.Redis.from_url(config.valkey_url, decode_responses=True)
            continue
        if job is None:
            continue
        _key, payload = job
        logger.info("received an invoice job from queue '%s'", config.invoice_queue_key)
        logger.debug("invoice job payload: %s", payload)


def run(config: WorkerConfig) -> None:
    """Connect to both stores, announce readiness and consume the queue."""
    with psycopg.connect(config.database_url) as conn:
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
