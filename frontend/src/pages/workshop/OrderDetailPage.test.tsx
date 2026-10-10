import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import "@testing-library/jest-dom/vitest";
import { AuthProvider } from "../../auth/AuthContext";
import { ApiError, type OrderDetail, type Status } from "../../api/types";
import OrderDetailPage from "./OrderDetailPage";
import {
  addWorkshopOrderItems,
  confirmWorkshopOrder,
  getWorkshopOrder,
  setWorkshopOrderStatus,
} from "../../api/workshopOrder";

vi.mock("../../api/workshopOrder", () => ({
  getWorkshopOrder: vi.fn(),
  confirmWorkshopOrder: vi.fn(),
  addWorkshopOrderItems: vi.fn(),
  setWorkshopOrderStatus: vi.fn(),
}));

const ORDER_NUMBER = "AU-2025-00121";
const SESSION_KEY = "werkstattportal.session";
const TOKEN = "test-token";

const mockGet = vi.mocked(getWorkshopOrder);
const mockConfirm = vi.mocked(confirmWorkshopOrder);
const mockAddItems = vi.mocked(addWorkshopOrderItems);
const mockSetStatus = vi.mocked(setWorkshopOrderStatus);

function makeOrder(overrides: Partial<OrderDetail> = {}): OrderDetail {
  return {
    order_number: ORDER_NUMBER,
    status: "in_arbeit",
    customer: {
      name: "T. Weber",
      email: "t.weber@beispiel.de",
      phone: "030 9876543",
    },
    vehicle: {
      license_plate: "S-GH 5566",
      make: "Audi",
      model: "A4 2.0 TDI",
      mileage_km: 84210,
    },
    preferred_date: "2025-03-12",
    problem_description: "Bremsen vorne quietschen beim Anhalten.",
    items: [],
    status_history: [
      { status: "angefragt", changed_at: "2025-03-05T17:48:00Z" },
      { status: "bestaetigt", changed_at: "2025-03-06T09:25:00Z" },
      { status: "in_arbeit", changed_at: "2025-03-07T08:10:00Z" },
    ],
    ...overrides,
  };
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={[`/werkstatt/auftraege/${ORDER_NUMBER}`]}>
      <AuthProvider>
        <Routes>
          <Route
            path="/werkstatt/auftraege/:orderNumber"
            element={<OrderDetailPage />}
          />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  window.localStorage.setItem(
    SESSION_KEY,
    JSON.stringify({
      token: TOKEN,
      employee: { id: 1, name: "Anna Muster", email: "anna@example.org" },
    }),
  );
});

afterEach(() => {
  cleanup();
  window.localStorage.clear();
  vi.clearAllMocks();
});

