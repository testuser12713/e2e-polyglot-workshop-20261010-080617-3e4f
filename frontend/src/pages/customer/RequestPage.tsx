import { useRef, useState, type FormEvent, type RefObject } from "react";
import { Link } from "react-router-dom";
import { ApiError } from "../../api/types";
import { createOrder } from "../../api/orders";
import { ErrorBanner } from "../../components/ErrorBanner";

/* ------------------------------------------------------------------ */
/* Field model & validation                                            */
/* ------------------------------------------------------------------ */

type FieldName =
  | "license_plate"
  | "make"
  | "model"
  | "mileage_km"
  | "preferred_date"
  | "problem_description"
  | "name"
  | "email"
  | "phone";

const FIELD_IDS: Record<FieldName, string> = {
  license_plate: "kfz-kennzeichen",
  make: "kfz-marke",
  model: "kfz-modell",
  mileage_km: "kfz-km",
  preferred_date: "kfz-termin",
  problem_description: "kfz-problem",
  name: "k-name",
  email: "k-email",
  phone: "k-tel",
};

interface FormValues {
  license_plate: string;
  make: string;
  model: string;
  mileage_km: string;
  preferred_date: string;
  problem_description: string;
  name: string;
  email: string;
  phone: string;
}

const EMPTY_VALUES: FormValues = {
  license_plate: "",
  make: "",
  model: "",
  mileage_km: "",
  preferred_date: "",
  problem_description: "",
  name: "",
  email: "",
  phone: "",
};

const PLATE_PATTERN = /^[A-ZÄÖÜ]-[A-ZÄÖÜ]{1,3}\s?\d{1,4}[EH]?$/;
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

const REQUIRED_MESSAGES: Record<FieldName, string> = {
  license_plate: "Kennzeichen ist erforderlich.",
  make: "Marke ist erforderlich.",
  model: "Modell ist erforderlich.",
  mileage_km: "Kilometerstand ist erforderlich.",
  preferred_date: "Wunschtermin ist erforderlich.",
  problem_description: "Bitte beschreiben Sie Ihr Anliegen.",
  name: "Name ist erforderlich.",
  email: "E-Mail ist erforderlich.",
  phone: "Telefon ist erforderlich.",
};

/** Mirror the server's size and format rules (DESIGN.md, Form & Validation). */
function validate(values: FormValues): Partial<Record<FieldName, string>> {
  const errors: Partial<Record<FieldName, string>> = {};

  if (!values.license_plate.trim()) {
    errors.license_plate = REQUIRED_MESSAGES.license_plate;
  } else if (!PLATE_PATTERN.test(values.license_plate.trim().toUpperCase())) {
    errors.license_plate = "Bitte im Format B-AB 1234 eingeben.";
  }

  if (!values.make.trim()) errors.make = REQUIRED_MESSAGES.make;
  if (!values.model.trim()) errors.model = REQUIRED_MESSAGES.model;

  if (!values.mileage_km.trim()) {
    errors.mileage_km = REQUIRED_MESSAGES.mileage_km;
  } else if (!/^\d+$/.test(values.mileage_km.replace(/\./g, "").trim())) {
    errors.mileage_km = "Bitte eine ganze Zahl eingeben.";
  }

  if (!values.preferred_date) {
    errors.preferred_date = REQUIRED_MESSAGES.preferred_date;
  }

  if (!values.problem_description.trim()) {
    errors.problem_description = REQUIRED_MESSAGES.problem_description;
  }

  if (!values.name.trim()) errors.name = REQUIRED_MESSAGES.name;

  if (!values.email.trim()) {
    errors.email = REQUIRED_MESSAGES.email;
  } else if (!EMAIL_PATTERN.test(values.email.trim())) {
    errors.email = "Bitte eine gültige E-Mail-Adresse eingeben.";
  }

  if (!values.phone.trim()) errors.phone = REQUIRED_MESSAGES.phone;

  return errors;
}

