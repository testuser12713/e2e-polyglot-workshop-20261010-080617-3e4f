import { apiFetch } from "./client";
import type { InvoiceItem, OrderDetail, Status } from "./types";

/**
 * API access for a single workshop order: load it, confirm it, capture labor /
 * part positions on it and advance its status.
 *
 * Every route needs a bearer token (the workshop area is authenticated) and the
 * paths match the sprint-wide contract exactly:
 *   GET  /api/workshop/orders/{order_number}
 *   POST /api/workshop/orders/{order_number}/confirm
 *   POST /api/workshop/orders/{order_number}/items
 *   POST /api/workshop/orders/{order_number}/status
 */

/** One captured position as the API expects it in POST .../items (no total). */
export type OrderItemInput = Omit<InvoiceItem, "total_cents">;

function bearer(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

function orderPath(orderNumber: string): string {
  return `/api/workshop/orders/${encodeURIComponent(orderNumber)}`;
}

/** Load the full detail of one workshop order. */
export function getWorkshopOrder(
  orderNumber: string,
  token: string,
): Promise<OrderDetail> {
  return apiFetch<OrderDetail>(orderPath(orderNumber), {
    headers: bearer(token),
  });
}

/** Confirm an "angefragt" order, moving it to "bestaetigt". */
export function confirmWorkshopOrder(
  orderNumber: string,
  token: string,
): Promise<OrderDetail> {
  return apiFetch<OrderDetail>(`${orderPath(orderNumber)}/confirm`, {
    method: "POST",
    headers: bearer(token),
  });
}

/** Capture one or more labor/part positions on an order. */
export function addWorkshopOrderItems(
  orderNumber: string,
  items: OrderItemInput[],
  token: string,
): Promise<OrderDetail> {
  return apiFetch<OrderDetail>(`${orderPath(orderNumber)}/items`, {
    method: "POST",
    headers: bearer(token),
    body: JSON.stringify({ items }),
  });
}

/** Advance the order to the requested next status. */
export function setWorkshopOrderStatus(
  orderNumber: string,
  status: Status,
  token: string,
): Promise<OrderDetail> {
  return apiFetch<OrderDetail>(`${orderPath(orderNumber)}/status`, {
    method: "POST",
    headers: bearer(token),
    body: JSON.stringify({ status }),
  });
}
