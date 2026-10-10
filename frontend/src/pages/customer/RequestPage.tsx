import { ComingSoon } from "../../components/ComingSoon";

/**
 * Customer appointment request (route "/"). Stub: the form is implemented by
 * its own ticket; until then the page renders no dead controls (AC-19).
 */
export default function RequestPage() {
  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Termin anfragen</h1>
        <p className="page__subtitle">
          Senden Sie uns Ihr Anliegen – wir melden uns mit einem Terminvorschlag.
        </p>
      </header>
      <div className="card">
        <div className="card__body">
          <ComingSoon note="Das Anfrageformular wird gerade fertiggestellt." />
        </div>
      </div>
    </section>
  );
}
