import { apiFetch } from "./client";
import type { Order } from "./types";

/**
 * The three figures of the workshop dashboard, exactly as
 * GET /api/workshop/dashboard returns them (integer cents for the revenue).
 */
export interface DashboardMetrics {
  open_orders: number;
  completed_today: number;
  revenue_month_cents: number;
}

/** Envelope of GET /api/workshop/orders. */
interface OrderListResponse {
  orders: Order[];
}

/**
 * Load the dashboard metrics behind the protected route. Every
 * /api/workshop/* route needs the bearer token; the shared client parses the
 * error envelope on failure.
 */
export function fetchDashboardMetrics(token: string): Promise<DashboardMetrics> {
  return apiFetch<DashboardMetrics>("/api/workshop/dashboard", {
    headers: { Authorization: `Bearer ${token}` },
  });
}

/**
 * Load the most recently finished workshop orders for the recent table, using
 * the order list endpoint filtered to the "fertig" status. The rows are ordered
 * newest first by the API.
 */
export async function fetchRecentFinishedOrders(token: string): Promise<Order[]> {
  const response = await apiFetch<OrderListResponse>(
    "/api/workshop/orders?status=fertig",
    { headers: { Authorization: `Bearer ${token}` } },
  );
  return Array.isArray(response.orders) ? response.orders : [];
}
