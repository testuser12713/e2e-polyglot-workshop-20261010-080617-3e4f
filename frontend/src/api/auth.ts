import type { Employee } from "../auth/AuthContext";
import { apiFetch } from "./client";

/** Credentials sent to POST /api/auth/login. */
export interface LoginCredentials {
  email: string;
  password: string;
}

/** Successful response of POST /api/auth/login. */
export interface LoginResult {
  token: string;
  employee: Employee;
}

/**
 * Authenticate an employee against the workshop API.
 *
 * Throws an {@link ApiError} carrying the shared machine code on failure:
 * `unauthorized` for a wrong e-mail/password and `rate_limited` once the API's
 * per-client rate limit trips.
 */
export function login(credentials: LoginCredentials): Promise<LoginResult> {
  return apiFetch<LoginResult>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({
      email: credentials.email,
      password: credentials.password,
    }),
  });
}
