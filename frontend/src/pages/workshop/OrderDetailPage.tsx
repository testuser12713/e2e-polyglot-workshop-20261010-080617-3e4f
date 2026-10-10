import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { useParams } from "react-router-dom";
import { StatusBadge } from "../../components/StatusBadge";
import { ErrorBanner } from "../../components/ErrorBanner";
import { useAuth } from "../../auth/AuthContext";
import {
  ApiError,
  type InvoiceItem,
  type OrderDetail,
  type Status,
} from "../../api/types";
import {
  addWorkshopOrderItems,
  confirmWorkshopOrder,
  getWorkshopOrder,
  setWorkshopOrderStatus,
  type OrderItemInput,
} from "../../api/workshopOrder";
import "./OrderDetailPage.css";

const STATUS_ORDER: Status[] = [
  "angefragt",
  "bestaetigt",
  "in_arbeit",
  "fertig",
  "abgeholt",
];

const STATUS_LABEL: Record<Status, string> = {
  angefragt: "angefragt",
  bestaetigt: "bestätigt",
  in_arbeit: "in Arbeit",
  fertig: "fertig",
  abgeholt: "abgeholt",
};

const STATUS_NODE: Record<Status, string> = {
  angefragt: "requested",
  bestaetigt: "confirmed",
  in_arbeit: "inprogress",
  fertig: "done",
  abgeholt: "pickedup",
};

/** The single next lifecycle state, or null for the terminal state. */
function nextStatus(status: Status): Status | null {
  const index = STATUS_ORDER.indexOf(status);
  if (index < 0 || index + 1 >= STATUS_ORDER.length) return null;
  return STATUS_ORDER[index + 1];
}

function toApiError(error: unknown): ApiError {
  return error instanceof ApiError ? error : new ApiError("internal_error");
}

/** Integer cents rendered as German currency, thousands dot, comma decimals. */
function formatMoney(cents: number): string {
  const sign = cents < 0 ? "-" : "";
  const abs = Math.abs(Math.round(cents));
  const euros = Math.floor(abs / 100);
  const rest = String(abs % 100).padStart(2, "0");
  const grouped = String(euros).replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  return `${sign}${grouped},${rest}\u00A0€`;
}

/** German duration from a whole number of minutes, e.g. "1 h 30 min". */
function formatDuration(minutes: number): string {
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (hours === 0) return `${rest} min`;
  if (rest === 0) return `${hours} h`;
  return `${hours} h ${rest} min`;
}

function formatDateOnly(value: string): string {
  const [year, month, day] = value.split("-");
  if (!year || !month || !day) return value;
  return `${day}.${month}.${year}`;
}

function formatDateTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const hours = String(date.getHours()).padStart(2, "0");
  const minutes = String(date.getMinutes()).padStart(2, "0");
  return `${day}.${month}.${date.getFullYear()}, ${hours}:${minutes} Uhr`;
}

/** Parse a German decimal ("1,5" or "1.5") into a number, or null. */
function parseDecimal(raw: string): number | null {
  const cleaned = raw.trim().replace(/\./g, "").replace(",", ".");
  if (!cleaned) return null;
  const value = Number(cleaned);
  return Number.isFinite(value) ? value : null;
}

/** Euro amount as an integer number of cents, or null when invalid. */
function parseEuroToCents(raw: string): number | null {
  const value = parseDecimal(raw);
  if (value == null || value < 0) return null;
  return Math.round(value * 100);
}

function parseCount(raw: string): number | null {
  const cleaned = raw.trim();
  if (!/^\d+$/.test(cleaned)) return null;
  return Number(cleaned);
}

function itemQuantityLabel(item: InvoiceItem): string {
  if (item.kind === "labor") return formatDuration(item.quantity);
  return String(item.quantity);
}

function itemDescriptionLabel(item: InvoiceItem): string {
  const prefix = item.kind === "labor" ? "Arbeitszeit" : "Einzelteil";
  return `${prefix} — ${item.description}`;
}

interface ModalProps {
  title: string;
  odId?: string;
  wide?: boolean;
  onClose: () => void;
  children: ReactNode;
}

