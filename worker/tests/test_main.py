"""End-to-end test: one queue message becomes one invoice in PostgreSQL."""

from __future__ import annotations

import json
import os

import psycopg
import redis
from worker.config import WorkerConfig

from worker import main

ORDER_NUMBER = "A-5001"


def _config(queue_key: str) -> WorkerConfig:
    return WorkerConfig(
        database_url=os.environ["DATABASE_URL"],
        valkey_url=os.environ["VALKEY_URL"],
        hourly_rate_cents=8000,
        vat_percent=19,
        invoice_queue_key=queue_key,
    )


def _seed_order(conn: psycopg.Connection, order_number: str = ORDER_NUMBER) -> None:
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO customers (name, email, phone) VALUES (%s, %s, %s) RETURNING id",
            ("Max Mustermann", "max@example.com", "0123"),
        )
        customer_id = cur.fetchone()[0]
        cur.execute(
            "INSERT INTO orders (order_number, customer_id, status) VALUES (%s, %s, %s)",
            (order_number, customer_id, "fertig"),
        )
        cur.execute(
            """
            INSERT INTO order_items (order_number, kind, description, quantity, unit_price_cents)
            VALUES
                (%s, 'labor', 'Bremsen wechseln', 90, 0),
                (%s, 'part', 'Bremsbelag', 2, 1500)
            """,
            (order_number, order_number),
        )
    conn.commit()


def _count(conn: psycopg.Connection, table: str) -> int:
    with conn.cursor() as cur:
        cur.execute(f"SELECT count(*) FROM {table}")
        return cur.fetchone()[0]


def test_one_queue_message_becomes_one_invoice(
    pg_conn: psycopg.Connection,
    valkey_client: redis.Redis,
    queue_key: str,
) -> None:
    _seed_order(pg_conn)
    config = _config(queue_key)
    valkey_client.lpush(queue_key, json.dumps({"order_number": ORDER_NUMBER}))

    handled = main.process_one(pg_conn, valkey_client, config)

    assert handled is True
    assert _count(pg_conn, "invoices") == 1
    assert _count(pg_conn, "invoice_items") == 2
    assert _count(pg_conn, "outbox") == 1
    with pg_conn.cursor() as cur:
        cur.execute(
            "SELECT net_cents, vat_cents, gross_cents FROM invoices WHERE order_number = %s",
            (ORDER_NUMBER,),
        )
        assert cur.fetchone() == (15000, 2850, 17850)
        cur.execute("SELECT recipient FROM outbox WHERE order_number = %s", (ORDER_NUMBER,))
        assert cur.fetchone()[0] == "max@example.com"


def test_reprocessing_the_same_order_creates_no_second_invoice(
    pg_conn: psycopg.Connection,
    valkey_client: redis.Redis,
    queue_key: str,
) -> None:
    _seed_order(pg_conn)
    config = _config(queue_key)

    valkey_client.lpush(queue_key, json.dumps({"order_number": ORDER_NUMBER}))
    assert main.process_one(pg_conn, valkey_client, config) is True
    valkey_client.lpush(queue_key, json.dumps({"order_number": ORDER_NUMBER}))
    assert main.process_one(pg_conn, valkey_client, config) is True

    assert _count(pg_conn, "invoices") == 1
    assert _count(pg_conn, "invoice_items") == 2
    assert _count(pg_conn, "outbox") == 1


def test_empty_queue_is_not_an_error(
    pg_conn: psycopg.Connection,
    valkey_client: redis.Redis,
    queue_key: str,
) -> None:
    assert main.process_one(pg_conn, valkey_client, _config(queue_key)) is False


def test_unknown_order_is_skipped(
    pg_conn: psycopg.Connection,
    valkey_client: redis.Redis,
    queue_key: str,
) -> None:
    config = _config(queue_key)
    valkey_client.lpush(queue_key, json.dumps({"order_number": "A-9999"}))

    assert main.process_one(pg_conn, valkey_client, config) is True
    assert _count(pg_conn, "invoices") == 0


def test_malformed_payload_is_skipped_without_crashing(
    pg_conn: psycopg.Connection,
    valkey_client: redis.Redis,
    queue_key: str,
) -> None:
    config = _config(queue_key)
    valkey_client.lpush(queue_key, "not-json")

    assert main.process_one(pg_conn, valkey_client, config) is True
    assert _count(pg_conn, "invoices") == 0
