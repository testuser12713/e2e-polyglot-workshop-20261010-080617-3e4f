import { Link } from "react-router-dom";

/**
 * Site-wide legal footer. The links to Impressum and Datenschutz are present on
 * every page (AC-28) and both targets stay on the own origin (AC-29).
 */
export function Footer() {
  return (
    <footer className="legal-footer">
      <div className="container legal-footer__inner">
        <ul className="legal-footer__links">
          <li>
            <Link to="/impressum">Impressum</Link>
          </li>
          <li>
            <Link to="/datenschutz">Datenschutzerklärung</Link>
          </li>
        </ul>
        <p className="legal-footer__note">
          Alle Angaben ohne Gewähr. Werkstattportal, Beispielwerkstatt
        </p>
      </div>
    </footer>
  );
}
