"""Invoice data shapes and the public calculation entry point.

This module owns the shared invoice model of the worker. The calculation itself
is delivered by the invoice-generation ticket; the signature below is the one
that ticket implements so callers (and tests) can rely on it now.
"""

from __future__ import annotations

from collections.abc import Mapping
from dataclasses import dataclass
from datetime import datetime
from typing import Any

KIND_LABOR = "labor"
KIND_PART = "part"


@dataclass(frozen=True)
class InvoiceItem:
    """One invoice line; all amounts are integer cents."""

    kind: str
    description: str
    quantity: int
    unit_price_cents: int
    total_cents: int


@dataclass(frozen=True)
class Invoice:
    """A finished invoice; all amounts are integer cents."""

    order_number: str
    items: list[InvoiceItem]
    net_cents: int
    vat_cents: int
    gross_cents: int
    created_at: datetime


def build_invoice(
    order: Mapping[str, Any],
    hourly_rate_cents: int,
    vat_percent: int,
) -> Invoice:
    """Build an invoice from an order, the hourly rate and the VAT percent.

    Not implemented yet: the invoice-generation ticket fills in the calculation.
    """
    raise NotImplementedError("build_invoice is implemented by the invoice-generation ticket")
