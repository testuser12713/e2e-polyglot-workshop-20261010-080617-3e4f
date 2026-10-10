import { ApiError } from "../api/types";

export interface ErrorBannerProps {
  error?: ApiError | string | null;
  onDismiss?: () => void;
}

/**
 * Inline error alert showing a readable German sentence for a failed request.
 * The machine code is kept in the `title` attribute, never shown as the only
 * message (AC-18).
 */
export function ErrorBanner({ error, onDismiss }: ErrorBannerProps) {
  if (!error) return null;

  const message = typeof error === "string" ? error : error.message;
  const code = error instanceof ApiError ? error.code : undefined;

  return (
    <div className="alert alert--error" role="alert">
      <span title={code}>{message}</span>
      {onDismiss ? (
        <button type="button" className="btn btn--ghost" onClick={onDismiss}>
          Schließen
        </button>
      ) : null}
    </div>
  );
}
