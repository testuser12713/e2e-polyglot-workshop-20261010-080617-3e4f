import { ComingSoon } from "../../components/ComingSoon";

/**
 * Workshop sign-in (route "/werkstatt/login"). Stub: authentication is
 * implemented by its own ticket. Until then the page stays inert.
 */
export default function LoginPage() {
  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Werkstatt-Anmeldung</h1>
        <p className="page__subtitle">
          Melden Sie sich mit Ihrem Mitarbeiterkonto an.
        </p>
      </header>
      <div className="card">
        <div className="card__body">
          <ComingSoon note="Die Anmeldung wird gerade fertiggestellt." />
        </div>
      </div>
    </section>
  );
}