function normalizePlate(value: string): string {
  return value.trim().toUpperCase().replace(/\s+/g, " ");
}

/* ------------------------------------------------------------------ */
/* Page                                                                */
/* ------------------------------------------------------------------ */

/**
 * Customer appointment request (route "/"). The request half of the approved
 * screen design (design/mockups/kunde.html): customer, vehicle, preferred date
 * and problem description, submitting POST /api/orders (AC-04).
 */
export default function RequestPage() {
  const [values, setValues] = useState<FormValues>(EMPTY_VALUES);
  const [touched, setTouched] = useState<Partial<Record<FieldName, boolean>>>({});
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [serverError, setServerError] = useState<ApiError | string | null>(null);
  const [orderNumber, setOrderNumber] = useState<string | null>(null);
  const firstFieldRef = useRef<HTMLInputElement | null>(null);

  const errors = validate(values);

  function errorVisible(name: FieldName): boolean {
    return (Boolean(touched[name]) || submitted) && Boolean(errors[name]);
  }

  function update(name: FieldName, value: string) {
    setValues((prev) => ({ ...prev, [name]: value }));
    setTouched((prev) => ({ ...prev, [name]: true }));
    setServerError(null);
  }

  function touch(name: FieldName) {
    setTouched((prev) => ({ ...prev, [name]: true }));
  }

  function reset() {
    setValues(EMPTY_VALUES);
    setTouched({});
    setSubmitted(false);
    setServerError(null);
    setOrderNumber(null);
    setSubmitting(false);
    window.setTimeout(() => firstFieldRef.current?.focus(), 0);
  }

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);

    const currentErrors = validate(values);
    const firstBad = (Object.keys(currentErrors) as FieldName[])[0];
    if (firstBad) {
      document.getElementById(FIELD_IDS[firstBad])?.focus();
      return;
    }

    setSubmitting(true);
    setServerError(null);
    try {
      const created = await createOrder({
        customer: {
          name: values.name.trim(),
          email: values.email.trim(),
          phone: values.phone.trim(),
        },
        vehicle: {
          license_plate: normalizePlate(values.license_plate),
          make: values.make.trim(),
          model: values.model.trim(),
          mileage_km: Number(values.mileage_km.replace(/\./g, "").trim()),
        },
        preferred_date: values.preferred_date,
        problem_description: values.problem_description.trim(),
      });
      setOrderNumber(created.order_number);
    } catch (error) {
      setServerError(error instanceof ApiError ? error : String(error));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1 id="anfrage-title">Werkstattauftrag anfragen</h1>
        <p className="page__subtitle">
          Beschreiben Sie Ihr Anliegen und Ihren Wunschtermin. Wir melden uns zur
          Bestätigung.
        </p>
      </header>

      <div className="card" data-od-id="anfrage-card">
        {orderNumber !== null ? (
          <Confirmation orderNumber={orderNumber} onReset={reset} />
        ) : (
          <>
            <ErrorSummary errors={errors} visible={submitted} />
            <Form
              values={values}
              errors={errors}
              errorVisible={errorVisible}
              submitting={submitting}
              serverError={serverError}
              firstFieldRef={firstFieldRef}
              onChange={update}
              onBlur={touch}
              onSubmit={onSubmit}
            />
          </>
        )}
      </div>
    </section>
  );
}

/* ------------------------------------------------------------------ */
/* Sub-components                                                      */
/* ------------------------------------------------------------------ */

interface ErrorSummaryProps {
  errors: Partial<Record<FieldName, string>>;
  visible: boolean;
}

