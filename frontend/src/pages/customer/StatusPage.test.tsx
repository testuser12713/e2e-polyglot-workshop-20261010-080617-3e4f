import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import StatusPage from "./StatusPage";

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/status"]}>
      <StatusPage />
    </MemoryRouter>,
  );
}

const IN_PROGRESS = {
  order_number: "AU-2025-00123",
  status: "in_arbeit",
  status_history: [
    { status: "angefragt", changed_at: "2025-03-05T09:14:00Z" },
    { status: "bestaetigt", changed_at: "2025-03-05T11:30:00Z" },
    { status: "in_arbeit", changed_at: "2025-03-07T08:02:00Z" },
  ],
};

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("StatusPage", () => {
  it("renders the lookup form without showing an error before submit", () => {
    renderPage();

    expect(
      screen.getByRole("heading", { level: 1, name: "Auftragsstatus" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/Auftragsnummer/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Kennzeichen/)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Status anzeigen" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("shows the current status and the history for a matching entry", async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(jsonResponse(IN_PROGRESS));
    renderPage();

    await user.type(screen.getByLabelText(/Auftragsnummer/), "AU-2025-00123");
    await user.type(screen.getByLabelText(/Kennzeichen/), "B-AB 1234");
    await user.click(screen.getByRole("button", { name: "Status anzeigen" }));

    expect(await screen.findByText("AU-2025-00123")).toBeInTheDocument();
    // "In Arbeit" appears both as the status badge and in the timeline.
    expect((await screen.findAllByText("In Arbeit")).length).toBeGreaterThanOrEqual(2);
    expect(
      document.querySelector('[data-od-id="status-timeline"]'),
    ).not.toBeNull();

    const calledUrl = String(fetchMock.mock.calls[0][0]);
    expect(calledUrl).toContain("/api/orders/status?");
    expect(calledUrl).toContain("order_number=AU-2025-00123");
    expect(calledUrl).toContain("license_plate=B-AB+1234");
  });

  it("shows a friendly message and no order data when nothing matches", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse({ error: { code: "not_found", message: "order not found" } }, 404),
    );
    renderPage();

    await user.type(screen.getByLabelText(/Auftragsnummer/), "AU-2025-99999");
    await user.type(screen.getByLabelText(/Kennzeichen/), "B-ZZ 0000");
    await user.click(screen.getByRole("button", { name: "Status anzeigen" }));

    expect(
      await screen.findByText(/kein Auftrag mit dem angegebenen Kennzeichen/),
    ).toBeInTheDocument();
    expect(
      document.querySelector('[data-od-id="status-result"]'),
    ).toBeNull();
    expect(screen.queryByText("Statusverlauf")).not.toBeInTheDocument();
  });

  it("offers the invoice once the order is fertig", async () => {
    const user = userEvent.setup();
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse({
        order_number: "AU-2025-00118",
        status: "fertig",
        status_history: [
          { status: "angefragt", changed_at: "2025-02-24T14:05:00Z" },
          { status: "fertig", changed_at: "2025-02-27T16:40:00Z" },
        ],
      }),
    );
    renderPage();

    await user.type(screen.getByLabelText(/Auftragsnummer/), "AU-2025-00118");
    await user.type(screen.getByLabelText(/Kennzeichen/), "B-XY 7788");
    await user.click(screen.getByRole("button", { name: "Status anzeigen" }));

    const link = await screen.findByRole("link", {
      name: /Rechnung AU-2025-00118 ansehen/,
    });
    expect(link).toHaveAttribute(
      "href",
      expect.stringContaining("/rechnung?order_number=AU-2025-00118"),
    );
    expect(
      document.querySelector('[data-od-id="status-invoice-hint"]'),
    ).not.toBeNull();
  });
});
