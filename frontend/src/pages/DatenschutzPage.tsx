/**
 * Datenschutzerklärung (route "/datenschutz"). Stub: the legal copy is
 * delivered by its own ticket; the route and its footer link exist from the
 * start (AC-28).
 */
export default function DatenschutzPage() {
  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Datenschutzerklärung</h1>
        <p className="page__subtitle">
          Informationen zur Verarbeitung personenbezogener Daten.
        </p>
      </header>
      <div className="card">
        <div className="card__body">
          <p>Die Datenschutzerklärung wird in Kürze veröffentlicht.</p>
        </div>
      </div>
    </section>
  );
}
