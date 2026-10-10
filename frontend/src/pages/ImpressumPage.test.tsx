import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import "@testing-library/jest-dom/vitest";
import ImpressumPage from "./ImpressumPage";

afterEach(() => {
  cleanup();
});

describe("ImpressumPage", () => {
  it("renders without a session and shows the page title", () => {
    render(<ImpressumPage />);
    expect(
      screen.getByRole("heading", { level: 1, name: "Impressum" }),
    ).toBeInTheDocument();
  });

  it("shows every required section of the legal notice", () => {
    render(<ImpressumPage />);
    for (const heading of [
      "Anbieter",
      "Vertreten durch",
      "Kontakt",
      "Registereintrag",
      "Umsatzsteuer-Identifikationsnummer",
      "Verbraucherstreitbeilegung",
      "Haftung für Inhalte und Links",
    ]) {
      expect(
        screen.getByRole("heading", { level: 2, name: heading }),
      ).toBeInTheDocument();
    }
  });

  it("names the workshop, its operator and its contact details", () => {
    render(<ImpressumPage />);
    expect(screen.getByText(/Beispielwerkstatt GmbH/)).toBeInTheDocument();
    expect(screen.getByText(/Anna Beispiel/)).toBeInTheDocument();
    expect(screen.getByText(/030 1234567/)).toBeInTheDocument();
    expect(
      screen.getByText(/info@beispielwerkstatt\.example/),
    ).toBeInTheDocument();
  });
});
