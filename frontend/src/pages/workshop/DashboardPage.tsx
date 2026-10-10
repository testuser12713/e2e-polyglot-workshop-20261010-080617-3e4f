import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ApiError, type Order } from "../../api/types";
import {
  fetchDashboardMetrics,
  fetchRecentFinishedOrders,
  type DashboardMetrics,
} from "../../api/dashboard";
import { useAuth } from "../../auth/AuthContext";
import { ComingSoon } from "../../components/ComingSoon";
import { ErrorBanner } from "../../components/ErrorBanner";
import { StatusBadge } from "../../components/StatusBadge";
import "./dashboard.css";

const RECENT_LIMIT = 5;

const MONEY_FORMATTER = new Intl.NumberFormat("de-DE", {
  style: "currency",
  currency: "EUR",
});

/**
 * Render integer cents as German currency, e.g. 123456 -> "1.234,56 €".
 * Display is the only place money is converted; the API stays in cents
 * (DESIGN.md formatting rule).
 */
export function formatCents(cents: number): string {
  const safe = Number.isFinite(cents) ? Math.round(cents) : 0;
  return MONEY_FORMATTER.format(safe / 100);
}

export interface StatTileProps {
  label: string;
  value: string;
  delta?: string;
  spanFull?: boolean;
  oid?: string;
}

/** One dashboard key figure (DESIGN.md StatTile). */
export function StatTile({ label, value, delta, spanFull, oid }: StatTileProps) {
  const className = spanFull ? "stat-tile stat-tile--span-full" : "stat-tile";
  return (
    <div className={className} data-od-id={oid}>
      <p className="stat-tile__label">{label}</p>
      <p className="stat-tile__value tabular">{value}</p>
      {delta ? <p className="stat-tile__delta">{delta}</p> : null}
    </div>
  );
}

/**
 * Workshop dashboard (route "/werkstatt/dashboard"). Shows the open orders,
 * the orders finished today and the current month's revenue, plus a table of
 * recently finished orders and quick links into the order list. The route is
 * guarded by ProtectedRoute; without a session the visitor lands on the login.
 */
export default function DashboardPage() {
  const { token } = useAuth();
  const [metrics, setMetrics] = useState<DashboardMetrics | null>(null);
  const [recentOrders, setRecentOrders] = useState<Order[]>([]);
  const [error, setError] = useState<ApiError | null>(null);
  const [loading, setLoading] = useState(true);
  const [reloadKey, setReloadKey] = useState(0);

  useEffect(() => {
    if (!token) {
      setLoading(false);
      return;
    }

    let cancelled = false;
    setLoading(true);
    setError(null);

    Promise.all([fetchDashboardMetrics(token), fetchRecentFinishedOrders(token)])
      .then(([nextMetrics, orders]) => {
        if (cancelled) return;
        setMetrics(nextMetrics);
        setRecentOrders(orders.slice(0, RECENT_LIMIT));
      })
      .catch((cause: unknown) => {
        if (cancelled) return;
        setMetrics(null);
        setRecentOrders([]);
        setError(cause instanceof ApiError ? cause : new ApiError("internal_error"));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [token, reloadKey]);

  return (
    <section className="page">
      <header className="page__header">
        <h1>Dashboard</h1>
        <p className="page__subtitle">
          Offene Aufträge, heute fertiggestellte Aufträge und Monatsumsatz.
        </p>
      </header>

      {error ? (
        <div className="dashboard-error">
          <ErrorBanner error={error} />
          <button
            type="button"
            className="btn btn--secondary"
            onClick={() => setReloadKey((key) => key + 1)}
          >
            Erneut versuchen
          </button>
        </div>
      ) : null}

      {loading ? (
        <div className="dashboard-state" role="status" aria-live="polite">
          Dashboard wird geladen…
        </div>
      ) : null}

      {metrics && !loading ? (
        <section
          className="section"
          data-od-id="stat-tiles"
          aria-label="Kennzahlen"
        >
          <div className="stat-grid">
            <StatTile
              oid="stat-open"
              label="Offene Aufträge"
              value={String(metrics.open_orders)}
            />
            <StatTile
              oid="stat-done-today"
              label="Heute fertig geworden"
              value={String(metrics.completed_today)}
            />
            <StatTile
              oid="stat-revenue"
              spanFull
              label="Umsatz laufender Monat"
              value={formatCents(metrics.revenue_month_cents)}
            />
          </div>
        </section>
      ) : null}

      <section
        className="section"
        data-od-id="dashboard-recent"
        aria-labelledby="recent-title"
      >
        <div className="card">
          <div className="card__header">
            <h2 className="card__title" id="recent-title">
              Zuletzt fertiggestellte Aufträge
            </h2>
            <Link className="btn btn--ghost" to="/werkstatt/auftraege">
              Alle Aufträge
            </Link>
          </div>

          {recentOrders.length === 0 ? (
            <div className="empty-state">
              <p className="empty-state__title">Noch keine fertiggestellten Aufträge</p>
              <p className="empty-state__text">
                Sobald ein Auftrag fertig ist, erscheint er hier.
              </p>
            </div>
          ) : (
            <div className="table-wrap">
              <table className="data-table" data-od-id="recent-table">
                <thead>
                  <tr>
                    <th scope="col">Auftragsnummer</th>
                    <th scope="col">Kennzeichen</th>
                    <th scope="col">Kunde</th>
                    <th scope="col">Status</th>
                    <th scope="col" className="data-table__amount">
                      Betrag
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {recentOrders.map((order) => (
                    <tr key={order.order_number}>
                      <td className="mono">
                        <Link to={`/werkstatt/auftraege/${order.order_number}`}>
                          {order.order_number}
                        </Link>
                      </td>
                      <td className="mono">{order.license_plate}</td>
                      <td>{order.customer_name}</td>
                      <td>
                        <StatusBadge status={order.status} showDot />
                      </td>
                      <td className="data-table__amount tabular">
                        {formatCents(order.total_cents)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </section>

      <section
        className="section"
        data-od-id="dashboard-actions"
        aria-labelledby="actions-title"
      >
        <h2 className="section__title" id="actions-title">
          Schnellaktionen
        </h2>
        <div className="dashboard-actions">
          <Link className="btn btn--primary" to="/werkstatt/auftraege">
            Auftragsliste öffnen
          </Link>
          <ComingSoon label="Neuer Auftrag" note="kommt bald" />
        </div>
      </section>
    </section>
  );
}
