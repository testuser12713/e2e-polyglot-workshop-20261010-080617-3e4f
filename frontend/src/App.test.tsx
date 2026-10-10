import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import "@testing-library/jest-dom/vitest";
import App from "./App";
import { AuthProvider } from "./auth/AuthContext";

const SESSION_KEY = "werkstattportal.session";

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthProvider>
        <App />
      </AuthProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
  window.localStorage.clear();
});

describe("App shell", () => {
  it("renders the wordmark and the main navigation", () => {
    renderAt("/");
    expect(
      screen.getByRole("link", { name: "Werkstattportal" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("navigation", { name: "Hauptnavigation" }),
    ).toBeInTheDocument();
  });

  it.each([
    ["/", "Werkstattauftrag anfragen"],
    ["/status", "Auftragsstatus"],
    ["/rechnung", "Rechnung"],
    ["/impressum", "Impressum"],
    ["/datenschutz", "Datenschutzerklärung"],
    ["/werkstatt/login", "Werkstatt-Anmeldung"],
  ])("resolves %s to its page", (path, heading) => {
    renderAt(path);
    expect(
      screen.getByRole("heading", { level: 1, name: heading }),
    ).toBeInTheDocument();
  });

  it.each(["/werkstatt/auftraege", "/werkstatt/dashboard"])(
    "redirects the unauthenticated workshop route %s to the login",
    (path) => {
      renderAt(path);
      expect(
        screen.getByRole("heading", { level: 1, name: "Werkstatt-Anmeldung" }),
      ).toBeInTheDocument();
    },
  );

  it("renders a workshop order for a signed-in employee", () => {
    window.localStorage.setItem(
      SESSION_KEY,
      JSON.stringify({
        token: "test-token",
        employee: { id: 1, name: "Anna Muster", email: "anna@example.org" },
      }),
    );
    renderAt("/werkstatt/auftraege/AU-2025-00001");
    expect(
      screen.getByRole("heading", { level: 1, name: /AU-2025-00001/ }),
    ).toBeInTheDocument();
    expect(screen.getByText("Anna Muster")).toBeInTheDocument();
  });

  it("shows the legal links from every page", () => {
    renderAt("/");
    expect(screen.getByRole("link", { name: "Impressum" })).toHaveAttribute(
      "href",
      "/impressum",
    );
    expect(
      screen.getByRole("link", { name: "Datenschutzerklärung" }),
    ).toHaveAttribute("href", "/datenschutz");
  });
});
