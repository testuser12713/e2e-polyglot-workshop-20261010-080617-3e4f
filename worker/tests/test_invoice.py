"""Tests for the invoice calculation (positions, hourly rate, 19% VAT, cents)."""

from __future__ import annotations

from datetime import datetime

from worker.invoice import (
    KIND_LABOR,
    KIND_PART,
    Invoice,
    build_invoice,
    labor_total_cents,
    round_half_up,
)

VAT_PERCENT = 19
HOURLY_RATE_CENTS = 8000


def _order(*items: dict) -> dict:
    return {
        "order_number": "A-1001",
        "customer_email": "kunde@example.com",
        "items": list(items),
    }


def test_labor_and_parts_are_summed_with_19_percent_vat() -> None:
    order = _order(
        {
            "kind": KIND_LABOR,
            "description": "Bremsen wechseln",
            "quantity": 90,
            "unit_price_cents": 0,
        },
        {
            "kind": KIND_PART,
            "description": "Bremsbelag",
            "quantity": 2,
            "unit_price_cents": 1500,
        },
    )

    invoice = build_invoice(order, HOURLY_RATE_CENTS, VAT_PERCENT)

    assert isinstance(invoice, Invoice)
    assert invoice.order_number == "A-1001"
    # 90 minutes of labor at 80.00 EUR/h == 120.00 EUR.
    assert invoice.items[0].total_cents == 12000
    # 2 brake pads at 15.00 EUR == 30.00 EUR.
    assert invoice.items[1].total_cents == 3000
    assert invoice.net_cents == 15000
    assert invoice.vat_cents == 2850
    assert invoice.gross_cents == 17850
    assert invoice.gross_cents == invoice.net_cents + invoice.vat_cents


def test_labor_uses_the_configured_hourly_rate_not_the_stored_price() -> None:
    order = _order(
        {
            "kind": KIND_LABOR,
            "description": "Inspektion",
            "quantity": 60,
            "unit_price_cents": 999999,
        }
    )

    invoice = build_invoice(order, 9500, VAT_PERCENT)

    assert invoice.items[0].unit_price_cents == 9500
    assert invoice.items[0].total_cents == 9500
    assert labor_total_cents(60, 9500) == 9500
    assert labor_total_cents(30, 9500) == 4750


def test_all_amounts_are_integer_cents_and_rounded_half_up() -> None:
    order = _order(
        {
            "kind": KIND_LABOR,
            "description": "Kleinarbeit",
            "quantity": 15,
            "unit_price_cents": 0,
        },
        {
            "kind": KIND_PART,
            "description": "Schraube",
            "quantity": 3,
            "unit_price_cents": 111,
        },
    )

    invoice = build_invoice(order, HOURLY_RATE_CENTS, VAT_PERCENT)

    # 15 min -> 2000 cents; 3 * 111 -> 333 cents; net 2333.
    assert invoice.net_cents == 2333
    # 2333 * 19 / 100 == 443.27 -> 443 cents, not 443.27.
    assert invoice.vat_cents == 443
    assert invoice.gross_cents == 2776
    for value in (invoice.net_cents, invoice.vat_cents, invoice.gross_cents):
        assert isinstance(value, int)
    assert all(isinstance(item.total_cents, int) for item in invoice.items)


def test_single_minute_labor_rounds_to_a_whole_cent() -> None:
    order = _order(
        {"kind": KIND_LABOR, "description": "Minute", "quantity": 1, "unit_price_cents": 0}
    )

    invoice = build_invoice(order, 9999, VAT_PERCENT)

    # 9999 / 60 == 166.65 -> 167 cents.
    assert invoice.items[0].total_cents == 167


def test_empty_order_yields_zero_amounts() -> None:
    invoice = build_invoice(_order(), HOURLY_RATE_CENTS, VAT_PERCENT)

    assert invoice.items == []
    assert invoice.net_cents == 0
    assert invoice.vat_cents == 0
    assert invoice.gross_cents == 0


def test_created_at_is_set() -> None:
    invoice = build_invoice(_order(), HOURLY_RATE_CENTS, VAT_PERCENT)
    assert isinstance(invoice.created_at, datetime)


def test_unknown_item_kind_is_rejected() -> None:
    order = _order(
        {"kind": "discount", "description": "Rabatt", "quantity": 1, "unit_price_cents": 100}
    )

    try:
        build_invoice(order, HOURLY_RATE_CENTS, VAT_PERCENT)
    except ValueError:
        pass
    else:  # pragma: no cover - the branch only runs on a broken implementation
        raise AssertionError("an unknown item kind must raise ValueError")


def test_round_half_up_rounds_halves_away_from_zero() -> None:
    assert round_half_up(4, 10) == 0
    assert round_half_up(5, 10) == 1
    assert round_half_up(15, 10) == 2
    assert round_half_up(25, 10) == 3
    assert round_half_up(-5, 10) == -1
