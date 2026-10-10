import { ComingSoon } from "../../components/ComingSoon";

/**
 * Workshop order list (route "/werkstatt/auftraege"). Stub: list, filter and
 * search are implemented by their own ticket.
 */
export default function OrdersPage() {
  return (
    <section className="page">
      <header className="page__header">
        <h1>Aufträge</h1>
        <p className="page__subtitle">
          Alle Werkstattaufträge filtern und nach Kennzeichen durchsuchen.
        </p>
      </header>
      <div className="card">
        <div className="card__body">
          <ComingSoon note="Die Auftragsliste wird gerade fertiggestellt." />
        </div>
      </div>
    </section>
  );
}
