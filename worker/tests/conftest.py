"""Shared fixtures for the invoice-generation tests.

The tests run against the real PostgreSQL and Valkey the worker uses. Every test
gets a throw-away PostgreSQL schema (``search_path``) it builds itself, so it
never touches tables another service or another test owns, and a throw-away
Valkey list it empties on start and on teardown.
"""

from __future__ import annotations

import os
import uuid
from collections.abc import Iterator

import psycopg
import pytest
import redis

TEST_SCHEMA = "worker_test"

SCHEMA_STATEMENTS = (
    """
    CREATE TABLE customers (
        id BIGSERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        email TEXT NOT NULL,
        phone TEXT NOT NULL DEFAULT ''
    )
    """,
    """
    CREATE TABLE orders (
        id BIGSERIAL PRIMARY KEY,
        order_number TEXT NOT NULL UNIQUE,
        customer_id BIGINT REFERENCES customers(id),
        status TEXT NOT NULL DEFAULT 'angefragt'
    )
    """,
    """
    CREATE TABLE order_items (
        id BIGSERIAL PRIMARY KEY,
        order_number TEXT NOT NULL,
        kind TEXT NOT NULL,
        description TEXT NOT NULL,
        quantity INTEGER NOT NULL,
        unit_price_cents INTEGER NOT NULL DEFAULT 0
    )
    """,
    """
    CREATE TABLE invoices (
        id BIGSERIAL PRIMARY KEY,
        order_number TEXT NOT NULL UNIQUE,
        net_cents INTEGER NOT NULL,
        vat_cents INTEGER NOT NULL,
        gross_cents INTEGER NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT now()
    )
    """,
    """
    CREATE TABLE invoice_items (
        id BIGSERIAL PRIMARY KEY,
        invoice_id BIGINT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
        kind TEXT NOT NULL,
        description TEXT NOT NULL,
        quantity INTEGER NOT NULL,
        unit_price_cents INTEGER NOT NULL,
        total_cents INTEGER NOT NULL
    )
    """,
    """
    CREATE TABLE outbox (
        id BIGSERIAL PRIMARY KEY,
        order_number TEXT NOT NULL,
        recipient TEXT,
        subject TEXT NOT NULL DEFAULT '',
        body TEXT NOT NULL DEFAULT '',
        channel TEXT NOT NULL DEFAULT 'email',
        created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
        sent_at TIMESTAMPTZ
    )
    """,
)


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
    """A connection whose search_path points at a freshly built test schema."""
    with psycopg.connect(database_url()) as conn:
        with conn.cursor() as cur:
            cur.execute(f"DROP SCHEMA IF EXISTS {TEST_SCHEMA} CASCADE")
            cur.execute(f"CREATE SCHEMA {TEST_SCHEMA}")
            cur.execute(f"SET search_path TO {TEST_SCHEMA}")
            for statement in SCHEMA_STATEMENTS:
                cur.execute(statement)
        conn.commit()
        yield conn
        with conn.cursor() as cur:
            cur.execute("SET search_path TO public")
            cur.execute(f"DROP SCHEMA IF EXISTS {TEST_SCHEMA} CASCADE")
        conn.commit()


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
