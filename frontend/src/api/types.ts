/*
 * Shared API types for the workshop portal.
 *
 * These mirror the sprint-wide interface exactly. Do not introduce a second
 * spelling for anything named here — other tickets import these names.
 */

export type Status =
  | "angefragt"
  | "bestaetigt"
  | "in_arbeit"
  | "fertig"
  | "abgeholt";

export const STATUS_VALUES: Status[] = [
  "angefragt",
  "bestaetigt",
  "in_arbeit",
  "fertig",
  "abgeholt",
];

export interface StatusHistoryEntry {
  status: Status;
  changed_at: string;
}

export interface InvoiceItem {
  kind: "labor" | "part";
  description: string;
  quantity: number;
  unit_price_cents: number;
  total_cents: number;
}

/** Workshop order list row (GET /api/workshop/orders). */
export interface Order {
  order_number: string;
  status: Status;
  license_plate: string;
  customer_name: string;
  preferred_date: string;
  total_cents: number;
}

export interface Customer {
  name: string;
  email: string;
  phone: string;
}

export interface Vehicle {
  license_plate: string;
  make: string;
  model: string;
  mileage_km: number;
}

export interface OrderDetail {
  order_number: string;
  status: Status;
  customer: Customer;
  vehicle: Vehicle;
  preferred_date: string;
  problem_description: string;
  items: InvoiceItem[];
  status_history: StatusHistoryEntry[];
}

export interface Invoice {
  order_number: string;
  items: InvoiceItem[];
  net_cents: number;
  vat_cents: number;
  gross_cents: number;
  created_at: string;
}

/** Response of GET /api/orders/status. */
export interface StatusLookup {
  order_number: string;
  status: Status;
  status_history: StatusHistoryEntry[];
}

/** Machine-readable error codes of the shared error envelope. */
export type ApiErrorCode =
  | "validation_error"
  | "not_found"
  | "conflict"
  | "invalid_transition"
  | "unauthorized"
  | "rate_limited"
  | "internal_error";

export interface ApiErrorEnvelope {
  error: {
    code: ApiErrorCode | string;
    message: string;
  };
}

const USER_MESSAGES: Record<string, string> = {
  validation_error: "Bitte prüfen Sie Ihre Eingaben.",
  not_found: "Es wurde nichts zu diesen Angaben gefunden.",
  conflict: "Diese Angabe ist bereits vorhanden.",
  invalid_transition: "Dieser Statuswechsel ist nicht möglich.",
  unauthorized: "Bitte melden Sie sich an.",
  rate_limited:
    "Zu viele Versuche. Bitte warten Sie einen Moment und versuchen Sie es erneut.",
  internal_error: "Es ist ein Fehler aufgetreten. Bitte versuchen Sie es erneut.",
};

/**
 * Error raised by {@link apiFetch} for any non-2xx response, a network failure
 * or an unreadable body. `message` is a human-readable German sentence derived
 * from the machine code; `code` keeps the machine code for programmatic use.
 */
export class ApiError extends Error {
  readonly code: string;
  readonly status: number;
  readonly envelope?: ApiErrorEnvelope;

  constructor(code: string, message?: string, status = 0, envelope?: ApiErrorEnvelope) {
    super(message ?? USER_MESSAGES[code] ?? USER_MESSAGES.internal_error);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
    this.envelope = envelope;
  }
}
