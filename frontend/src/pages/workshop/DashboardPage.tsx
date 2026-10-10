import { ComingSoon } from "../../components/ComingSoon";

/**
 * Workshop dashboard (route "/werkstatt/dashboard"). Stub: the metrics are
 * implemented by their own ticket.
 */
export default function DashboardPage() {
  return (
    <section className="page">
      <header className="page__header">
        <h1>Dashboard</h1>
        <p className="page__subtitle">
          Offene Aufträge, heute fertiggestellte Aufträge und Monatsumsatz.
        </p>
      </header>
      <div className="card">
        <div className="card__body">
          <ComingSoon note="Das Dashboard wird gerade fertiggestellt." />
        </div>
      </div>
    </section>
  );
}
