import { ComingSoon } from "../../components/ComingSoon";

/**
 * Customer order status lookup (route "/status"). Stub: the lookup is
 * implemented by its own ticket.
 */
export default function StatusPage() {
  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Auftragsstatus</h1>
        <p className="page__subtitle">
          Verfolgen Sie den Fortschritt Ihres Auftrags mit Auftragsnummer und
          Kennzeichen.
        </p>
      </header>
      <div className="card">
        <div className="card__body">
          <ComingSoon note="Die Statusabfrage wird gerade fertiggestellt." />
        </div>
      </div>
    </section>
  );
}
