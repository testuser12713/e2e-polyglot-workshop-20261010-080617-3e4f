import { apiFetch } from "./client";
import type { Invoice, StatusLookup } from "./types";

/**
 * Customer-facing lookups of the public API.
 *
 * Both routes are keyed by the pair (order number, licence plate): the API only
 * answers with data when both match the same order and otherwise returns a
 * 404, so the caller never learns whether the order number exists at all
 * (AC-07 / AC-31).
 */

/** GET /api/orders/status?order_number=&license_plate= */
export function lookupOrderStatus(
  orderNumber: string,
  licensePlate: string,
): Promise<StatusLookup> {
  const query = new URLSearchParams({
    order_number: orderNumber,
    license_plate: licensePlate,
  });
  return apiFetch<StatusLookup>(`/api/orders/status?${query.toString()}`);
}

/** GET /api/orders/{order_number}/invoice?license_plate= */
export function fetchInvoice(
  orderNumber: string,
  licensePlate: string,
): Promise<Invoice> {
  const query = new URLSearchParams({ license_plate: licensePlate });
  const path = `/api/orders/${encodeURIComponent(orderNumber)}/invoice?${query.toString()}`;
  return apiFetch<Invoice>(path);
}
