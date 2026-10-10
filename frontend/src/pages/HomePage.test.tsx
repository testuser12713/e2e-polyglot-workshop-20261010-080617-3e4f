import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import "@testing-library/jest-dom/vitest";
import HomePage from "./HomePage";

function renderHome() {
  return render(
    <MemoryRouter>
      <HomePage />
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
});

describe("HomePage hub", () => {
  it("shows the welcome heading", () => {
    renderHome();
    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Willkommen im Werkstattportal",
      }),
    ).toBeInTheDocument();
  });

  it("renders both hub regions", () => {
    renderHome();
    expect(
      screen.getByRole("heading", { level: 2, name: "Kundenbereich" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { level: 2, name: "Werkstattbereich" }),
    ).toBeInTheDocument();
  });

  it("links the customer region into the customer area", () => {
    renderHome();
    expect(
      screen.getByRole("link", { name: "Zum Kundenbereich" }),
    ).toHaveAttribute("href", "/status");
    expect(
      screen.getByRole("link", { name: "Auftrag anfragen" }),
    ).toHaveAttribute("href", "/anfrage");
    expect(screen.getByRole("link", { name: "Status abrufen" })).toHaveAttribute(
      "href",
      "/status",
    );
    expect(
      screen.getByRole("link", { name: "Rechnung ansehen" }),
    ).toHaveAttribute("href", "/rechnung");
  });

  it("links the workshop region into the workshop area", () => {
    renderHome();
    expect(screen.getByRole("link", { name: "Zur Anmeldung" })).toHaveAttribute(
      "href",
      "/werkstatt/login",
    );
    expect(
      screen.getByRole("link", { name: "Anmeldung Mitarbeiter" }),
    ).toHaveAttribute("href", "/werkstatt/login");
    expect(screen.getByRole("link", { name: "Dashboard" })).toHaveAttribute(
      "href",
      "/werkstatt/dashboard",
    );
    expect(screen.getByRole("link", { name: "Auftragsliste" })).toHaveAttribute(
      "href",
      "/werkstatt/auftraege",
    );
  });

  it("keeps every hub link on the own origin", () => {
    renderHome();
    for (const link of screen.getAllByRole("link")) {
      const href = link.getAttribute("href") ?? "";
      expect(href.startsWith("/")).toBe(true);
      expect(href.startsWith("//")).toBe(false);
    }
  });
});