function ErrorSummary({ errors, visible }: ErrorSummaryProps) {
  const names = Object.keys(errors) as FieldName[];
  if (!visible || names.length === 0) return null;

  return (
    <div
      id="anfrage-error-summary"
      className="alert alert--error"
      role="alert"
      data-od-id="anfrage-error-summary"
      style={{ marginBottom: "var(--space-4)" }}
    >
      <span>Bitte prüfen Sie die markierten Felder.</span>
      <ul style={{ margin: "var(--space-0) 0 0", paddingLeft: "var(--space-3)" }}>
        {names.map((name) => (
          <li key={name}>
            <a href={`#${FIELD_IDS[name]}`}>{errors[name]}</a>
          </li>
        ))}
      </ul>
    </div>
  );
}

interface FormProps {
  values: FormValues;
  errors: Partial<Record<FieldName, string>>;
  errorVisible: (name: FieldName) => boolean;
  submitting: boolean;
  serverError: ApiError | string | null;
  firstFieldRef: RefObject<HTMLInputElement | null>;
  onChange: (name: FieldName, value: string) => void;
  onBlur: (name: FieldName) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}

function Form({
  values,
  errors,
  errorVisible,
  submitting,
  serverError,
  firstFieldRef,
  onChange,
  onBlur,
  onSubmit,
}: FormProps) {
  return (
    <form
      id="anfrage-form"
      className="page"
      noValidate
      data-od-id="anfrage-form"
      onSubmit={onSubmit}
    >
      {serverError ? <ErrorBanner error={serverError} /> : null}

      <TextField
        field="license_plate"
        label="Kennzeichen"
        value={values.license_plate}
        error={errorVisible("license_plate") ? errors.license_plate : undefined}
        help="Format B-AB 1234"
        placeholder="B-AB 1234"
        mono
        autoComplete="off"
        inputRef={firstFieldRef}
        onChange={onChange}
        onBlur={onBlur}
      />
      <TextField
        field="make"
        label="Marke"
        value={values.make}
        error={errorVisible("make") ? errors.make : undefined}
        placeholder="z. B. Volkswagen"
        onChange={onChange}
        onBlur={onBlur}
      />
      <TextField
        field="model"
        label="Modell"
        value={values.model}
        error={errorVisible("model") ? errors.model : undefined}
        placeholder="z. B. Golf"
        onChange={onChange}
        onBlur={onBlur}
      />
      <TextField
        field="mileage_km"
        label="Kilometerstand"
        value={values.mileage_km}
        error={errorVisible("mileage_km") ? errors.mileage_km : undefined}
        help="Ganze Zahl in km, z. B. 128.450"
        placeholder="128450"
        numeric
        inputMode="numeric"
        onChange={onChange}
        onBlur={onBlur}
      />
      <TextField
        field="preferred_date"
        label="Wunschtermin"
        type="date"
        value={values.preferred_date}
        error={errorVisible("preferred_date") ? errors.preferred_date : undefined}
        onChange={onChange}
        onBlur={onBlur}
      />
      <TextField
        field="problem_description"
        label="Problembeschreibung"
        value={values.problem_description}
        error={
          errorVisible("problem_description") ? errors.problem_description : undefined
        }
        placeholder="Was ist aufgefallen? z. B. Motorleuchte leuchtet, Geräusch beim Bremsen"
        textarea
        onChange={onChange}
        onBlur={onBlur}
      />
      <TextField
        field="name"
        label="Ihr Name"
        value={values.name}
        error={errorVisible("name") ? errors.name : undefined}
        autoComplete="name"
        onChange={onChange}
        onBlur={onBlur}
      />
      <TextField
        field="email"
        label="E-Mail"
        type="email"
        value={values.email}
        error={errorVisible("email") ? errors.email : undefined}
        placeholder="name@beispiel.de"
        autoComplete="email"
        onChange={onChange}
        onBlur={onBlur}
      />
      <TextField
        field="phone"
        label="Telefon"
        type="tel"
        value={values.phone}
        error={errorVisible("phone") ? errors.phone : undefined}
        placeholder="030 1234567"
        autoComplete="tel"
        inputMode="tel"
        onChange={onChange}
        onBlur={onBlur}
      />

      <div
        className="form-submit-row"
        style={{ display: "flex", justifyContent: "flex-end" }}
      >
        <button
          type="submit"
          id="anfrage-submit"
          data-od-id="anfrage-submit"
          className="btn btn--primary btn--block-mobile"
          disabled={submitting}
          aria-busy={submitting}
        >
          {submitting ? "Wird gesendet…" : "Auftrag anfragen"}
        </button>
      </div>
    </form>
  );
}

