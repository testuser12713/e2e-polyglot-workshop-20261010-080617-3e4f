import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { ApiError, type Status } from "../../api/types";
import {
  fetchWorkshopOrders,
  type WorkshopOrderSummary,
} from "../../api/workshop";
import { useAuth } from "../../auth/AuthContext";
import { ErrorBanner } from "../../components/ErrorBanner";
import { StatusBadge } from "../../components/StatusBadge";
import { formatDate } from "../../lib/format";
import "./OrdersPage.css";

/** Debounce for the licence plate search so typing does not fire per keystroke. */
const PLATE_DEBOUNCE_MS = 300;

interface StatusChip {
  value: Status | "alle";
  label: string;
  modifier: string;
}

/** The 'Alle' chip plus one chip per lifecycle state (DESIGN.md FilterBar). */
const STATUS_CHIPS: StatusChip[] = [
  { value: "alle", label: "Alle", modifier: "alle" },
  { value: "angefragt", label: "angefragt", modifier: "requested" },
  { value: "bestaetigt", label: "bestätigt", modifier: "confirmed" },
  { value: "in_arbeit", label: "in Arbeit", modifier: "inprogress" },
  { value: "fertig", label: "fertig", modifier: "done" },
  { value: "abgeholt", label: "abgeholt", modifier: "pickedup" },
];

/** Render an ISO date (YYYY-MM-DD) as DD.MM.YYYY without timezone drift. */
function formatPreferredDate(value: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  if (match) return `${match[3]}.${match[2]}.${match[1]}`;
  return formatDate(value);
}

/**
 * Workshop order list (route "/werkstatt/auftraege").
 *
 * Shows every workshop order and narrows the list server-side through the
 * FilterBar: a segmented row of status chips plus a debounced licence plate
 * search. Both controls refetch GET /api/workshop/orders, so the table, the
 * aria-live result count and the empty state always reflect the current filter.
 * A 401 sends the visitor back to the workshop login.
 */
export default function OrdersPage() {
  const { token } = useAuth();
  const navigate = useNavigate();

  const [status, setStatus] = useState<Status | "alle">("alle");
  const [plate, setPlate] = useState("");
  const [debouncedPlate, setDebouncedPlate] = useState("");
  const [orders, setOrders] = useState<WorkshopOrderSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError | null>(null);

  useEffect(() => {
    const timer = window.setTimeout(
      () => setDebouncedPlate(plate),
      PLATE_DEBOUNCE_MS,
    );
    return () => window.clearTimeout(timer);
  }, [plate]);

  useEffect(() => {
    if (!token) {
      setLoading(false);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);

    fetchWorkshopOrders({ status, license_plate: debouncedPlate }, token)
      .then((next) => {
        if (cancelled) return;
        setOrders(next);
      })
      .catch((cause: unknown) => {
        if (cancelled) return;
        const apiError =
          cause instanceof ApiError ? cause : new ApiError("internal_error");
        setOrders([]);
        setError(apiError);
        if (apiError.status === 401) {
          navigate("/werkstatt/login", { replace: true });
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [token, status, debouncedPlate, navigate]);

  const countMessage = loading
    ? "Aufträge werden geladen…"
    : error
      ? ""
      : orders.length === 1
        ? "1 Auftrag wird angezeigt"
        : `${orders.length} Aufträge werden angezeigt`;

  return (
    <section className="page">
      <header className="page__header">
        <h1>Aufträge</h1>
        <p className="page__subtitle">
          Filtern Sie nach Status und suchen Sie nach Kennzeichen.
        </p>
      </header>

      <section
        className="orders-list"
        data-od-id="auftraege-list"
        aria-label="Auftragsliste"
      >
        <div className="filter-bar" data-od-id="filter-bar">
          <div
            className="filter-chips"
            role="group"
            aria-label="Nach Status filtern"
          >
            {STATUS_CHIPS.map((chip) => (
              <button
                key={chip.value}
                type="button"
                className={`filter-chip filter-chip--${chip.modifier}`}
                data-status={chip.value}
                aria-pressed={status === chip.value}
                onClick={() => setStatus(chip.value)}
              >
                {chip.label}
              </button>
            ))}
          </div>

          <div className="filter-search" data-od-id="plate-search" role="search">
            <label className="visually-hidden" htmlFor="plate-search">
              Kennzeichen suchen
            </label>
            <input
              id="plate-search"
              className="field__input field__input--mono"
              type="search"
              placeholder="Kennzeichen suchen"
              autoComplete="off"
              value={plate}
              onChange={(event) => setPlate(event.target.value)}
            />
          </div>
        </div>

        <p
          className="filter-count"
          data-od-id="filter-count"
          aria-live="polite"
        >
          {countMessage}
        </p>

        {error ? (
          <ErrorBanner error={error} />
        ) : loading ? (
          <div className="orders-state" role="status">
            Aufträge werden geladen…
          </div>
        ) : orders.length === 0 ? (
          <div className="card" data-od-id="empty-state">
            <div className="empty-state">
              <h2 className="empty-state__title">Keine Aufträge gefunden</h2>
              <p className="empty-state__text">
                Für diese Filter gibt es derzeit keine Aufträge. Ändern Sie den
                Statusfilter oder löschen Sie die Suche im Kennzeichen.
              </p>
            </div>
          </div>
        ) : (
          <div className="orders-table-wrap">
            <table className="orders-table" data-od-id="auftraege-table">
              <thead>
                <tr>
                  <th scope="col">Auftragsnummer</th>
                  <th scope="col">Status</th>
                  <th scope="col">Kennzeichen</th>
                  <th scope="col">Kunde</th>
                  <th scope="col">Wunschtermin</th>
                  <th scope="col" className="orders-table__actions">
                    Aktionen
                  </th>
                </tr>
              </thead>
              <tbody>
                {orders.map((order) => {
                  const href = `/werkstatt/auftraege/${order.order_number}`;
                  return (
                    <tr key={order.order_number}>
                      <td data-label="Auftragsnummer" className="mono">
                        <Link to={href}>{order.order_number}</Link>
                      </td>
                      <td data-label="Status">
                        <StatusBadge status={order.status} showDot />
                      </td>
                      <td
                        data-label="Kennzeichen"
                        className="mono orders-table__plate"
                      >
                        {order.license_plate}
                      </td>
                      <td data-label="Kunde">{order.customer_name}</td>
                      <td
                        data-label="Wunschtermin"
                        className="tabular orders-table__date"
                      >
                        {formatPreferredDate(order.preferred_date)}
                      </td>
                      <td
                        data-label="Aktionen"
                        className="orders-table__actions"
                      >
                        <Link className="btn btn--ghost" to={href}>
                          Öffnen
                        </Link>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </section>
  );
}
