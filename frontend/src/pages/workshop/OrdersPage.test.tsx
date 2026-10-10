import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import "@testing-library/jest-dom/vitest";
import { AuthProvider } from "../../auth/AuthContext";
import type { WorkshopOrderSummary } from "../../api/workshop";
import OrdersPage from "./OrdersPage";

const SESSION_KEY = "werkstattportal.session";
const TOKEN = "test-token";

const ORDERS: WorkshopOrderSummary[] = [
  {
    order_number: "AU-2025-00124",
    status: "angefragt",
    license_plate: "K-EF 9012",
    customer_name: "Erika Muster",
    preferred_date: "2025-03-18",
    total_cents: 0,
  },
  {
    order_number: "AU-2025-00122",
    status: "bestaetigt",
    license_plate: "B-AB 1234",
    customer_name: "Hans Beispiel",
    preferred_date: "2025-03-14",
    total_cents: 12500,
  },
  {
    order_number: "AU-2025-00118",
    status: "fertig",
    license_plate: "B-XY 7788",
    customer_name: "Petra Kunde",
    preferred_date: "2025-02-28",
    total_cents: 45600,
  },
];

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

/** Filter the fixture the way the API would, based on the query string. */
function mockApi() {
  return vi.spyOn(globalThis, "fetch").mockImplementation((input) => {
    const url = new URL(String(input));
    if (url.pathname !== "/api/workshop/orders") {
      return Promise.resolve(
        jsonResponse({ error: { code: "not_found", message: "weg" } }, 404),
      );
    }

    const status = url.searchParams.get("status");
    const plate = url.searchParams.get("license_plate");
    let result = ORDERS;
    if (status) {
      result = result.filter((order) => order.status === status);
    }
    if (plate) {
      const needle = plate.toUpperCase();
      result = result.filter((order) =>
        order.license_plate.toUpperCase().includes(needle),
      );
    }
    return Promise.resolve(jsonResponse({ orders: result }));
  });
}

function signIn(): void {
  window.localStorage.setItem(
    SESSION_KEY,
    JSON.stringify({
      token: TOKEN,
      employee: { id: 1, name: "Anna Muster", email: "anna@example.org" },
    }),
  );
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/werkstatt/auftraege"]}>
      <AuthProvider>
        <OrdersPage />
      </AuthProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  signIn();
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  window.localStorage.clear();
});

describe("OrdersPage", () => {
  it("renders all mocked orders with their columns", async () => {
    mockApi();

    renderPage();

    expect(await screen.findByText("AU-2025-00124")).toBeInTheDocument();
    expect(screen.getByText("AU-2025-00122")).toBeInTheDocument();
    expect(screen.getByText("AU-2025-00118")).toBeInTheDocument();
    expect(screen.getByText("K-EF 9012")).toBeInTheDocument();
    expect(screen.getByText("Erika Muster")).toBeInTheDocument();
    expect(screen.getByText("18.03.2025")).toBeInTheDocument();
  });

  it("loads the orders with the bearer token from the auth context", async () => {
    const fetchMock = mockApi();

    renderPage();

    await screen.findByText("AU-2025-00124");
    const call = fetchMock.mock.calls.find(([input]) =>
      String(input).includes("/api/workshop/orders"),
    );
    expect(call).toBeDefined();
    const init = call?.[1] as RequestInit | undefined;
    const headers = new Headers(init?.headers);
    expect(headers.get("Authorization")).toBe(`Bearer ${TOKEN}`);
  });

  it("filters by status and leaves only the matching rows", async () => {
    const fetchMock = mockApi();
    const user = userEvent.setup();

    renderPage();

    await screen.findByText("AU-2025-00124");
    await user.click(screen.getByRole("button", { name: "fertig" }));

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(([input]) =>
          String(input).includes("status=fertig"),
        ),
      ).toBe(true);
    });
    await waitFor(() => {
      expect(screen.queryByText("AU-2025-00124")).not.toBeInTheDocument();
    });
    expect(screen.getByText("AU-2025-00118")).toBeInTheDocument();
    expect(screen.queryByText("AU-2025-00122")).not.toBeInTheDocument();
  });

  it("searches by licence plate and leaves only the matching rows", async () => {
    const fetchMock = mockApi();
    const user = userEvent.setup();

    renderPage();

    await screen.findByText("AU-2025-00124");
    await user.type(
      screen.getByRole("searchbox", { name: "Kennzeichen suchen" }),
      "B-AB",
    );

    await waitFor(() => {
      expect(
        fetchMock.mock.calls.some(([input]) =>
          String(input).includes("license_plate=B-AB"),
        ),
      ).toBe(true);
    });
    await waitFor(() => {
      expect(screen.queryByText("AU-2025-00124")).not.toBeInTheDocument();
    });
    expect(screen.getByText("B-AB 1234")).toBeInTheDocument();
    expect(screen.queryByText("AU-2025-00118")).not.toBeInTheDocument();
  });

  it("shows the empty state when no order matches the filter", async () => {
    mockApi();
    const user = userEvent.setup();

    renderPage();

    await screen.findByText("AU-2025-00124");
    await user.click(screen.getByRole("button", { name: "abgeholt" }));

    expect(
      await screen.findByText("Keine Aufträge gefunden"),
    ).toBeInTheDocument();
  });

  it("shows the shared error message when the request fails", async () => {
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

  it("redirects to the workshop login on a 401", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse({ error: { code: "unauthorized", message: "nein" } }, 401),
    );

    render(
      <MemoryRouter initialEntries={["/werkstatt/auftraege"]}>
        <AuthProvider>
          <Routes>
            <Route path="/werkstatt/auftraege" element={<OrdersPage />} />
            <Route
              path="/werkstatt/login"
              element={<h1>Anmeldung Werkstattbereich</h1>}
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>,
    );

    expect(
      await screen.findByRole("heading", {
        name: "Anmeldung Werkstattbereich",
      }),
    ).toBeInTheDocument();
  });
});
