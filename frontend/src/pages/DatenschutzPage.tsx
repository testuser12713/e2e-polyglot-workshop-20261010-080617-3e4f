/**
 * Datenschutzerklärung (route "/datenschutz").
 *
 * Privacy notice of the workshop portal (AC-28): explains in plain language
 * which personal data is stored for what purpose and how long, that invoices
 * are kept, that no data leaves the site, and that no real e-mails are sent.
 * Rendered inside the AppShell and reachable from the footer of every page.
 */
export default function DatenschutzPage() {
  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Datenschutzerklärung</h1>
        <p className="page__subtitle">
          Informationen zur Verarbeitung personenbezogener Daten im
          Werkstattportal
        </p>
      </header>

      <div className="card">
        <div className="card__body">
          <h2>Verantwortliche Stelle</h2>
          <p>
            Beispielwerkstatt GmbH
            <br />
            Werkstattstraße 12
            <br />
            10115 Berlin
            <br />
            E-Mail: datenschutz@beispielwerkstatt.example
          </p>

          <h2>Allgemeines</h2>
          <p>
            Wir verarbeiten personenbezogene Daten nur, soweit dies für die
            Bereitstellung des Kundenportals, die Bearbeitung von
            Werkstattaufträgen und die Erfüllung gesetzlicher Pflichten
            erforderlich ist. Eine automatisierte Entscheidungsfindung
            einschließlich Profiling findet nicht statt.
          </p>

          <h2>Kundenbereich</h2>
          <p>
            Wenn Sie einen Werkstattauftrag anfragen, verarbeiten wir Ihren
            Namen, Ihre E-Mail-Adresse, Ihre Telefonnummer, das Kennzeichen,
            Fahrzeugdaten, den Wunschtermin und Ihre Problembeschreibung.
            Rechtsgrundlage ist Art. 6 Abs. 1 lit. b DSGVO (Vertragserfüllung und
            vorvertragliche Maßnahmen). Die Daten werden gelöscht, sobald sie für
            die Vertragsabwicklung nicht mehr erforderlich sind, soweit
            gesetzliche Aufbewahrungsfristen (insbesondere handels- und
            steuerrechtliche Fristen) keine längere Speicherung verlangen. Die zu
            einem Auftrag erstellte Rechnung wird für die Dauer der gesetzlichen
            Aufbewahrungsfristen gespeichert.
          </p>

          <h2>Statusabruf</h2>
          <p>
            Beim Abruf des Auftragsstatus werden Auftragsnummer und Kennzeichen
            abgeglichen. Es werden ausschließlich Daten des zugehörigen Auftrags
            zurückgegeben — keine Kontaktdaten und keine Daten anderer Kunden.
            Die Eingaben werden nicht in Klartext protokolliert.
          </p>

          <h2>Werkstattbereich</h2>
          <p>
            Mitarbeiterkonten dienen der internen Auftragsverwaltung. Passwörter
            werden ausschließlich in gehashter Form gespeichert. Anmeldeversuche
            werden zur Abwehr von Missbrauch gegen ein Zeitlimit geprüft; dabei
            fallen keine Klartextpasswörter in die Anwendung oder die
            Protokolle.
          </p>

          <h2>Benachrichtigungen</h2>
          <p>
            Nach dem Rechnungslauf wird eine Benachrichtigung in den internen
            Postausgang des Portals gestellt. Es werden über dieses Portal keine
            kommerziellen E-Mails an Dritte versendet.
          </p>

          <h2>Empfänger</h2>
          <p>
            Die im Portal gespeicherten Daten werden nur an interne Systeme der
            Werkstatt übermittelt. Eine Weitergabe an Dritte erfolgt nicht,
            außer zur Erfüllung gesetzlicher Pflichten.
          </p>

          <h2>Ihre Rechte</h2>
          <ul>
            <li>
              Auskunft über die zu Ihrer Person gespeicherten Daten (Art. 15
              DSGVO)
            </li>
            <li>Berichtigung unrichtiger Daten (Art. 16 DSGVO)</li>
            <li>
              Löschung (Art. 17 DSGVO), soweit keine gesetzlichen
              Aufbewahrungspflichten entgegenstehen
            </li>
            <li>Einschränkung der Verarbeitung (Art. 18 DSGVO)</li>
            <li>Widerspruch gegen die Verarbeitung (Art. 21 DSGVO)</li>
            <li>Datenübertragbarkeit (Art. 20 DSGVO)</li>
          </ul>
          <p>
            Zur Ausübung Ihrer Rechte genügt eine formlose Nachricht an die oben
            genannte Kontaktadresse. Sie haben zudem das Recht, sich bei einer
            Datenschutz-Aufsichtsbehörde zu beschweren.
          </p>

          <h2>Externe Ressourcen</h2>
          <p>
            Dieses Portal lädt keine Schriften, Skripte oder sonstigen Inhalte
            von fremden Domains. Alle Ressourcen stammen aus dem eigenen
            Ursprung des Portals.
          </p>
        </div>
      </div>
    </section>
  );
}
