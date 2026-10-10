/*
 * Shared display formatters for money, dates and durations.
 *
 * DESIGN.md names ONE canonical formatting for every value in both areas:
 * money is rendered from integer cents (never float arithmetic for display),
 * dates follow DD.MM.YYYY, date + time DD.MM.YYYY, HH:MM Uhr and durations are
 * written in German hours/minutes. Other tickets render the same values, so the
 * helpers live here once and are imported instead of re-implemented.
 */

const NON_BREAKING_SPACE = "\u00A0";

function pad(value: number): string {
  return value.toString().padStart(2, "0");
}

/** Render integer cents as German currency, e.g. 123456 -> "1.234,56 €". */
export function formatEuroCents(cents: number): string {
  const value = Number.isFinite(cents) ? Math.trunc(cents) : 0;
  const negative = value < 0;
  const absolute = Math.abs(value);
  const euros = Math.floor(absolute / 100)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  const centsPart = pad(absolute % 100);
  return `${negative ? "-" : ""}${euros},${centsPart}${NON_BREAKING_SPACE}€`;
}

/** Parse a date value leniently; returns null for unparseable input. */
function toDate(value: string | Date): Date | null {
  const date = value instanceof Date ? value : new Date(value);
  return Number.isNaN(date.getTime()) ? null : date;
}

/** Render a date as DD.MM.YYYY, e.g. "05.03.2025". */
export function formatDate(value: string | Date): string {
  const date = toDate(value);
  if (!date) return "";
  return `${pad(date.getDate())}.${pad(date.getMonth() + 1)}.${date.getFullYear()}`;
}

/** Render a timestamp as DD.MM.YYYY, HH:MM Uhr, e.g. "05.03.2025, 09:14 Uhr". */
export function formatDateTime(value: string | Date): string {
  const date = toDate(value);
  if (!date) return "";
  return `${formatDate(date)}, ${pad(date.getHours())}:${pad(date.getMinutes())} Uhr`;
}

/** Render a minute count in German hours/minutes, e.g. 150 -> "2 h 30 min". */
export function formatMinutes(minutes: number): string {
  const value = Number.isFinite(minutes) ? Math.max(0, Math.trunc(minutes)) : 0;
  const hours = Math.floor(value / 60);
  const rest = value % 60;
  if (hours === 0) return `${rest} min`;
  if (rest === 0) return `${hours} h`;
  return `${hours} h ${rest} min`;
}

/**
 * Quantity of one invoice position. Labor is billed in minutes and shown as a
 * duration; parts are shown as a plain count.
 */
export function formatInvoiceQuantity(kind: string, quantity: number): string {
  return kind === "labor" ? formatMinutes(quantity) : String(quantity);
}
