"""Shared fixtures for the invoice-generation tests.

The tests run against the real PostgreSQL and Valkey the worker uses, and they
build the schema from the API's actual migration
(``backend/migrations/0001_init.sql``) inside a throw-away PostgreSQL schema.
That way a column or table the product really has cannot hide behind a private
test schema, and every test owns only its own rows.
"""

from __future__ import annotations

import os
import uuid
from collections.abc import Callable, Iterator, Sequence
from pathlib import Path
from typing import Any

import psycopg
import pytest
import redis

TEST_SCHEMA = "worker_test"
MIGRATION_PATH = Path(__file__).resolve().parents[2] / "backend" / "migrations" / "0001_init.sql"

OrderSeeder = Callable[..., dict[str, Any]]


def database_url() -> str:
    url = os.environ.get("DATABASE_URL")
    if not url:
        pytest.fail("DATABASE_URL is not set; the worker tests need a real PostgreSQL")
    return url


def valkey_url() -> str:
    url = os.environ.get("VALKEY_URL")
    if not url:
        pytest.fail("VALKEY_URL is not set; the worker tests need a real Valkey")
    return url


@pytest.fixture()
def pg_conn() -> Iterator[psycopg.Connection]:
    """A connection whose search_path points at the real migrated test schema."""
    if not MIGRATION_PATH.is_file():
        pytest.fail(f"the API migration is missing at {MIGRATION_PATH}")
    migration_sql = MIGRATION_PATH.read_text()

    conn = psycopg.connect(database_url(), autocommit=True)
    with conn.cursor() as cur:
        cur.execute(f"DROP SCHEMA IF EXISTS {TEST_SCHEMA} CASCADE")
        cur.execute(f"CREATE SCHEMA {TEST_SCHEMA}")
        cur.execute(f"SET search_path TO {TEST_SCHEMA}")
        cur.execute(migration_sql)
    yield conn
    with conn.cursor() as cur:
        cur.execute("SET search_path TO public")
        cur.execute(f"DROP SCHEMA IF EXISTS {TEST_SCHEMA} CASCADE")
    conn.close()


@pytest.fixture()
def make_order(pg_conn: psycopg.Connection) -> OrderSeeder:
    """Insert a customer, a vehicle, an order and its positions; return the order.

    Returns a mapping with ``order_id``, ``order_number`` and ``customer_email``
    plus the seeded ``items``.
    """

    def _make(
        order_number: str = "A-1001",
        items: Sequence[dict[str, Any]] = (),
        email: str = "kunde@example.com",
        status: str = "fertig",
    ) -> dict[str, Any]:
        with pg_conn.cursor() as cur:
            cur.execute(
                "INSERT INTO customers (name, email, phone) VALUES (%s, %s, %s) RETURNING id",
                ("Max Mustermann", email, "0123"),
            )
            customer_id = cur.fetchone()[0]
            cur.execute(
                """
                INSERT INTO vehicles (license_plate, make, model, mileage_km)
                VALUES (%s, %s, %s, %s) RETURNING id
                """,
                (f"M-{uuid.uuid4().hex[:8].upper()}", "VW", "Golf", 100000),
            )
            vehicle_id = cur.fetchone()[0]
            cur.execute(
                """
                INSERT INTO orders (order_number, customer_id, vehicle_id, status)
                VALUES (%s, %s, %s, %s) RETURNING id
                """,
                (order_number, customer_id, vehicle_id, status),
            )
            order_id = cur.fetchone()[0]
            for item in items:
                cur.execute(
                    """
                    INSERT INTO order_items
                        (order_id, kind, description, quantity, unit_price_cents)
                    VALUES (%s, %s, %s, %s, %s)
                    """,
                    (
                        order_id,
                        item["kind"],
                        item["description"],
                        item["quantity"],
                        item.get("unit_price_cents", 0),
                    ),
                )
        return {
            "order_id": order_id,
            "order_number": order_number,
            "customer_email": email,
            "items": list(items),
        }

    return _make


@pytest.fixture()
def valkey_client() -> Iterator[redis.Redis]:
    client = redis.Redis.from_url(valkey_url(), decode_responses=True)
    yield client
    client.close()


@pytest.fixture()
def queue_key(valkey_client: redis.Redis) -> Iterator[str]:
    """A dedicated queue list that is emptied before and after the test."""
    key = f"test:invoices:{uuid.uuid4().hex}"
    valkey_client.delete(key)
    yield key
    valkey_client.delete(key)
