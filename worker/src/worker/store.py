"""PostgreSQL persistence for the invoice worker.

All SQL here uses bound parameters only; no statement is ever assembled from
external input. The worker writes the finished invoice with its items into
``invoices`` / ``invoice_items`` and queues one notification row in the
``notifications`` outbox for the customer. It never sends a real e-mail.

The schema is owned by the API migration (``backend/migrations/0001_init.sql``):
``invoices`` is unique on ``order_id`` and ``order_items`` / ``notifications``
are keyed by ``order_id``, so every write resolves the order id first.
"""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any

import psycopg

from worker.invoice import Invoice, InvoiceItem

INSERT_INVOICE_SQL = """
INSERT INTO invoices (order_id, order_number, net_cents, vat_cents, gross_cents, created_at)
VALUES (%s, %s, %s, %s, %s, %s)
ON CONFLICT (order_id) DO NOTHING
RETURNING id
"""

INSERT_INVOICE_ITEM_SQL = """
INSERT INTO invoice_items
    (invoice_id, kind, description, quantity, unit_price_cents, total_cents)
VALUES (%s, %s, %s, %s, %s, %s)
"""

INSERT_NOTIFICATION_SQL = """
INSERT INTO notifications (order_id, channel, recipient, subject, body, status)
VALUES (%s, %s, %s, %s, %s, %s)
"""


def invoice_exists(conn: psycopg.Connection[Any], order_number: str) -> bool:
    """Return whether an invoice already exists for the given order number."""
    with conn.cursor() as cur:
        cur.execute(
            "SELECT 1 FROM invoices WHERE order_number = %s LIMIT 1",
            (order_number,),
        )
        return cur.fetchone() is not None


def load_order(
    conn: psycopg.Connection[Any],
    order_number: str,
) -> Mapping[str, Any] | None:
    """Load an order and its positions for invoice calculation.

    Returns ``None`` when no such order exists. The mapping carries the order id
    (the schema keys invoices, items and notifications by it), the customer's
    e-mail for the notification, and the positions to bill.
    """
    with conn.cursor() as cur:
        cur.execute(
            """
            SELECT o.id, c.email
            FROM orders o
            JOIN customers c ON c.id = o.customer_id
            WHERE o.order_number = %s
            """,
            (order_number,),
        )
        row = cur.fetchone()
        if row is None:
            return None
        order_id, customer_email = row

        cur.execute(
            """
            SELECT kind, description, quantity, unit_price_cents
            FROM order_items
            WHERE order_id = %s
            ORDER BY id
            """,
            (order_id,),
        )
        items = [
            {
                "kind": item_row[0],
                "description": item_row[1],
                "quantity": item_row[2],
                "unit_price_cents": item_row[3],
            }
            for item_row in cur.fetchall()
        ]

    return {
        "order_id": order_id,
        "order_number": order_number,
        "customer_email": customer_email,
        "items": items,
    }


def save_invoice_and_notification(
    conn: psycopg.Connection[Any],
    invoice: Invoice,
    notification: Mapping[str, Any],
) -> bool:
    """Persist an invoice, its items and one outbox notification in one transaction.

    Returns ``True`` when the invoice was newly written and ``False`` when an
    invoice for the same order already existed (the insert is guarded by
    ``ON CONFLICT (order_id) DO NOTHING``). In the latter case no item and no
    notification row is written, so reprocessing an order cannot double up.
    """
    order_id = notification.get("order_id")
    if order_id is None:
        raise ValueError("the notification is missing order_id")

    with conn.transaction(), conn.cursor() as cur:
        cur.execute(
            INSERT_INVOICE_SQL,
            (
                order_id,
                invoice.order_number,
                invoice.net_cents,
                invoice.vat_cents,
                invoice.gross_cents,
                invoice.created_at,
            ),
        )
        returned = cur.fetchone()
        if returned is None:
            return False
        invoice_id = returned[0]

        for item in invoice.items:
            _insert_item(cur, invoice_id, item)

        cur.execute(
            INSERT_NOTIFICATION_SQL,
            (
                order_id,
                notification.get("channel", "email"),
                notification.get("recipient"),
                notification.get("subject", ""),
                notification.get("body", ""),
                notification.get("status", "queued"),
            ),
        )
    return True


def _insert_item(cur: psycopg.Cursor[Any], invoice_id: int, item: InvoiceItem) -> None:
    cur.execute(
        INSERT_INVOICE_ITEM_SQL,
        (
            invoice_id,
            item.kind,
            item.description,
            item.quantity,
            item.unit_price_cents,
            item.total_cents,
        ),
    )
