import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import App from "../../App";
import { AuthProvider } from "../../auth/AuthContext";
import DashboardPage, { formatCents } from "./DashboardPage";

const SESSION_KEY = "werkstattportal.session";

const METRICS = {
  open_orders: 12,
  completed_today: 3,
  revenue_month_cents: 1874200,
};

const RECENT = {
  orders: [
    {
      order_number: "AU-2025-00118",
      status: "fertig",
      license_plate: "B-XY 7788",
      customer_name: "Erika Muster",
      preferred_date: "2025-02-27",
      total_cents: 45600,
    },
  ],
};

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

function signIn(): void {
  window.localStorage.setItem(
    SESSION_KEY,
    JSON.stringify({
      token: "test-token",
      employee: { id: 1, name: "Anna Muster", email: "anna@example.org" },
    }),
  );
}

function mockApi() {
  return vi.spyOn(globalThis, "fetch").mockImplementation((input) => {
    const url = String(input);
    if (url.includes("/api/workshop/dashboard")) {
      return Promise.resolve(jsonResponse(METRICS));
    }
    if (url.includes("/api/workshop/orders")) {
      return Promise.resolve(jsonResponse(RECENT));
    }
    return Promise.resolve(
      jsonResponse({ error: { code: "not_found", message: "weg" } }, 404),
    );
  });
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/werkstatt/dashboard"]}>
      <AuthProvider>
        <DashboardPage />
      </AuthProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  window.localStorage.clear();
});

describe("formatCents", () => {
  it("renders integer cents as German currency", () => {
    const formatted = formatCents(1874200);
    expect(formatted).toContain("18.742,00");
    expect(formatted).toContain("€");
  });

  it("renders zero cents as 0,00 €", () => {
    expect(formatCents(0)).toContain("0,00");
  });
});

describe("DashboardPage", () => {
  it("shows a loading state while the figures are fetched", () => {
    signIn();
    vi.spyOn(globalThis, "fetch").mockReturnValue(new Promise(() => {}));

    renderPage();

    expect(screen.getByText(/Dashboard wird geladen/)).toBeInTheDocument();
  });

  it("shows open orders, orders finished today and the month's revenue", async () => {
    signIn();
    mockApi();

    renderPage();

    expect(
      await screen.findByText("Offene Aufträge"),
    ).toBeInTheDocument();
    expect(screen.getByText("Heute fertig geworden")).toBeInTheDocument();
    expect(screen.getByText("Umsatz laufender Monat")).toBeInTheDocument();

    expect(screen.getByText("12")).toBeInTheDocument();
    expect(screen.getByText("3")).toBeInTheDocument();
    expect(screen.getByText(/18\.742,00/)).toBeInTheDocument();
  });

  it("renders the recently finished orders in the recent table", async () => {
    signIn();
    mockApi();

    renderPage();

    const table = await screen.findByRole("table");
    expect(table).toBeInTheDocument();
    expect(screen.getByText("AU-2025-00118")).toBeInTheDocument();
    expect(screen.getByText("B-XY 7788")).toBeInTheDocument();
    expect(screen.getByText("Erika Muster")).toBeInTheDocument();
  });

  it("loads the metrics with the bearer token from the protected route", async () => {
    signIn();
    const fetchMock = mockApi();

    renderPage();

    await screen.findByText("Offene Aufträge");
    const dashboardCall = fetchMock.mock.calls.find((call) =>
      String(call[0]).includes("/api/workshop/dashboard"),
    );
    expect(dashboardCall).toBeDefined();
    const init = dashboardCall?.[1] as RequestInit | undefined;
    const headers = new Headers(init?.headers);
    expect(headers.get("Authorization")).toBe("Bearer test-token");
  });

  it("shows the shared error message when the request fails", async () => {
    signIn();
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse(
        { error: { code: "internal_error", message: "kaputt" } },
        500,
      ),
    );

    renderPage();

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent(
      "Es ist ein Fehler aufgetreten. Bitte versuchen Sie es erneut.",
    );
  });

  it("redirects an unauthenticated visitor to the workshop login", () => {
    render(
      <MemoryRouter initialEntries={["/werkstatt/dashboard"]}>
        <AuthProvider>
          <App />
        </AuthProvider>
      </MemoryRouter>,
    );

    expect(
      screen.getByRole("heading", {
        level: 1,
        name: "Anmeldung Werkstattbereich",
      }),
    ).toBeInTheDocument();
  });
});