interface TextFieldProps {
  field: FieldName;
  label: string;
  value: string;
  error?: string;
  help?: string;
  placeholder?: string;
  type?: string;
  autoComplete?: string;
  inputMode?: "numeric" | "tel" | "text";
  mono?: boolean;
  numeric?: boolean;
  textarea?: boolean;
  inputRef?: RefObject<HTMLInputElement | null>;
  onChange: (name: FieldName, value: string) => void;
  onBlur: (name: FieldName) => void;
}

function TextField({
  field,
  label,
  value,
  error,
  help,
  placeholder,
  type = "text",
  autoComplete,
  inputMode,
  mono = false,
  numeric = false,
  textarea = false,
  inputRef,
  onChange,
  onBlur,
}: TextFieldProps) {
  const id = FIELD_IDS[field];
  const helpId = `${id}-helper`;
  const className = [
    textarea ? "field__textarea" : "field__input",
    mono ? "field__input--mono" : "",
    numeric ? "field__input--numeric" : "",
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <div className={`field${error ? " field--error" : ""}`}>
      <label className="field__label" htmlFor={id}>
        {label} <span className="field__required">erforderlich</span>
      </label>
      {textarea ? (
        <textarea
          id={id}
          name={field}
          className={className}
          value={value}
          placeholder={placeholder}
          aria-invalid={error ? true : undefined}
          aria-describedby={helpId}
          onChange={(event) => onChange(field, event.target.value)}
          onBlur={() => onBlur(field)}
          required
        />
      ) : (
        <input
          id={id}
          name={field}
          className={className}
          type={type}
          value={value}
          placeholder={placeholder}
          autoComplete={autoComplete}
          inputMode={inputMode}
          ref={inputRef}
          aria-invalid={error ? true : undefined}
          aria-describedby={helpId}
          onChange={(event) => onChange(field, event.target.value)}
          onBlur={() => onBlur(field)}
          required
        />
      )}
      <div className="field__help" id={helpId} title={error ? "validation_error" : undefined}>
        {error ?? help ?? ""}
      </div>
    </div>
  );
}

interface ConfirmationProps {
  orderNumber: string;
  onReset: () => void;
}

function Confirmation({ orderNumber, onReset }: ConfirmationProps) {
  return (
    <div id="anfrage-confirmation" data-od-id="anfrage-confirmation">
      <div className="alert alert--success" role="status">
        <span>Ihre Anfrage ist eingegangen. Die Auftragsnummer lautet:</span>
      </div>
      <p
        className="mono tabular"
        style={{
          fontSize: "24px",
          fontWeight: 600,
          margin: "var(--space-3) 0 var(--space-1)",
        }}
      >
        {orderNumber}
      </p>
      <p style={{ color: "var(--color-fg-muted)" }}>
        Wir bestätigen den Termin per E-Mail. Den Status können Sie jederzeit mit
        Auftragsnummer und Kennzeichen abrufen.
      </p>
      <div
        style={{
          display: "flex",
          gap: "var(--space-1)",
          flexWrap: "wrap",
          marginTop: "var(--space-3)",
        }}
      >
        <Link className="btn btn--secondary" to="/status">
          Status abrufen
        </Link>
        <button
          type="button"
          id="anfrage-reset"
          data-od-id="anfrage-reset"
          className="btn btn--ghost"
          onClick={onReset}
        >
          Weitere Anfrage stellen
        </button>
      </div>
    </div>
  );
}