function Modal({ title, odId, wide = false, onClose, children }: ModalProps) {
  const titleId = useId();
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  useEffect(() => {
    const focusable = panelRef.current?.querySelector<HTMLElement>(
      "input, select, textarea, button",
    );
    focusable?.focus();
  }, []);

  return (
    <div
      className="od-modal-overlay"
      onClick={(event) => {
        if (event.target === event.currentTarget) onClose();
      }}
    >
      <div
        ref={panelRef}
        className={`od-modal${wide ? " od-modal--wide" : ""}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        data-od-id={odId}
      >
        <div className="od-modal__header">
          <h2 id={titleId}>{title}</h2>
          <button
            type="button"
            className="od-modal__close"
            aria-label="Schließen"
            onClick={onClose}
          >
            ×
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}

interface PositionsFormProps {
  busy: boolean;
  error: ApiError | string | null;
  onSubmit: (item: OrderItemInput) => void;
  onCancel: () => void;
}

interface PositionFieldErrors {
  description?: string;
  quantity?: string;
  price?: string;
}

function PositionsForm({ busy, error, onSubmit, onCancel }: PositionsFormProps) {
  const [kind, setKind] = useState<"labor" | "part">("labor");
  const [description, setDescription] = useState("");
  const [duration, setDuration] = useState("");
  const [count, setCount] = useState("");
  const [price, setPrice] = useState("");
  const [errors, setErrors] = useState<PositionFieldErrors>({});

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const next: PositionFieldErrors = {};
    const trimmed = description.trim();
    if (!trimmed) next.description = "Bitte eine Beschreibung angeben.";

    let quantity = 0;
    if (kind === "labor") {
      const hours = parseDecimal(duration);
      if (hours == null || hours <= 0) {
        next.quantity = "Bitte eine gültige Dauer angeben, z. B. 1,5.";
      } else {
        quantity = Math.round(hours * 60);
      }
    } else {
      const value = parseCount(count);
      if (value == null || value <= 0) {
        next.quantity = "Bitte eine ganze Anzahl größer 0 angeben.";
      } else {
        quantity = value;
      }
    }

    const priceCents = parseEuroToCents(price);
    if (priceCents == null) {
      next.price = "Bitte einen gültigen Betrag angeben, z. B. 95,00.";
    }

    setErrors(next);
    if (next.description || next.quantity || next.price) return;

    onSubmit({
      kind,
      description: trimmed,
      quantity,
      unit_price_cents: priceCents as number,
    });
  }

  return (
    <form
      className="od-form"
      onSubmit={handleSubmit}
      noValidate
      data-od-id="positions-form"
    >
      {error ? <ErrorBanner error={error} /> : null}
      <div className="od-form__fields">
        <div className="field">
          <label className="field__label" htmlFor="od-pos-kind">
            Typ
          </label>
          <select
            id="od-pos-kind"
            className="od-select"
            value={kind}
            onChange={(event) => {
              setKind(event.target.value as "labor" | "part");
              setErrors({});
            }}
          >
            <option value="labor">Arbeitszeit</option>
            <option value="part">Einzelteil</option>
          </select>
        </div>

        <div className={`field${errors.description ? " field--error" : ""}`}>
          <label className="field__label" htmlFor="od-pos-description">
            Beschreibung <span className="field__required">erforderlich</span>
          </label>
          <input
            id="od-pos-description"
            className="field__input"
            type="text"
            placeholder="z. B. Bremsbeläge vorne"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
          />
          <div className="field__help">{errors.description ?? ""}</div>
        </div>

        {kind === "labor" ? (
          <div className={`field${errors.quantity ? " field--error" : ""}`}>
            <label className="field__label" htmlFor="od-pos-duration">
              Dauer (Stunden){" "}
              <span className="field__required">erforderlich</span>
            </label>
            <input
              id="od-pos-duration"
              className="field__input field__input--numeric"
              type="text"
              inputMode="decimal"
              placeholder="1,5"
              value={duration}
              onChange={(event) => setDuration(event.target.value)}
            />
            <div className="field__help">
              {errors.quantity ?? "Dezimalzahl mit Komma, z. B. 1,5 für 1 h 30 min"}
            </div>
          </div>
        ) : (
          <div className={`field${errors.quantity ? " field--error" : ""}`}>
            <label className="field__label" htmlFor="od-pos-count">
              Anzahl <span className="field__required">erforderlich</span>
            </label>
            <input
              id="od-pos-count"
              className="field__input field__input--numeric"
              type="text"
              inputMode="numeric"
              placeholder="1"
              value={count}
              onChange={(event) => setCount(event.target.value)}
            />
            <div className="field__help">
              {errors.quantity ?? "Ganze Zahl"}
            </div>
          </div>
        )}

        <div className={`field${errors.price ? " field--error" : ""}`}>
          <label className="field__label" htmlFor="od-pos-price">
            Einzelpreis netto (€){" "}
            <span className="field__required">erforderlich</span>
          </label>
          <input
            id="od-pos-price"
            className="field__input field__input--numeric"
            type="text"
            inputMode="decimal"
            placeholder="95,00"
            value={price}
            onChange={(event) => setPrice(event.target.value)}
          />
          <div className="field__help">
            {errors.price ??
              (kind === "labor"
                ? "Stundensatz für die Arbeitszeit"
                : "Preis pro Einzelteil")}
          </div>
        </div>
      </div>
      <div className="od-form__footer">
        <button type="button" className="btn btn--secondary" onClick={onCancel}>
          Abbrechen
        </button>
        <button type="submit" className="btn btn--primary" disabled={busy}>
          Position hinzufügen
        </button>
      </div>
    </form>
  );
}

interface StatusModalProps {
  orderNumber: string;
  target: Status;
  busy: boolean;
  error: ApiError | string | null;
  onConfirm: () => void;
  onClose: () => void;
}

function StatusModal({
  orderNumber,
  target,
  busy,
  error,
  onConfirm,
  onClose,
}: StatusModalProps) {
  return (
    <>
      <p className="od-confirm-text">
        Auftrag <span className="mono">{orderNumber}</span> — nächster erlaubter
        Status: <strong>{STATUS_LABEL[target]}</strong>.
      </p>
      {error ? <ErrorBanner error={error} /> : null}
      <form
        className="od-form"
        onSubmit={(event) => {
          event.preventDefault();
          onConfirm();
        }}
      >
        <div className="field">
          <label className="field__label" htmlFor="od-status-target">
            Neuer Status
          </label>
          <select id="od-status-target" className="od-select" value={target} disabled>
            <option value={target}>{STATUS_LABEL[target]}</option>
          </select>
          <div className="field__help">
            Es ist nur der nächste Schritt entlang des Auftragslebenszyklus
            möglich; Sprünge und Rückwärtswechsel lehnt die API ab.
          </div>
        </div>
        <div className="od-form__footer">
          <button type="button" className="btn btn--secondary" onClick={onClose}>
            Abbrechen
          </button>
          <button type="submit" className="btn btn--primary" disabled={busy}>
            Auf {orderNumber} auf „{STATUS_LABEL[target]}“ setzen
          </button>
        </div>
      </form>
    </>
  );
}

/**
 * Workshop order detail (route "/werkstatt/auftraege/:orderNumber").
 *
 * Loads the order through GET /api/workshop/orders/{order_number}, offers the
 * confirm action while the order is "angefragt", lets the employee capture
 * labor/part positions and advances the status one legal step at a time. Every
 * successful action refreshes the order so positions and history stay current.
 */
export default function OrderDetailPage() {
  const { orderNumber } = useParams<{ orderNumber: string }>();
  const { token } = useAuth();

  const [order, setOrder] = useState<OrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<ApiError | string | null>(null);

  const [busy, setBusy] = useState(false);
  const [actionError, setActionError] = useState<ApiError | string | null>(null);
  const [formError, setFormError] = useState<ApiError | string | null>(null);
  const [statusError, setStatusError] = useState<ApiError | string | null>(null);

  const [positionsOpen, setPositionsOpen] = useState(false);
  const [statusOpen, setStatusOpen] = useState(false);
  const [toast, setToast] = useState<string | null>(null);

  const loadOrder = useCallback(async () => {
    if (!orderNumber || !token) return;
    setLoading(true);
    setLoadError(null);
    try {
      const detail = await getWorkshopOrder(orderNumber, token);
      setOrder(detail);
    } catch (error) {
      setLoadError(toApiError(error));
    } finally {
      setLoading(false);
    }
  }, [orderNumber, token]);

  useEffect(() => {
    void loadOrder();
  }, [loadOrder]);

  useEffect(() => {
    if (!toast) return;
    const timer = window.setTimeout(() => setToast(null), 6000);
    return () => window.clearTimeout(timer);
  }, [toast]);

  const handleConfirm = useCallback(async () => {
    if (!orderNumber || !token) return;
    setBusy(true);
    setActionError(null);
    try {
      await confirmWorkshopOrder(orderNumber, token);
      setToast(`Auftrag ${orderNumber} wurde bestätigt.`);
      await loadOrder();
    } catch (error) {
      setActionError(toApiError(error));
    } finally {
      setBusy(false);
    }
  }, [orderNumber, token, loadOrder]);

  const handleAddPosition = useCallback(
    async (item: OrderItemInput) => {
      if (!orderNumber || !token) return;
      setBusy(true);
      setFormError(null);
      try {
        await addWorkshopOrderItems(orderNumber, [item], token);
        setPositionsOpen(false);
        setToast("Position wurde erfasst.");
        await loadOrder();
      } catch (error) {
        setFormError(toApiError(error));
      } finally {
        setBusy(false);
      }
    },
    [orderNumber, token, loadOrder],
  );

  const handleSetStatus = useCallback(
    async (target: Status) => {
      if (!orderNumber || !token) return;
      setBusy(true);
      setStatusError(null);
      try {
        await setWorkshopOrderStatus(orderNumber, target, token);
        setStatusOpen(false);
        setToast(
          `Auftrag ${orderNumber} ist jetzt „${STATUS_LABEL[target]}“.`,
        );
        await loadOrder();
      } catch (error) {
        setStatusError(toApiError(error));
      } finally {
        setBusy(false);
      }
    },
    [orderNumber, token, loadOrder],
  );

  if (loading && !order) {
    return (
      <section className="page">
        <header className="page__header">
          <h1>
            Auftrag <span className="mono">{orderNumber}</span>
          </h1>
        </header>
        <div className="card">
          <div className="card__body">
            <p className="page__subtitle">Auftrag wird geladen…</p>
          </div>
        </div>
      </section>
    );
  }

  if (loadError && !order) {
    return (
      <section className="page">
        <header className="page__header">
          <h1>
            Auftrag <span className="mono">{orderNumber}</span>
          </h1>
        </header>
        <div className="card">
          <div className="card__body">
            <ErrorBanner error={loadError} />
            <button
              type="button"
              className="btn btn--secondary"
              onClick={() => void loadOrder()}
            >
              Erneut versuchen
            </button>
          </div>
        </div>
      </section>
    );
  }

  if (!order) {
    return null;
  }

  const next = nextStatus(order.status);
  const netCents = order.items.reduce((sum, item) => sum + item.total_cents, 0);
  const vatCents = Math.round((netCents * 19) / 100);
  const grossCents = netCents + vatCents;
  const history = [...order.status_history].reverse();
  const positionsAllowed = order.status !== "abgeholt";

  return (
    <section className="page order-detail" data-od-id="auftrag-main">
      <div className="order-detail__head">
        <div className="page__header">
          <h1>
            Auftrag <span className="mono">{order.order_number}</span>
          </h1>
          <p className="page__subtitle">
            {order.preferred_date
              ? `Wunschtermin ${formatDateOnly(order.preferred_date)}`
              : "Kein Wunschtermin angegeben"}
          </p>
        </div>
        <StatusBadge status={order.status} showDot />
      </div>

      <div className="od-grid">
        <div className="od-col">
          <section
            className="card"
            data-od-id="auftrag-info"
            aria-labelledby="od-info-title"
          >
            <h2 className="od-section__title" id="od-info-title">
              Fahrzeug und Kunde
            </h2>
            <dl className="od-detail-grid">
              <div className="od-detail-item">
                <dt>Kennzeichen</dt>
                <dd className="mono">{order.vehicle.license_plate}</dd>
              </div>
              <div className="od-detail-item">
                <dt>Fahrzeug</dt>
                <dd>
                  {order.vehicle.make} {order.vehicle.model}
                </dd>
              </div>
              <div className="od-detail-item">
                <dt>Kilometerstand</dt>
                <dd className="tabular">
                  {order.vehicle.mileage_km.toLocaleString("de-DE")} km
                </dd>
              </div>
              <div className="od-detail-item">
                <dt>Kunde</dt>
                <dd>{order.customer.name}</dd>
              </div>
              <div className="od-detail-item">
                <dt>E-Mail</dt>
                <dd>{order.customer.email}</dd>
              </div>
              <div className="od-detail-item">
                <dt>Telefon</dt>
                <dd className="tabular">{order.customer.phone}</dd>
              </div>
            </dl>
            <p className="od-problem-label">Problembeschreibung</p>
            <p className="od-problem-text">{order.problem_description}</p>
          </section>

          <section
            className="card"
            data-od-id="auftrag-positionen"
            aria-labelledby="od-positions-title"
          >
            <div className="od-section__header">
              <h2 className="od-section__title" id="od-positions-title">
                Erfasste Positionen
              </h2>
              <button
                type="button"
                className="btn btn--secondary"
                onClick={() => setPositionsOpen(true)}
                disabled={busy || !positionsAllowed}
              >
                Positionen erfassen
              </button>
            </div>

            {order.items.length === 0 ? (
              <div className="empty-state" data-od-id="positions-empty">
                <svg
                  className="empty-state__icon"
                  viewBox="0 0 48 48"
                  fill="none"
                  aria-hidden="true"
                >
                  <rect
                    x="10"
                    y="8"
                    width="28"
                    height="32"
                    rx="3"
                    stroke="currentColor"
                    strokeWidth="2"
                  />
                  <path
                    d="M16 16h16M16 22h16M16 28h10"
                    stroke="currentColor"
                    strokeWidth="2"
                    strokeLinecap="round"
                  />
                </svg>
                <h3 className="empty-state__title">
                  Noch keine Positionen erfasst
                </h3>
                <p className="empty-state__text">
                  Erfassen Sie Arbeitszeit und Teile, bevor Sie den Auftrag
                  fertigstellen. Die Rechnung wird danach automatisch erstellt.
                </p>
              </div>
            ) : (
              <div data-od-id="positions-content">
                <table className="od-table" data-od-id="positions-table">
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
                    {order.items.map((item, index) => (
                      <tr key={`${item.kind}-${index}`}>
                        <td>{itemDescriptionLabel(item)}</td>
                        <td className="right tabular">
                          {itemQuantityLabel(item)}
                        </td>
                        <td className="right tabular">
                          {formatMoney(item.unit_price_cents)}
                        </td>
                        <td className="right tabular">
                          {formatMoney(item.total_cents)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                <div className="od-totals" data-od-id="positions-totals">
                  <div className="od-totals__row">
                    <span>Nettobetrag</span>
                    <span className="tabular">{formatMoney(netCents)}</span>
                  </div>
                  <div className="od-totals__row">
                    <span>Mehrwertsteuer (19 %)</span>
                    <span className="tabular">{formatMoney(vatCents)}</span>
                  </div>
                  <div className="od-totals__row od-totals__row--grand">
                    <span>Gesamtbetrag</span>
                    <span className="tabular">{formatMoney(grossCents)}</span>
                  </div>
                </div>
              </div>
            )}
          </section>

          <section
            className="card"
            data-od-id="auftrag-timeline"
            aria-labelledby="od-timeline-title"
          >
            <h2 className="od-section__title" id="od-timeline-title">
              Statusverlauf
            </h2>
            {history.length === 0 ? (
              <p className="od-note">Noch keine Statusereignisse erfasst.</p>
            ) : (
              <ol className="od-timeline">
                {history.map((entry, index) => (
                  <li
                    className="od-timeline__item"
                    key={`${entry.status}-${entry.changed_at}-${index}`}
                  >
                    <span
                      className={`od-timeline__node od-timeline__node--${STATUS_NODE[entry.status]}`}
                      aria-hidden="true"
                    />
                    <p className="od-timeline__state">
                      {STATUS_LABEL[entry.status]}
                    </p>
                    <p className="od-timeline__meta">
                      {formatDateTime(entry.changed_at)}
                    </p>
                  </li>
                ))}
              </ol>
            )}
            <p className="od-note">
              {next ? (
                <>
                  Nächster erlaubter Schritt:{" "}
                  <strong>{STATUS_LABEL[next]}</strong>. Sprünge und
                  Rückwärtswechsel werden abgelehnt.
                </>
              ) : (
                "Der Auftrag ist abgeholt; es sind keine weiteren Statuswechsel möglich."
              )}
            </p>
          </section>
        </div>

        <aside className="od-col">
          <section
            className="card"
            data-od-id="auftrag-actions"
            aria-labelledby="od-actions-title"
          >
            <h2 className="od-section__title" id="od-actions-title">
              Aktionen
            </h2>
            {actionError ? <ErrorBanner error={actionError} /> : null}
            <div className="od-actions">
              {order.status === "angefragt" ? (
                <button
                  type="button"
                  className="btn btn--primary"
                  onClick={() => void handleConfirm()}
                  disabled={busy}
                >
                  Auftrag bestätigen
                </button>
              ) : null}
              {order.status !== "angefragt" && next ? (
                <button
                  type="button"
                  className="btn btn--primary"
                  onClick={() => setStatusOpen(true)}
                  disabled={busy}
                >
                  Status setzen
                </button>
              ) : null}
              <button
                type="button"
                className="btn btn--secondary"
                onClick={() => setPositionsOpen(true)}
                disabled={busy || !positionsAllowed}
              >
                Positionen erfassen
              </button>
              <button
                type="button"
                className="btn btn--secondary"
                disabled
                aria-disabled="true"
                title="kommt bald"
              >
                Rechnung anzeigen
                <span className="od-coming-soon">kommt bald</span>
              </button>
            </div>
            <p className="od-note">
              Der Auftrag ist <strong>{STATUS_LABEL[order.status]}</strong>.
              Fertigstellen erst, wenn alle Positionen erfasst sind.
            </p>
          </section>

          <section
            className="card"
            data-od-id="auftrag-rechnung-hint"
            aria-labelledby="od-rechnung-title"
          >
            <h2 className="od-section__title" id="od-rechnung-title">
              Rechnung
            </h2>
            <p className="page__subtitle">
              Die Rechnung wird erstellt, sobald der Auftrag auf{" "}
              <strong>fertig</strong> gesetzt ist. Ein zweites Fertigstellen
              erzeugt keine zweite Rechnung.
            </p>
          </section>
        </aside>
      </div>

      {positionsOpen ? (
        <Modal
          title="Positionen erfassen"
          odId="positions-modal"
          wide
          onClose={() => {
            setPositionsOpen(false);
            setFormError(null);
          }}
        >
          <PositionsForm
            busy={busy}
            error={formError}
            onSubmit={(item) => void handleAddPosition(item)}
            onCancel={() => {
              setPositionsOpen(false);
              setFormError(null);
            }}
          />
        </Modal>
      ) : null}

      {statusOpen && next ? (
        <Modal
          title="Status setzen"
          odId="status-modal"
          onClose={() => {
            setStatusOpen(false);
            setStatusError(null);
          }}
        >
          <StatusModal
            orderNumber={order.order_number}
            target={next}
            busy={busy}
            error={statusError}
            onConfirm={() => void handleSetStatus(next)}
            onClose={() => {
              setStatusOpen(false);
              setStatusError(null);
            }}
          />
        </Modal>
      ) : null}

      <div className="od-toast-region" aria-live="polite">
        {toast ? (
          <div className="od-toast" role="status">
            {toast}
          </div>
        ) : null}
      </div>
    </section>
  );
}
