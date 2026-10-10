"""Tests for invoice persistence and idempotence (store)."""

from __future__ import annotations

from datetime import UTC, datetime

import psycopg
from worker.invoice import KIND_LABOR, KIND_PART, Invoice, InvoiceItem
from worker.store import invoice_exists, save_invoice_and_notification


def _invoice(order_number: str = "A-2001") -> Invoice:
    return Invoice(
        order_number=order_number,
        items=[
            InvoiceItem(KIND_LABOR, "Bremsen wechseln", 90, 8000, 12000),
            InvoiceItem(KIND_PART, "Bremsbelag", 2, 1500, 3000),
        ],
        net_cents=15000,
        vat_cents=2850,
        gross_cents=17850,
        created_at=datetime.now(UTC),
    )


def _notification(order_number: str = "A-2001") -> dict:
    return {
        "order_number": order_number,
        "recipient": "kunde@example.com",
        "subject": f"Ihre Rechnung zum Auftrag {order_number}",
        "body": "Die Rechnung ist fertig.",
        "channel": "email",
    }


def _count(conn: psycopg.Connection, table: str) -> int:
    with conn.cursor() as cur:
        cur.execute(f"SELECT count(*) FROM {table}")
        return cur.fetchone()[0]


def test_invoice_exists_is_false_before_writing(pg_conn: psycopg.Connection) -> None:
    assert invoice_exists(pg_conn, "A-2001") is False


def test_saving_writes_invoice_items_and_outbox(pg_conn: psycopg.Connection) -> None:
    saved = save_invoice_and_notification(pg_conn, _invoice(), _notification())

    assert saved is True
    assert invoice_exists(pg_conn, "A-2001") is True
    assert _count(pg_conn, "invoices") == 1
    assert _count(pg_conn, "invoice_items") == 2
    assert _count(pg_conn, "outbox") == 1

    with pg_conn.cursor() as cur:
        cur.execute(
            "SELECT net_cents, vat_cents, gross_cents FROM invoices WHERE order_number = %s",
            ("A-2001",),
        )
        assert cur.fetchone() == (15000, 2850, 17850)
        cur.execute(
            "SELECT recipient, subject, channel FROM outbox WHERE order_number = %s",
            ("A-2001",),
        )
        recipient, subject, channel = cur.fetchone()
        assert recipient == "kunde@example.com"
        assert "A-2001" in subject
        assert channel == "email"


def test_reprocessing_the_same_order_creates_no_second_invoice(
    pg_conn: psycopg.Connection,
) -> None:
    assert save_invoice_and_notification(pg_conn, _invoice(), _notification()) is True

    saved_again = save_invoice_and_notification(pg_conn, _invoice(), _notification())

    assert saved_again is False
    assert _count(pg_conn, "invoices") == 1
    assert _count(pg_conn, "invoice_items") == 2
    assert _count(pg_conn, "outbox") == 1


def test_different_orders_get_their_own_invoice(pg_conn: psycopg.Connection) -> None:
    save_invoice_and_notification(pg_conn, _invoice("A-3001"), _notification("A-3001"))
    save_invoice_and_notification(pg_conn, _invoice("A-3002"), _notification("A-3002"))

    assert invoice_exists(pg_conn, "A-3001") is True
    assert invoice_exists(pg_conn, "A-3002") is True
    assert _count(pg_conn, "invoices") == 2
    assert _count(pg_conn, "outbox") == 2
