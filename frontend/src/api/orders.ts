import { apiFetch } from "./client";
import type { Status } from "./types";

/** Customer contact data sent with a new order request. */
export interface OrderRequestCustomer {
  name: string;
  email: string;
  phone: string;
}

/** Vehicle data sent with a new order request. */
export interface OrderRequestVehicle {
  license_plate: string;
  make: string;
  model: string;
  mileage_km: number;
}

/** Body of POST /api/orders. */
export interface OrderRequest {
  customer: OrderRequestCustomer;
  vehicle: OrderRequestVehicle;
  preferred_date: string;
  problem_description: string;
}

/** Response of POST /api/orders (201). */
export interface OrderCreated {
  order_number: string;
  status: Status;
  created_at: string;
}

/**
 * Create a workshop order request (AC-04).
 *
 * Calls POST /api/orders through the shared {@link apiFetch}, which parses the
 * shared error envelope and raises an `ApiError` carrying its machine code.
 */
export function createOrder(payload: OrderRequest): Promise<OrderCreated> {
  return apiFetch<OrderCreated>("/api/orders", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}
