"""Tests for invoice persistence and idempotence (store)."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Any

import psycopg
from worker.invoice import KIND_LABOR, KIND_PART, Invoice, InvoiceItem
from worker.store import invoice_exists, save_invoice_and_notification


def _invoice(order_number: str) -> Invoice:
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


def _notification(order: dict[str, Any]) -> dict[str, Any]:
    return {
        "order_id": order["order_id"],
        "order_number": order["order_number"],
        "recipient": order["customer_email"],
        "subject": f"Ihre Rechnung zum Auftrag {order['order_number']}",
        "body": "Die Rechnung ist fertig.",
        "channel": "email",
        "status": "queued",
    }


def _count(conn: psycopg.Connection, table: str) -> int:
    with conn.cursor() as cur:
        cur.execute(f"SELECT count(*) FROM {table}")
        return cur.fetchone()[0]


def test_invoice_exists_is_false_before_writing(
    pg_conn: psycopg.Connection,
    make_order: Any,
) -> None:
    order = make_order("A-2001")

    assert invoice_exists(pg_conn, order["order_number"]) is False


def test_saving_writes_invoice_items_and_notification(
    pg_conn: psycopg.Connection,
    make_order: Any,
) -> None:
    order = make_order("A-2001")

    saved = save_invoice_and_notification(pg_conn, _invoice("A-2001"), _notification(order))

    assert saved is True
    assert invoice_exists(pg_conn, "A-2001") is True
    assert _count(pg_conn, "invoices") == 1
    assert _count(pg_conn, "invoice_items") == 2
    assert _count(pg_conn, "notifications") == 1

    with pg_conn.cursor() as cur:
        cur.execute(
            """
            SELECT net_cents, vat_cents, gross_cents
            FROM invoices WHERE order_number = %s
            """,
            ("A-2001",),
        )
        assert cur.fetchone() == (15000, 2850, 17850)
        cur.execute(
            "SELECT recipient, subject, status FROM notifications WHERE order_id = %s",
            (order["order_id"],),
        )
        recipient, subject, status = cur.fetchone()
        assert recipient == order["customer_email"]
        assert "A-2001" in subject
        assert status == "queued"


def test_reprocessing_the_same_order_creates_no_second_invoice(
    pg_conn: psycopg.Connection,
    make_order: Any,
) -> None:
    order = make_order("A-2001")

    assert save_invoice_and_notification(pg_conn, _invoice("A-2001"), _notification(order)) is True

    saved_again = save_invoice_and_notification(pg_conn, _invoice("A-2001"), _notification(order))

    assert saved_again is False
    assert _count(pg_conn, "invoices") == 1
    assert _count(pg_conn, "invoice_items") == 2
    assert _count(pg_conn, "notifications") == 1


def test_different_orders_get_their_own_invoice(
    pg_conn: psycopg.Connection,
    make_order: Any,
) -> None:
    first = make_order("A-3001")
    second = make_order("A-3002")

    save_invoice_and_notification(pg_conn, _invoice("A-3001"), _notification(first))
    save_invoice_and_notification(pg_conn, _invoice("A-3002"), _notification(second))

    assert invoice_exists(pg_conn, "A-3001") is True
    assert invoice_exists(pg_conn, "A-3002") is True
    assert _count(pg_conn, "invoices") == 2
    assert _count(pg_conn, "notifications") == 2
