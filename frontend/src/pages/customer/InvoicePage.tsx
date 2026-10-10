import { ComingSoon } from "../../components/ComingSoon";

/**
 * Customer invoice view (route "/rechnung"). Stub: the invoice lookup is
 * implemented by its own ticket.
 */
export default function InvoicePage() {
  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Rechnung</h1>
        <p className="page__subtitle">
          Sehen Sie die fertige Rechnung zu Ihrem Auftrag ein.
        </p>
      </header>
      <div className="card">
        <div className="card__body">
          <ComingSoon note="Die Rechnungsansicht wird gerade fertiggestellt." />
        </div>
      </div>
    </section>
  );
}
