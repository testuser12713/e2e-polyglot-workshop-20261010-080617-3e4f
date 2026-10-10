/**
 * Impressum (route "/impressum").
 *
 * Legal notice of the workshop portal (AC-28): names the provider, the person
 * who represents it and the contact details. Rendered inside the AppShell and
 * reachable from the footer of every page in both areas.
 */
export default function ImpressumPage() {
  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Impressum</h1>
        <p className="page__subtitle">Angaben gemäß § 5 DDG (ehem. § 5 TMG)</p>
      </header>

      <div className="card">
        <div className="card__body">
          <h2>Anbieter</h2>
          <p>
            Beispielwerkstatt GmbH
            <br />
            Werkstattstraße 12
            <br />
            10115 Berlin
          </p>

          <h2>Vertreten durch</h2>
          <p>Geschäftsführung: Anna Beispiel</p>

          <h2>Kontakt</h2>
          <p>
            Telefon: 030 1234567
            <br />
            E-Mail: info@beispielwerkstatt.example
          </p>

          <h2>Registereintrag</h2>
          <p>
            Eintragung im Handelsregister
            <br />
            Registergericht: Amtsgericht Berlin-Charlottenburg
            <br />
            Registernummer: HRB 12345 B
          </p>

          <h2>Umsatzsteuer-Identifikationsnummer</h2>
          <p>DE 123 456 789</p>

          <h2>Verbraucherstreitbeilegung</h2>
          <p>
            Die Europäische Kommission stellt eine Plattform zur
            Online-Streitbeilegung bereit. Wir sind nicht verpflichtet und nicht
            bereit, an einem Streitbeilegungsverfahren vor einer
            Verbraucherschlichtungsstelle teilzunehmen.
          </p>

          <h2>Haftung für Inhalte und Links</h2>
          <p>
            Als Diensteanbieter sind wir für eigene Inhalte auf diesen Seiten nach
            den allgemeinen Gesetzen verantwortlich. Unser Angebot enthält Links
            zu externen Websites Dritter, auf deren Inhalte wir keinen Einfluss
            haben. Für die Inhalte der verlinkten Seiten ist stets der jeweilige
            Anbieter verantwortlich.
          </p>
        </div>
      </div>
    </section>
  );
}
