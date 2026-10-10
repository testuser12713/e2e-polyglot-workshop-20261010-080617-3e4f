"""PostgreSQL persistence for the invoice worker.

Only the signatures are defined here; the invoice-generation ticket fills in the
SQL. All queries there must use bound parameters.
"""

from __future__ import annotations

from collections.abc import Mapping
from typing import Any

import psycopg

from worker.invoice import Invoice


def invoice_exists(conn: psycopg.Connection[Any], order_number: str) -> bool:
    """Return whether an invoice already exists for the given order number."""
    raise NotImplementedError("invoice_exists is implemented by the invoice-generation ticket")


def save_invoice_and_notification(
    conn: psycopg.Connection[Any],
    invoice: Invoice,
    notification: Mapping[str, Any],
) -> None:
    """Persist an invoice and its customer notification in one transaction."""
    raise NotImplementedError(
        "save_invoice_and_notification is implemented by the invoice-generation ticket"
    )
