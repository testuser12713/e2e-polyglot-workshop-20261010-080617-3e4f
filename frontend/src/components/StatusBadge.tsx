import type { Status } from "../api/types";

const LABELS: Record<Status, string> = {
  angefragt: "Angefragt",
  bestaetigt: "Bestätigt",
  in_arbeit: "In Arbeit",
  fertig: "Fertig",
  abgeholt: "Abgeholt",
};

const MODIFIERS: Record<Status, string> = {
  angefragt: "requested",
  bestaetigt: "confirmed",
  in_arbeit: "inprogress",
  fertig: "done",
  abgeholt: "pickedup",
};

export interface StatusBadgeProps {
  status: Status;
  showDot?: boolean;
}

/**
 * Pill showing one of the five lifecycle states. The state name is always
 * written out so colour is never the only signal.
 */
export function StatusBadge({ status, showDot = false }: StatusBadgeProps) {
  const modifier = MODIFIERS[status] ?? "requested";
  const label = LABELS[status] ?? status;

  return (
    <span
      className={`status-badge status-badge--${modifier}`}
      data-status={status}
      title={status}
    >
      {showDot ? <span className="status-badge__dot" aria-hidden="true" /> : null}
      {label}
    </span>
  );
}
