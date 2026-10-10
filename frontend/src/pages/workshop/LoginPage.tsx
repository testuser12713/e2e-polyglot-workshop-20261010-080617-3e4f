import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { login } from "../../api/auth";
import { ApiError } from "../../api/types";
import { useAuth } from "../../auth/AuthContext";

/** How long the success notice stays visible before the redirect. */
const REDIRECT_DELAY_MS = 600;

const WRONG_CREDENTIALS_MESSAGE =
  "Die Anmeldung ist fehlgeschlagen. Bitte prüfen Sie E-Mail-Adresse und Passwort.";
const RATE_LIMITED_MESSAGE =
  "Zu viele Fehlversuche. Die Anmeldung ist für kurze Zeit gesperrt. Bitte versuchen Sie es später erneut.";
const GENERIC_FAILURE_MESSAGE =
  "Die Anmeldung ist fehlgeschlagen. Bitte versuchen Sie es erneut.";

/** 16px inline spinner shown while the sign-in request is in flight. */
function LoginSpinner() {
  return (
    <svg width="16" height="16" viewBox="0 0 16 16" aria-hidden="true" focusable="false">
      <circle
        cx="8"
        cy="8"
        r="6"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeDasharray="26 12"
        strokeLinecap="round"
      >
        <animateTransform
          attributeName="transform"
          type="rotate"
          from="0 8 8"
          to="360 8 8"
          dur="0.8s"
          repeatCount="indefinite"
        />
      </circle>
    </svg>
  );
}

/**
 * Workshop sign-in (route "/werkstatt/login").
 *
 * On success the session is stored in the auth context so ProtectedRoute lets
 * the workshop routes through; a wrong password keeps the e-mail field intact,
 * and a rate-limited response shows a distinct notice. Neither the password nor
 * the token is ever logged.
 */
export default function LoginPage() {
  const { isAuthenticated, signIn } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const passwordRef = useRef<HTMLInputElement>(null);

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [signedIn, setSignedIn] = useState(false);
  const [alert, setAlert] = useState<string | null>(null);

  const state = location.state as { from?: string } | null;
  const from = state?.from;
  const redirectTo =
    from && from.startsWith("/werkstatt") && from !== "/werkstatt/login"
      ? from
      : "/werkstatt/dashboard";

  // A visitor who is already signed in is sent straight on (mount only — the
  // success flow below drives its own redirect once the notice was shown).
  useEffect(() => {
    if (isAuthenticated) {
      navigate(redirectTo, { replace: true });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!signedIn) return;
    const timer = window.setTimeout(
      () => navigate(redirectTo, { replace: true }),
      REDIRECT_DELAY_MS,
    );
    return () => window.clearTimeout(timer);
  }, [signedIn, redirectTo, navigate]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (submitting || signedIn) return;

    const trimmedEmail = email.trim();
    if (!trimmedEmail || !password) {
      setAlert("Bitte E-Mail-Adresse und Passwort eingeben.");
      return;
    }

    setAlert(null);
    setSubmitting(true);
    try {
      const result = await login({ email: trimmedEmail, password });
      signIn(result.token, result.employee);
      setPassword("");
      setSubmitting(false);
      setSignedIn(true);
    } catch (error) {
      setSubmitting(false);
      if (error instanceof ApiError && error.code === "rate_limited") {
        setAlert(RATE_LIMITED_MESSAGE);
      } else if (error instanceof ApiError && error.code === "unauthorized") {
        // The e-mail stays in the field so the user only retypes the password.
        setAlert(WRONG_CREDENTIALS_MESSAGE);
        setPassword("");
        window.setTimeout(() => passwordRef.current?.focus(), 0);
      } else {
        setAlert(
          error instanceof ApiError ? error.message : GENERIC_FAILURE_MESSAGE,
        );
      }
    }
  }

  return (
    <section className="page page--reading">
      <header className="page__header">
        <h1>Anmeldung Werkstattbereich</h1>
        <p className="page__subtitle">
          Melden Sie sich mit Ihrem Mitarbeiter-Konto an. Kundenkonten gelten hier
          nicht.
        </p>
      </header>

      <div className="card" data-od-id="login-card">
        <div className="card__body">
          {alert ? (
            <div
              id="login-alert"
              className="alert alert--error"
              role="alert"
              data-od-id="login-alert"
            >
              <span>{alert}</span>
            </div>
          ) : null}

          {signedIn ? (
            <div id="login-success" data-od-id="login-success">
              <div className="alert alert--success" role="status">
                <span>Anmeldung erfolgreich. Sie werden weitergeleitet…</span>
              </div>
              <div style={{ marginTop: "var(--space-3)" }}>
                <Link className="btn btn--primary" to={redirectTo}>
                  Zum Dashboard
                </Link>
              </div>
            </div>
          ) : (
            <form
              id="login-form"
              className="form-stack"
              style={{ display: "flex", flexDirection: "column", gap: "20px" }}
              noValidate
              data-od-id="login-form"
              onSubmit={handleSubmit}
            >
              <div className="field">
                <label className="field__label" htmlFor="login-user">
                  E-Mail-Adresse{" "}
                  <span className="field__required">erforderlich</span>
                </label>
                <input
                  id="login-user"
                  name="email"
                  type="email"
                  className="field__input"
                  autoComplete="username"
                  required
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  aria-describedby="login-user-helper"
                />
                <div className="field__help" id="login-user-helper" />
              </div>

              <div className="field">
                <label className="field__label" htmlFor="login-pass">
                  Passwort <span className="field__required">erforderlich</span>
                </label>
                <input
                  id="login-pass"
                  name="password"
                  type="password"
                  className="field__input"
                  autoComplete="current-password"
                  required
                  ref={passwordRef}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  aria-describedby="login-pass-helper"
                />
                <div className="field__help" id="login-pass-helper" />
              </div>

              <div style={{ marginTop: "var(--space-4)" }}>
                <button
                  id="login-submit"
                  data-od-id="login-submit"
                  type="submit"
                  className="btn btn--primary btn--block-mobile"
                  aria-label="Anmelden"
                  aria-busy={submitting}
                  disabled={submitting}
                >
                  {submitting ? <LoginSpinner /> : "Anmelden"}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>

      <p
        className="page__subtitle"
        style={{ fontSize: "var(--size-sm)" }}
      >
        Zugangsdaten erhalten Sie von der Werkstattleitung. Nach mehreren
        Fehlversuchen wird die Anmeldung zeitweise gesperrt.
      </p>
    </section>
  );
}
