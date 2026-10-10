import { apiFetch } from "./client";
import type { Status } from "./types";

/**
 * Workshop order list (GET /api/workshop/orders).
 *
 * The list route is authenticated like every other /api/workshop/* route; the
 * caller passes the bearer token from the auth context. Filters are optional:
 * an omitted/"alle" status and a blank licence plate are simply not sent, so the
 * API answers with all orders.
 */
export type WorkshopOrderSummary = {
  order_number: string;
  status: Status;
  license_plate: string;
  customer_name: string;
  preferred_date: string;
  total_cents: number;
};

export interface WorkshopOrderQuery {
  /** Lifecycle state to filter by; "alle" (or omitted) means no status filter. */
  status?: Status | "alle";
  /** Partial licence plate to search for; blank (or omitted) means no search. */
  license_plate?: string;
}

/** Response envelope of GET /api/workshop/orders. */
interface WorkshopOrdersResponse {
  orders: WorkshopOrderSummary[];
}

/**
 * Load the workshop order list, optionally narrowed by status and/or a licence
 * plate fragment. Only non-empty parameters are appended to the query string.
 * Any error envelope from {@link apiFetch} surfaces unchanged to the caller.
 */
export function fetchWorkshopOrders(
  params: WorkshopOrderQuery,
  token: string,
): Promise<WorkshopOrderSummary[]> {
  const search = new URLSearchParams();

  if (params.status && params.status !== "alle") {
    search.set("status", params.status);
  }

  const plate = params.license_plate?.trim();
  if (plate) {
    search.set("license_plate", plate);
  }

  const query = search.toString();
  const path = query
    ? `/api/workshop/orders?${query}`
    : "/api/workshop/orders";

  return apiFetch<WorkshopOrdersResponse>(path, {
    headers: { Authorization: `Bearer ${token}` },
  }).then((body) => body.orders ?? []);
}
