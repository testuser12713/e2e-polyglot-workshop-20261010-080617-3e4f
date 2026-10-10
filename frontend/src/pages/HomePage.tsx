import { Link } from "react-router-dom";
import "./HomePage.css";

/**
 * Start hub (route "/"). The landing page of the portal: one header plus the
 * two hub regions Kundenbereich and Werkstattbereich, each linking into its
 * area. Every link leads to a declared route of the shared web contract, on the
 * own origin only (AC-29).
 */
export default function HomePage() {
  return (
    <section className="page" data-od-id="main-hub">
      <header className="page__header" data-od-id="page-header">
        <h1>Willkommen im Werkstattportal</h1>
        <p className="page__subtitle">
          Wählen Sie Ihren Bereich. Kunden buchen Termine und verfolgen Aufträge
          — die Werkstatt verwaltet Aufträge und Rechnungen.
        </p>
      </header>

      <div className="hub-grid">
        <section
          className="card hub-card"
          data-od-id="hub-kundenbereich"
          aria-labelledby="hub-kunde-title"
        >
          <h2 id="hub-kunde-title">Kundenbereich</h2>
          <p>
            Werkstattauftrag anfragen, Status mit Auftragsnummer und Kennzeichen
            abrufen, fertige Rechnung einsehen.
          </p>
          <Link className="btn btn--primary btn--block-mobile" to="/status">
            Zum Kundenbereich
          </Link>
          <ul className="hub-links">
            <li>
              <Link to="/anfrage">Auftrag anfragen</Link>
            </li>
            <li>
              <Link to="/status">Status abrufen</Link>
            </li>
            <li>
              <Link to="/rechnung">Rechnung ansehen</Link>
            </li>
          </ul>
        </section>

        <section
          className="card hub-card"
          data-od-id="hub-werkstattbereich"
          aria-labelledby="hub-werkstatt-title"
        >
          <h2 id="hub-werkstatt-title">Werkstattbereich</h2>
          <p>
            Anmelden, offene Aufträge einsehen, Positionen erfassen, Status
            setzen und den Werkstatt-Dashboard nutzen.
          </p>
          <Link
            className="btn btn--secondary btn--block-mobile"
            to="/werkstatt/login"
          >
            Zur Anmeldung
          </Link>
          <ul className="hub-links">
            <li>
              <Link to="/werkstatt/login">Anmeldung Mitarbeiter</Link>
            </li>
            <li>
              <Link to="/werkstatt/dashboard">Dashboard</Link>
            </li>
            <li>
              <Link to="/werkstatt/auftraege">Auftragsliste</Link>
            </li>
          </ul>
        </section>
      </div>
    </section>
  );
}
