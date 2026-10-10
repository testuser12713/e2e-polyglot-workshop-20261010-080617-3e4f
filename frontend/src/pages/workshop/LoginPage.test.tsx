import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import App from "../../App";
import { AuthProvider } from "../../auth/AuthContext";
import LoginPage from "./LoginPage";

const SESSION_KEY = "werkstattportal.session";

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

function renderLogin() {
  return render(
    <MemoryRouter initialEntries={["/werkstatt/login"]}>
      <AuthProvider>
        <Routes>
          <Route path="/werkstatt/login" element={<LoginPage />} />
          <Route
            path="/werkstatt/dashboard"
            element={<h1>Dashboard Bereich</h1>}
          />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

function emailField(): HTMLInputElement {
  return screen.getByLabelText(/E-Mail-Adresse/) as HTMLInputElement;
}

function passwordField(): HTMLInputElement {
  return screen.getByLabelText(/Passwort/) as HTMLInputElement;
}

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  window.localStorage.clear();
});

describe("LoginPage", () => {
  it("renders the login card with heading and form regions", () => {
    renderLogin();

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Anmeldung Werkstattbereich",
      }),
    ).toBeInTheDocument();
    expect(document.getElementById("login-form")).toBeInTheDocument();
    expect(document.getElementById("login-submit")).toBeInTheDocument();
    // Neutral start: no validation message before the user acts.
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("signs in with valid credentials and opens the workshop area", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse({
        token: "session-token",
        employee: { id: 1, name: "Anna Muster", email: "anna@example.org" },
      }),
    );

    renderLogin();
    await user.type(emailField(), "anna@example.org");
    await user.type(passwordField(), "geheim");
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    expect(
      await screen.findByRole(
        "heading",
        { name: "Dashboard Bereich" },
        { timeout: 3000 },
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();

    const stored = JSON.parse(
      window.localStorage.getItem(SESSION_KEY) ?? "null",
    ) as { employee?: { email?: string } } | null;
    expect(stored?.employee?.email).toBe("anna@example.org");
  });

  it("keeps the e-mail field and shows a message on a wrong password", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse(
        { error: { code: "unauthorized", message: "invalid credentials" } },
        401,
      ),
    );

    renderLogin();
    await user.type(emailField(), "anna@example.org");
    await user.type(passwordField(), "falsch");
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent(/fehlgeschlagen/i);
    expect(emailField().value).toBe("anna@example.org");
    expect(passwordField().value).toBe("");
    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Anmeldung Werkstattbereich",
      }),
    ).toBeInTheDocument();
  });

  it("shows a rate-limit notice on a 429 response", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse(
        { error: { code: "rate_limited", message: "too many attempts" } },
        429,
      ),
    );

    renderLogin();
    await user.type(emailField(), "anna@example.org");
    await user.type(passwordField(), "falsch");
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /zu viele fehlversuche/i,
    );
  });

  it("does not treat an untouched form as an error", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");
    const user = userEvent.setup();

    renderLogin();
    // Empty submit: a form-level hint appears, but no request is sent.
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(
      /eingeben/i,
    );
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("logs out again from the workshop nav and returns to the login", async () => {
    const user = userEvent.setup();
    window.localStorage.setItem(
      SESSION_KEY,
      JSON.stringify({
        token: "session-token",
        employee: { id: 1, name: "Anna Muster", email: "anna@example.org" },
      }),
    );

    render(
      <MemoryRouter initialEntries={["/werkstatt/dashboard"]}>
        <AuthProvider>
          <App />
        </AuthProvider>
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole("button", { name: "Abmelden" }));

    expect(
      await screen.findByRole("heading", {
        level: 1,
        name: "Anmeldung Werkstattbereich",
      }),
    ).toBeInTheDocument();
    expect(window.localStorage.getItem(SESSION_KEY)).toBeNull();
  });
});
