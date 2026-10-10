import { ApiError, type ApiErrorEnvelope, type ApiErrorCode } from "./types";

/**
 * Base URL of the workshop API.
 *
 * Comes from VITE_API_BASE_URL (declared in RUN.json as the api service origin).
 * While the api skeleton is not merged the variable is unset, so a working
 * development default is used instead. Read lazily so the build never crashes
 * on a missing value.
 */
const FALLBACK_BASE_URL = "http://localhost:8080";

function readBaseUrl(): string {
  const raw =
    typeof import.meta.env?.VITE_API_BASE_URL === "string"
      ? import.meta.env.VITE_API_BASE_URL.trim()
      : "";
  if (!raw) return FALLBACK_BASE_URL;
  return raw.replace(/\/+$/, "");
}

export const API_BASE_URL = readBaseUrl();

/** Turn a relative API path into an absolute URL against the API base. */
export function resolveUrl(path: string): string {
  if (/^https?:\/\//i.test(path)) return path;
  const suffix = path.startsWith("/") ? path : `/${path}`;
  return `${API_BASE_URL}${suffix}`;
}

function isEnvelope(value: unknown): value is ApiErrorEnvelope {
  if (typeof value !== "object" || value === null) return false;
  const error = (value as { error?: unknown }).error;
  return typeof error === "object" && error !== null;
}

/**
 * Perform a JSON request against the API.
 *
 * On success the parsed body is returned as `T`. On any non-2xx response the
 * shared error envelope is parsed and an {@link ApiError} is thrown carrying the
 * machine code. Network failures and unreadable bodies also raise `ApiError`
 * rather than leaking a raw fetch error to the UI.
 */
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (!headers.has("Accept")) headers.set("Accept", "application/json");
  if (init.body != null && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  let response: Response;
  try {
    response = await fetch(resolveUrl(path), { ...init, headers });
  } catch {
    throw new ApiError("internal_error", "Der Server ist nicht erreichbar.", 0);
  }

  const text = await response.text();
  let data: unknown;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = undefined;
    }
  }

  if (!response.ok) {
    const envelope = isEnvelope(data) ? data : undefined;
    const code = envelope?.error?.code ?? "internal_error";
    const serverMessage = envelope?.error?.message;
    const message =
      isMappedCode(code) || !serverMessage ? undefined : serverMessage;
    throw new ApiError(code, message, response.status, envelope);
  }

  return data as T;
}

const MAPPED_CODES: ApiErrorCode[] = [
  "validation_error",
  "not_found",
  "conflict",
  "invalid_transition",
  "unauthorized",
  "rate_limited",
  "internal_error",
];

function isMappedCode(code: string): code is ApiErrorCode {
  return (MAPPED_CODES as string[]).includes(code);
}
