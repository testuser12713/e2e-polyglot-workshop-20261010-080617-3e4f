import { type FormEvent, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { fetchInvoice, lookupOrderStatus } from "../../api/lookup";
import {
  ApiError,
  type Invoice,
  type Status,
  type StatusLookup,
} from "../../api/types";
import { StatusBadge } from "../../components/StatusBadge";
import {
  formatDateTime,
  formatEuroCents,
  formatInvoiceQuantity,
} from "../../lib/format";
import "./customer.css";

const STATUS_LABELS: Record<Status, string> = {
  angefragt: "Angefragt",
  bestaetigt: "Bestätigt",
  in_arbeit: "In Arbeit",
  fertig: "Fertig",
  abgeholt: "Abgeholt",
};

const NODE_MODIFIERS: Record<Status, string> = {
  angefragt: "requested",
  bestaetigt: "confirmed",
  in_arbeit: "inprogress",
  fertig: "done",
  abgeholt: "pickedup",
};

const NOT_FOUND_MESSAGE =
  "Zu diesen Angaben wurde keine Rechnung gefunden. Prüfen Sie Auftragsnummer und Kennzeichen.";

const REQUIRED_MESSAGE = "Bitte geben Sie Auftragsnummer und Kennzeichen ein.";

/** Normalise the licence plate: trim, collapse spaces, uppercase. */
function normalisePlate(value: string): string {
  return value.trim().toUpperCase().replace(/\s+/g, " ");
}

function timelineNodes(history: StatusLookup["status_history"]) {
  return [...history].reverse().map((entry, index) => (
    <li key={`${entry.status}-${entry.changed_at}-${index}`}>
      <span
        className={`timeline-node timeline-node--${NODE_MODIFIERS[entry.status] ?? "requested"}`}
        aria-hidden="true"
      />
      <p className="timeline-state">
        {STATUS_LABELS[entry.status] ?? entry.status}
      </p>
      <p className="timeline-meta">{formatDateTime(entry.changed_at)}</p>
    </li>
  ));
}

/**
 * Customer invoice view (route "/rechnung").
 *
 * Loads GET /api/orders/{order_number}/invoice and shows the positions, the net
 * amount, the 19 % VAT and the gross total, every amount rendered from integer
 * cents. The page offers no download and no payment — only the note that the
 * invoice is visible here (AC-12).
 */
export default function InvoicePage() {
  const [searchParams] = useSearchParams();
  const initialNumber = searchParams.get("order_number") ?? "";
  const initialPlate = searchParams.get("license_plate") ?? "";

  const [orderNumber, setOrderNumber] = useState(initialNumber);
  const [licensePlate, setLicensePlate] = useState(initialPlate);
  const [invoice, setInvoice] = useState<Invoice | null>(null);
  const [status, setStatus] = useState<StatusLookup | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const autoLoaded = useRef(false);

  async function load(number: string, plate: string) {
    setLoading(true);
    setError(null);
    setInvoice(null);
    setStatus(null);
    try {
      const loaded = await fetchInvoice(number, plate);
      setInvoice(loaded);
      try {
        setStatus(await lookupOrderStatus(number, plate));
      } catch {
        setStatus(null);
      }
    } catch (caught) {
      if (caught instanceof ApiError && caught.code === "not_found") {
        setError(NOT_FOUND_MESSAGE);
      } else if (caught instanceof ApiError) {
        setError(caught.message);
      } else {
        setError(
          "Die Rechnung konnte nicht geladen werden. Bitte versuchen Sie es erneut.",
        );
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (autoLoaded.current) return;
    autoLoaded.current = true;
    const number = initialNumber.trim().toUpperCase();
    const plate = normalisePlate(initialPlate);
    if (number && plate) void load(number, plate);
    // Deliberately run once on first mount from the URL query.
  }, []);

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const number = orderNumber.trim().toUpperCase();
    const plate = normalisePlate(licensePlate);
    if (!number || !plate) {
      setInvoice(null);
      setStatus(null);
      setError(REQUIRED_MESSAGE);
      return;
    }
    void load(number, plate);
  }

  const plateForLink = normalisePlate(licensePlate);

  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Rechnung</h1>
        {invoice ? (
          <p className="page__subtitle">
            Auftrag <span className="mono">{invoice.order_number}</span> ·
            Kennzeichen <span className="mono">{plateForLink}</span>
            {status ? (
              <>
                {" "}
                · Status <StatusBadge status={status.status} showDot />
              </>
            ) : null}
          </p>
        ) : (
          <p className="page__subtitle">
            Sehen Sie die fertige Rechnung zu Ihrem Auftrag ein.
          </p>
        )}
      </header>

      {invoice === null ? (
        <section
          className="card"
          data-od-id="invoice-lookup-card"
          aria-labelledby="invoice-lookup-title"
        >
          <h2 className="card__title" id="invoice-lookup-title">
            Rechnung abrufen
          </h2>
          <form
            className="card__body"
            data-od-id="invoice-lookup-form"
            noValidate
            onSubmit={onSubmit}
          >
            <div className="field">
              <label className="field__label" htmlFor="invoice-order-number">
                Auftragsnummer{" "}
                <span className="field__required">erforderlich</span>
              </label>
              <input
                className="field__input field__input--mono"
                id="invoice-order-number"
                name="order_number"
                type="text"
                placeholder="AU-2025-00123"
                autoComplete="off"
                value={orderNumber}
                onChange={(event) => setOrderNumber(event.target.value)}
              />
              <p className="field__help">Format AU-JJJJ-NNNNN</p>
            </div>

            <div className="field">
              <label className="field__label" htmlFor="invoice-license-plate">
                Kennzeichen{" "}
                <span className="field__required">erforderlich</span>
              </label>
              <input
                className="field__input field__input--mono"
                id="invoice-license-plate"
                name="license_plate"
                type="text"
                placeholder="B-AB 1234"
                autoComplete="off"
                value={licensePlate}
                onChange={(event) => setLicensePlate(event.target.value)}
              />
              <p className="field__help" />
            </div>

            <div className="btn-row">
              <button
                className="btn btn--primary btn--block-mobile"
                type="submit"
                data-od-id="invoice-submit"
                disabled={loading}
                aria-busy={loading}
              >
                {loading ? "Wird geladen…" : "Rechnung anzeigen"}
              </button>
            </div>
          </form>

          {error ? (
            <div
              className="alert alert--error lookup-feedback"
              role="alert"
              data-od-id="invoice-error"
            >
              <span>{error}</span>
            </div>
          ) : null}
        </section>
      ) : (
        <>
          <section
            className="card"
            data-od-id="invoice-card"
            aria-labelledby="invoice-title"
          >
            <div className="card__header">
              <h2 className="card__title" id="invoice-title">
                Rechnungspositionen
              </h2>
              <span className="mono lookup-result__plate">
                Auftrag {invoice.order_number}
              </span>
            </div>

            {invoice.items.length === 0 ? (
              <p className="invoice-note">
                Für diesen Auftrag sind keine Positionen erfasst.
              </p>
            ) : (
              <table className="invoice-lines" data-od-id="invoice-lines">
                <thead>
                  <tr>
                    <th scope="col">Beschreibung</th>
                    <th scope="col" className="right">
                      Menge
                    </th>
                    <th scope="col" className="right">
                      Einzelpreis
                    </th>
                    <th scope="col" className="right">
                      Betrag
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {invoice.items.map((item, index) => (
                    <tr key={`${item.kind}-${index}`}>
                      <td>{item.description}</td>
                      <td className="right">
                        {formatInvoiceQuantity(item.kind, item.quantity)}
                      </td>
                      <td className="right">
                        {formatEuroCents(item.unit_price_cents)}
                      </td>
                      <td className="right">
                        {formatEuroCents(item.total_cents)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}

            <div className="invoice-totals" data-od-id="invoice-totals">
              <div className="invoice-totals-row">
                <span>Nettobetrag</span>
                <span>{formatEuroCents(invoice.net_cents)}</span>
              </div>
              <div className="invoice-totals-row">
                <span>Mehrwertsteuer (19 %)</span>
                <span>{formatEuroCents(invoice.vat_cents)}</span>
              </div>
              <div className="invoice-totals-row invoice-totals-row--grand">
                <span>Gesamtbetrag</span>
                <span>{formatEuroCents(invoice.gross_cents)}</span>
              </div>
            </div>

            <p className="invoice-note">
              Die Rechnung ist hier nur einzusehen. Ein Download und eine
              Online-Zahlung sind derzeit nicht vorgesehen.
            </p>
          </section>

          {status && status.status_history.length > 0 ? (
            <section
              className="section"
              data-od-id="rechnung-timeline-section"
              aria-labelledby="rechnung-timeline-title"
            >
              <h3
                className="lookup-result__history-title"
                id="rechnung-timeline-title"
              >
                Statusverlauf des Auftrags
              </h3>
              <div className="card">
                <ol className="timeline" data-od-id="rechnung-timeline">
                  {timelineNodes(status.status_history)}
                </ol>
              </div>
            </section>
          ) : null}

          <div className="btn-row">
            <Link className="btn btn--secondary" to="/status">
              Zurück zum Statusabruf
            </Link>
            <Link className="btn btn--ghost" to="/anfrage">
              Weiteren Auftrag anfragen
            </Link>
          </div>
        </>
      )}
    </section>
  );
}
