import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import "@testing-library/jest-dom/vitest";
import DatenschutzPage from "./DatenschutzPage";

afterEach(() => {
  cleanup();
});

describe("DatenschutzPage", () => {
  it("renders without a session and shows the page title", () => {
    render(<DatenschutzPage />);
    expect(
      screen.getByRole("heading", { level: 1, name: "Datenschutzerklärung" }),
    ).toBeInTheDocument();
  });

  it("shows every required section of the privacy notice", () => {
    render(<DatenschutzPage />);
    for (const heading of [
      "Verantwortliche Stelle",
      "Allgemeines",
      "Kundenbereich",
      "Statusabruf",
      "Werkstattbereich",
      "Benachrichtigungen",
      "Empfänger",
    ]) {
      expect(
        screen.getByRole("heading", { level: 2, name: heading }),
      ).toBeInTheDocument();
    }
  });

  it("explains which personal data is stored and for what purpose", () => {
    render(<DatenschutzPage />);
    expect(screen.getByText(/Namen, Ihre E-Mail-Adresse/)).toBeInTheDocument();
    expect(screen.getByText(/Telefonnummer, das Kennzeichen/)).toBeInTheDocument();
    expect(screen.getByText(/Ihre Problembeschreibung/)).toBeInTheDocument();
  });

  it("states that invoices are stored for the statutory retention period", () => {
    render(<DatenschutzPage />);
    expect(screen.getByText(/Rechnung wird für die Dauer/)).toBeInTheDocument();
  });

  it("states that no data leaves the site and no real e-mails are sent", () => {
    render(<DatenschutzPage />);
    expect(screen.getByText(/keine kommerziellen E-Mails/)).toBeInTheDocument();
    expect(
      screen.getByText(/keine Schriften, Skripte oder sonstigen Inhalte/),
    ).toBeInTheDocument();
  });
});
