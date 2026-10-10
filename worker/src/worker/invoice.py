"""Invoice data shapes and the public calculation entry point.

This module owns the shared invoice model of the worker and the calculation that
turns an order (its labor and part positions), the configured hourly rate and the
VAT percent into an invoice. Every amount is an integer number of cents.

Labor positions are billed as ``quantity`` minutes times the configured hourly
rate (``quantity * hourly_rate_cents / 60``); part positions are billed as
``quantity * unit_price_cents``. Both are rounded half-up to whole cents, as is
the VAT, so no cent is ever lost to a floating-point artifact.
"""

from __future__ import annotations

from collections.abc import Mapping
from dataclasses import dataclass
from datetime import UTC, datetime
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


def round_half_up(numerator: int, denominator: int) -> int:
    """Divide two integers, rounding halves away from zero.

    Money must not depend on Python's banker's rounding: 0.5 cent rounds up,
    both for positive and (defensively) for negative numerators.
    """
    if denominator <= 0:
        raise ValueError("denominator must be positive")
    if numerator >= 0:
        return (numerator + denominator // 2) // denominator
    return -((-numerator + denominator // 2) // denominator)


def labor_total_cents(quantity_minutes: int, hourly_rate_cents: int) -> int:
    """Cost of ``quantity_minutes`` of labor at the configured hourly rate."""
    return round_half_up(quantity_minutes * hourly_rate_cents, 60)


def _as_int(raw: Any, default: int = 0) -> int:
    if raw is None or raw == "":
        return default
    return int(raw)


def build_invoice(
    order: Mapping[str, Any],
    hourly_rate_cents: int,
    vat_percent: int,
) -> Invoice:
    """Build an invoice from an order, the hourly rate and the VAT percent.

    ``order`` is the persisted order as a mapping with at least an
    ``order_number`` and an ``items`` sequence; each item carries ``kind``,
    ``description``, ``quantity`` and (for parts) ``unit_price_cents``. Labor is
    always priced with the configured hourly rate, never with a stored price.
    """
    order_number = str(order["order_number"])
    raw_items = order.get("items") or []
    items: list[InvoiceItem] = []
    net_cents = 0

    for raw in raw_items:
        kind = str(raw.get("kind", ""))
        description = str(raw.get("description", ""))
        quantity = _as_int(raw.get("quantity"))
        if kind == KIND_LABOR:
            unit_price_cents = int(hourly_rate_cents)
            total_cents = labor_total_cents(quantity, hourly_rate_cents)
        elif kind == KIND_PART:
            unit_price_cents = _as_int(raw.get("unit_price_cents"))
            total_cents = quantity * unit_price_cents
        else:
            raise ValueError(f"unknown invoice item kind: {kind!r}")
        items.append(
            InvoiceItem(
                kind=kind,
                description=description,
                quantity=quantity,
                unit_price_cents=unit_price_cents,
                total_cents=total_cents,
            )
        )
        net_cents += total_cents

    vat_cents = round_half_up(net_cents * int(vat_percent), 100)
    gross_cents = net_cents + vat_cents

    return Invoice(
        order_number=order_number,
        items=items,
        net_cents=net_cents,
        vat_cents=vat_cents,
        gross_cents=gross_cents,
        created_at=datetime.now(UTC),
    )
