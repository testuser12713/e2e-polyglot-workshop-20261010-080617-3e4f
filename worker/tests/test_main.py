"""End-to-end test: one queue message becomes one invoice in PostgreSQL."""

from __future__ import annotations

import json
import os
from typing import Any

import psycopg
import pytest
import redis
from worker.config import WorkerConfig

from worker import main

ORDER_NUMBER = "A-5001"

ITEMS = [
    {"kind": "labor", "description": "Bremsen wechseln", "quantity": 90, "unit_price_cents": 0},
    {"kind": "part", "description": "Bremsbelag", "quantity": 2, "unit_price_cents": 1500},
]


def _config(queue_key: str) -> WorkerConfig:
    return WorkerConfig(
        database_url=os.environ["DATABASE_URL"],
        valkey_url=os.environ["VALKEY_URL"],
        hourly_rate_cents=8000,
        vat_percent=19,
        invoice_queue_key=queue_key,
    )


def _count(conn: psycopg.Connection, table: str) -> int:
    with conn.cursor() as cur:
        cur.execute(f"SELECT count(*) FROM {table}")
        return cur.fetchone()[0]


def test_one_queue_message_becomes_one_invoice(
    pg_conn: psycopg.Connection,
    make_order: Any,
    valkey_client: redis.Redis,
    queue_key: str,
) -> None:
    order = make_order(ORDER_NUMBER, items=ITEMS)
    config = _config(queue_key)
    valkey_client.lpush(queue_key, json.dumps({"order_number": ORDER_NUMBER}))

    handled = main.process_one(pg_conn, valkey_client, config)

    assert handled is True
    assert _count(pg_conn, "invoices") == 1
    assert _count(pg_conn, "invoice_items") == 2
    assert _count(pg_conn, "notifications") == 1
    with pg_conn.cursor() as cur:
        cur.execute(
            """
            SELECT net_cents, vat_cents, gross_cents
            FROM invoices WHERE order_number = %s
            """,
            (ORDER_NUMBER,),
        )
        assert cur.fetchone() == (15000, 2850, 17850)
        cur.execute(
            "SELECT recipient FROM notifications WHERE order_id = %s",
            (order["order_id"],),
        )
        assert cur.fetchone()[0] == order["customer_email"]


def test_reprocessing_the_same_order_creates_no_second_invoice(
    pg_conn: psycopg.Connection,
    make_order: Any,
    valkey_client: redis.Redis,
    queue_key: str,
) -> None:
    make_order(ORDER_NUMBER, items=ITEMS)
    config = _config(queue_key)

    valkey_client.lpush(queue_key, json.dumps({"order_number": ORDER_NUMBER}))
    assert main.process_one(pg_conn, valkey_client, config) is True
    valkey_client.lpush(queue_key, json.dumps({"order_number": ORDER_NUMBER}))
    assert main.process_one(pg_conn, valkey_client, config) is True

    assert _count(pg_conn, "invoices") == 1
    assert _count(pg_conn, "invoice_items") == 2
    assert _count(pg_conn, "notifications") == 1


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


def test_broken_order_does_not_kill_the_worker(
    monkeypatch: pytest.MonkeyPatch,
    pg_conn: psycopg.Connection,
    valkey_client: redis.Redis,
    queue_key: str,
) -> None:
    broken_items = [
        {"kind": "discount", "description": "Rabatt", "quantity": 1, "unit_price_cents": 100}
    ]

    def fake_load_order(conn: psycopg.Connection, order_number: str) -> dict[str, Any]:
        return {
            "order_id": 1,
            "order_number": order_number,
            "customer_email": "kunde@example.com",
            "items": broken_items,
        }

    monkeypatch.setattr(main.store, "load_order", fake_load_order)
    config = _config(queue_key)
    valkey_client.lpush(queue_key, json.dumps({"order_number": "A-6001"}))

    # A ValueError while building the invoice is caught: process_one returns
    # normally instead of tearing the worker down.
    assert main.process_one(pg_conn, valkey_client, config) is True
    assert _count(pg_conn, "invoices") == 0
