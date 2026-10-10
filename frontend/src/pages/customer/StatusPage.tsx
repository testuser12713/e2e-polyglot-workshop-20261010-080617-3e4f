import { type FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { lookupOrderStatus } from "../../api/lookup";
import { ApiError, type Status, type StatusLookup } from "../../api/types";
import { StatusBadge } from "../../components/StatusBadge";
import { formatDateTime } from "../../lib/format";
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
  "Zu dieser Auftragsnummer wurde kein Auftrag mit dem angegebenen Kennzeichen gefunden. Prüfen Sie beide Eingaben.";

const REQUIRED_MESSAGE = "Bitte geben Sie Auftragsnummer und Kennzeichen ein.";

/** Normalise the licence plate the same way the mockup does: trim, collapse spaces, uppercase. */
function normalisePlate(value: string): string {
  return value.trim().toUpperCase().replace(/\s+/g, " ");
}

/**
 * Customer order status lookup (route "/status").
 *
 * Live status and status history are shown only when the order number and the
 * licence plate match the same order. Any non-match answers the same 404, which
 * is rendered as one friendly sentence and never as order data (AC-07 / AC-31).
 */
export default function StatusPage() {
  const [orderNumber, setOrderNumber] = useState("");
  const [licensePlate, setLicensePlate] = useState("");
  const [result, setResult] = useState<StatusLookup | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function runLookup(number: string, plate: string) {
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      const lookup = await lookupOrderStatus(number, plate);
      setResult(lookup);
    } catch (caught) {
      if (caught instanceof ApiError && caught.code === "not_found") {
        setError(NOT_FOUND_MESSAGE);
      } else if (caught instanceof ApiError) {
        setError(caught.message);
      } else {
        setError("Der Auftragsstatus konnte nicht geladen werden. Bitte versuchen Sie es erneut.");
      }
    } finally {
      setLoading(false);
    }
  }

  function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const number = orderNumber.trim().toUpperCase();
    const plate = normalisePlate(licensePlate);
    if (!number || !plate) {
      setResult(null);
      setError(REQUIRED_MESSAGE);
      return;
    }
    void runLookup(number, plate);
  }

  const showInvoiceHint =
    result != null && (result.status === "fertig" || result.status === "abgeholt");
  const plateForLink = normalisePlate(licensePlate);

  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Auftragsstatus</h1>
        <p className="page__subtitle">
          Verfolgen Sie den Fortschritt Ihres Auftrags mit Auftragsnummer und
          Kennzeichen.
        </p>
      </header>

      <section
        className="card"
        data-od-id="status-lookup-card"
        aria-labelledby="status-title"
      >
        <h2 className="card__title" id="status-title">
          Status abrufen
        </h2>

        <form
          className="card__body"
          data-od-id="status-form"
          noValidate
          onSubmit={onSubmit}
        >
          <div className="field">
            <label className="field__label" htmlFor="status-order-number">
              Auftragsnummer <span className="field__required">erforderlich</span>
            </label>
            <input
              className="field__input field__input--mono"
              id="status-order-number"
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
            <label className="field__label" htmlFor="status-license-plate">
              Kennzeichen <span className="field__required">erforderlich</span>
            </label>
            <input
              className="field__input field__input--mono"
              id="status-license-plate"
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
              data-od-id="status-submit"
              disabled={loading}
              aria-busy={loading}
            >
              {loading ? "Wird gesucht…" : "Status anzeigen"}
            </button>
          </div>
        </form>

        {error ? (
          <div
            className="alert alert--error lookup-feedback"
            role="alert"
            data-od-id="status-error"
          >
            <span>{error}</span>
          </div>
        ) : null}

        {result ? (
          <div className="lookup-result" data-od-id="status-result">
            <div className="lookup-result__head">
              <div className="lookup-result__id">
                <span className="mono lookup-result__number">
                  {result.order_number}
                </span>
                <span className="mono lookup-result__plate">
                  {plateForLink}
                </span>
              </div>
              <StatusBadge status={result.status} showDot />
            </div>

            <div className="lookup-result__history">
              <h3 className="lookup-result__history-title">Statusverlauf</h3>
              <ol className="timeline" data-od-id="status-timeline">
                {[...result.status_history].reverse().map((entry, index) => (
                  <li key={`${entry.status}-${entry.changed_at}-${index}`}>
                    <span
                      className={`timeline-node timeline-node--${NODE_MODIFIERS[entry.status] ?? "requested"}`}
                      aria-hidden="true"
                    />
                    <p className="timeline-state">
                      {STATUS_LABELS[entry.status] ?? entry.status}
                    </p>
                    <p className="timeline-meta">
                      {formatDateTime(entry.changed_at)}
                    </p>
                  </li>
                ))}
              </ol>
            </div>

            {showInvoiceHint ? (
              <div
                className="alert alert--info lookup-feedback"
                data-od-id="status-invoice-hint"
              >
                <span>
                  Der Auftrag ist fertiggestellt.{" "}
                  <Link
                    to={`/rechnung?order_number=${encodeURIComponent(
                      result.order_number,
                    )}&license_plate=${encodeURIComponent(plateForLink)}`}
                  >
                    Rechnung {result.order_number} ansehen
                  </Link>
                </span>
              </div>
            ) : null}
          </div>
        ) : null}
      </section>
    </section>
  );
}