describe("OrderDetailPage", () => {
  it("loads the order and shows vehicle, customer and problem", async () => {
    mockGet.mockResolvedValue(makeOrder());

    renderPage();

    expect(
      screen.getByRole("heading", { level: 1, name: new RegExp(ORDER_NUMBER) }),
    ).toBeInTheDocument();
    expect(await screen.findByText("S-GH 5566")).toBeInTheDocument();
    expect(screen.getByText("T. Weber")).toBeInTheDocument();
    expect(
      screen.getByText("Bremsen vorne quietschen beim Anhalten."),
    ).toBeInTheDocument();
    expect(mockGet).toHaveBeenCalledWith(ORDER_NUMBER, TOKEN);
  });

  it("shows the empty state when no positions exist", async () => {
    mockGet.mockResolvedValue(makeOrder({ items: [] }));

    renderPage();

    expect(
      await screen.findByText("Noch keine Positionen erfasst"),
    ).toBeInTheDocument();
    expect(
      document.querySelector('[data-od-id="positions-empty"]'),
    ).not.toBeNull();
  });

  it("renders positions with their totals", async () => {
    mockGet.mockResolvedValue(
      makeOrder({
        items: [
          {
            kind: "labor",
            description: "Bremsbeläge vorne",
            quantity: 90,
            unit_price_cents: 9500,
            total_cents: 14250,
          },
          {
            kind: "part",
            description: "Bremsbelagsatz vorne",
            quantity: 1,
            unit_price_cents: 8420,
            total_cents: 8420,
          },
        ],
      }),
    );

    renderPage();

    expect(
      await screen.findByText("Arbeitszeit — Bremsbeläge vorne"),
    ).toBeInTheDocument();
    expect(screen.getByText("1 h 30 min")).toBeInTheDocument();
    expect(
      document.querySelector('[data-od-id="positions-table"]'),
    ).not.toBeNull();
    expect(screen.getByText("226,70 €")).toBeInTheDocument();
    expect(screen.getByText("269,77 €")).toBeInTheDocument();
  });

  it("offers the confirm action only while the order is angefragt", async () => {
    mockGet.mockResolvedValue(makeOrder({ status: "angefragt" }));
    mockConfirm.mockResolvedValue(makeOrder({ status: "bestaetigt" }));
    const user = userEvent.setup();

    renderPage();

    const confirm = await screen.findByRole("button", {
      name: "Auftrag bestätigen",
    });
    expect(
      screen.queryByRole("button", { name: "Status setzen" }),
    ).not.toBeInTheDocument();

    await user.click(confirm);

    await waitFor(() => {
      expect(mockConfirm).toHaveBeenCalledWith(ORDER_NUMBER, TOKEN);
    });
    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(2);
    });
  });

  it("hides the confirm action once the order is confirmed", async () => {
    mockGet.mockResolvedValue(makeOrder({ status: "bestaetigt" }));

    renderPage();

    expect(
      await screen.findByRole("button", { name: "Status setzen" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Auftrag bestätigen" }),
    ).not.toBeInTheDocument();
  });

  it("captures a labor position as minutes and integer cents", async () => {
    mockGet.mockResolvedValue(makeOrder({ status: "bestaetigt" }));
    mockAddItems.mockResolvedValue(makeOrder({ status: "bestaetigt" }));
    const user = userEvent.setup();

    renderPage();

    await user.click(
      (await screen.findAllByRole("button", {
        name: "Positionen erfassen",
      }))[0],
    );

    const dialog = await screen.findByRole("dialog", {
      name: "Positionen erfassen",
    });
    expect(dialog).toBeInTheDocument();

    await user.type(
      screen.getByLabelText(/Beschreibung/),
      "Bremsbeläge vorne",
    );
    await user.type(screen.getByLabelText(/Dauer/), "1,5");
    await user.type(screen.getByLabelText(/Einzelpreis/), "95,00");
    await user.click(
      screen.getByRole("button", { name: "Position hinzufügen" }),
    );

    await waitFor(() => {
      expect(mockAddItems).toHaveBeenCalledWith(
        ORDER_NUMBER,
        [
          {
            kind: "labor",
            description: "Bremsbeläge vorne",
            quantity: 90,
            unit_price_cents: 9500,
          },
        ],
        TOKEN,
      );
    });
  });

  it("advances the status by exactly one step after confirmation", async () => {
    mockGet.mockResolvedValue(makeOrder({ status: "bestaetigt" }));
    mockSetStatus.mockResolvedValue(makeOrder({ status: "in_arbeit" }));
    const user = userEvent.setup();

    renderPage();

    await user.click(await screen.findByRole("button", { name: "Status setzen" }));

    const dialog = await screen.findByRole("dialog", { name: "Status setzen" });
    expect(dialog).toBeInTheDocument();
    // Only the next allowed status is offered.
    expect(screen.getByRole("option", { name: "in Arbeit" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "abgeholt" })).toBeNull();

    await user.click(
      screen.getByRole("button", {
        name: `Auf ${ORDER_NUMBER} auf „in Arbeit“ setzen`,
      }),
    );

    await waitFor(() => {
      expect(mockSetStatus).toHaveBeenCalledWith(ORDER_NUMBER, "in_arbeit", TOKEN);
    });
    await waitFor(() => {
      expect(mockGet).toHaveBeenCalledTimes(2);
    });
  });

  it("shows an invalid transition from the API as a readable message", async () => {
    mockGet.mockResolvedValue(makeOrder({ status: "bestaetigt" }));
    mockSetStatus.mockRejectedValue(
      new ApiError("invalid_transition", undefined, 400),
    );
    const user = userEvent.setup();

    renderPage();

    await user.click(await screen.findByRole("button", { name: "Status setzen" }));
    await user.click(
      await screen.findByRole("button", {
        name: `Auf ${ORDER_NUMBER} auf „in Arbeit“ setzen`,
      }),
    );

    expect(
      await screen.findByText("Dieser Statuswechsel ist nicht möglich."),
    ).toBeInTheDocument();
    // The modal stays open so the employee sees the reason.
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("shows a readable error when the order cannot be loaded", async () => {
    mockGet.mockRejectedValue(new ApiError("not_found", undefined, 404));

    renderPage();

    expect(
      await screen.findByText("Es wurde nichts zu diesen Angaben gefunden."),
    ).toBeInTheDocument();
  });

  it("disables the controls the current status does not allow", async () => {
    const status: Status = "abgeholt";
    mockGet.mockResolvedValue(makeOrder({ status }));

    renderPage();

    await screen.findByText("S-GH 5566");
    expect(
      screen.queryByRole("button", { name: "Auftrag bestätigen" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Status setzen" }),
    ).not.toBeInTheDocument();
    const [positionsButton] = screen.getAllByRole("button", {
      name: "Positionen erfassen",
    });
    expect(positionsButton).toBeDisabled();
  });
});
