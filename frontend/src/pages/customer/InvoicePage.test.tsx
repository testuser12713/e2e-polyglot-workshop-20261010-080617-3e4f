import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import InvoicePage from "./InvoicePage";

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

const INVOICE = {
  order_number: "AU-2025-00118",
  items: [
    {
      kind: "labor",
      description: "Arbeitszeit — Bremsbeläge vorne",
      quantity: 75,
      unit_price_cents: 9500,
      total_cents: 11875,
    },
    {
      kind: "part",
      description: "Einzelteil — Bremsbelagsatz vorne",
      quantity: 1,
      unit_price_cents: 8420,
      total_cents: 8420,
    },
  ],
  net_cents: 20295,
  vat_cents: 3856,
  gross_cents: 24151,
  created_at: "2025-02-27T16:45:00Z",
};

const STATUS = {
  order_number: "AU-2025-00118",
  status: "fertig",
  status_history: [
    { status: "angefragt", changed_at: "2025-02-24T14:05:00Z" },
    { status: "fertig", changed_at: "2025-02-27T16:40:00Z" },
  ],
};

const QUERY_PATH =
  "/rechnung?order_number=AU-2025-00118&license_plate=B-XY%207788";

function renderPage(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <InvoicePage />
    </MemoryRouter>,
  );
}

function mockInvoiceApi() {
  return vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    const url = String(input);
    if (url.includes("/invoice")) return jsonResponse(INVOICE);
    if (url.includes("/api/orders/status")) return jsonResponse(STATUS);
    return jsonResponse(
      { error: { code: "not_found", message: "not found" } },
      404,
    );
  });
}

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("InvoicePage", () => {
  it("renders the positions, net, VAT and gross total from integer cents", async () => {
    mockInvoiceApi();
    renderPage(QUERY_PATH);

    expect(await screen.findByText("Rechnungspositionen")).toBeInTheDocument();
    expect(screen.getByText(/Bremsbelagsatz vorne/)).toBeInTheDocument();
    expect(screen.getByText("Nettobetrag")).toBeInTheDocument();
    expect(screen.getByText(/202,95/)).toBeInTheDocument();
    expect(screen.getByText("Mehrwertsteuer (19 %)")).toBeInTheDocument();
    expect(screen.getByText(/38,56/)).toBeInTheDocument();
    expect(screen.getByText("Gesamtbetrag")).toBeInTheDocument();
    expect(screen.getByText(/241,51/)).toBeInTheDocument();
  });

  it("shows the order status history and no download or payment control", async () => {
    mockInvoiceApi();
    renderPage(QUERY_PATH);

    await screen.findByText("Rechnungspositionen");

    expect(
      screen.getByRole("heading", { name: "Statusverlauf des Auftrags" }),
    ).toBeInTheDocument();
    expect(
      document.querySelector('[data-od-id="rechnung-timeline"]'),
    ).not.toBeNull();
    expect(screen.getByText(/nur einzusehen/)).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: /download|bezahl|zahlen/i }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: /download|bezahl|zahlen/i }),
    ).not.toBeInTheDocument();
  });

  it("offers a lookup form when opened without an order reference", () => {
    renderPage("/rechnung");

    expect(
      screen.getByRole("heading", { level: 1, name: "Rechnung" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Rechnung anzeigen" }),
    ).toBeInTheDocument();
    expect(screen.queryByText("Rechnungspositionen")).not.toBeInTheDocument();
  });

  it("shows a friendly message when no invoice matches", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse(
        { error: { code: "not_found", message: "order not found" } },
        404,
      ),
    );
    renderPage("/rechnung");

    await user.type(screen.getByLabelText(/Auftragsnummer/), "AU-2025-99999");
    await user.type(screen.getByLabelText(/Kennzeichen/), "B-ZZ 0000");
    await user.click(screen.getByRole("button", { name: "Rechnung anzeigen" }));

    expect(
      await screen.findByText(/keine Rechnung gefunden/),
    ).toBeInTheDocument();
    expect(screen.queryByText("Rechnungspositionen")).not.toBeInTheDocument();
  });
});
