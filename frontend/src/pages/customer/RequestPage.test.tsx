import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import { API_BASE_URL } from "../../api/client";
import RequestPage from "./RequestPage";

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

function renderPage() {
  return render(
    <MemoryRouter>
      <RequestPage />
    </MemoryRouter>,
  );
}

async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText(/Kennzeichen/), "B-AB 1234");
  await user.type(screen.getByLabelText(/Marke/), "Volkswagen");
  await user.type(screen.getByLabelText(/Modell/), "Golf");
  await user.type(screen.getByLabelText(/Kilometerstand/), "128450");
  fireEvent.change(screen.getByLabelText(/Wunschtermin/), {
    target: { value: "2025-03-12" },
  });
  await user.type(
    screen.getByLabelText(/Problembeschreibung/),
    "Motorleuchte leuchtet.",
  );
  await user.type(screen.getByLabelText(/Ihr Name/), "Anna Muster");
  await user.type(screen.getByLabelText(/E-Mail/), "anna@example.org");
  await user.type(screen.getByLabelText(/Telefon/), "030 1234567");
}

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

describe("RequestPage", () => {
  it("renders the request form with its controls wired", () => {
    renderPage();
    expect(
      screen.getByRole("heading", { level: 1, name: "Werkstattauftrag anfragen" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Auftrag anfragen" }),
    ).toBeEnabled();
    expect(screen.getByLabelText(/Wunschtermin/)).toHaveAttribute("type", "date");
    expect(screen.getByLabelText(/Problembeschreibung/)).toBeInTheDocument();
  });

  it("reports untouched required fields instead of submitting", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Auftrag anfragen" }));

    expect(
      screen.getByRole("alert"),
    ).toHaveTextContent("Bitte prüfen Sie die markierten Felder.");
    expect(
      screen.getAllByText("Kennzeichen ist erforderlich."),
    ).not.toHaveLength(0);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("reports an invalid e-mail without submitting", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch");
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText(/E-Mail/), "keine-mail");
    await user.click(screen.getByRole("button", { name: "Auftrag anfragen" }));

    expect(
      screen.getAllByText("Bitte eine gültige E-Mail-Adresse eingeben."),
    ).not.toHaveLength(0);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("submits a valid request and shows the returned order number", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse(
        {
          order_number: "AU-2025-00124",
          status: "angefragt",
          created_at: "2025-03-05T09:14:00Z",
        },
        201,
      ),
    );
    const user = userEvent.setup();
    renderPage();

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Auftrag anfragen" }));

    await waitFor(() =>
      expect(screen.getByText("AU-2025-00124")).toBeInTheDocument(),
    );

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe(`${API_BASE_URL}/api/orders`);
    expect(init?.method).toBe("POST");
    const body = JSON.parse(String(init?.body));
    expect(body.customer).toEqual({
      name: "Anna Muster",
      email: "anna@example.org",
      phone: "030 1234567",
    });
    expect(body.vehicle).toEqual({
      license_plate: "B-AB 1234",
      make: "Volkswagen",
      model: "Golf",
      mileage_km: 128450,
    });
    expect(body.preferred_date).toBe("2025-03-12");
    expect(body.problem_description).toBe("Motorleuchte leuchtet.");

    expect(
      screen.getByRole("button", { name: "Weitere Anfrage stellen" }),
    ).toBeInTheDocument();
  });

  it("resets back to an empty form after confirmation", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse(
        {
          order_number: "AU-2025-00124",
          status: "angefragt",
          created_at: "2025-03-05T09:14:00Z",
        },
        201,
      ),
    );
    const user = userEvent.setup();
    renderPage();

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Auftrag anfragen" }));
    await waitFor(() =>
      expect(screen.getByText("AU-2025-00124")).toBeInTheDocument(),
    );

    await user.click(
      screen.getByRole("button", { name: "Weitere Anfrage stellen" }),
    );

    expect(screen.getByLabelText(/Kennzeichen/)).toHaveValue("");
    expect(
      screen.getByRole("button", { name: "Auftrag anfragen" }),
    ).toBeInTheDocument();
  });

  it("shows the server validation error from the shared envelope", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse(
        {
          error: {
            code: "validation_error",
            message: "Die E-Mail-Adresse ist bereits bekannt.",
          },
        },
        400,
      ),
    );
    const user = userEvent.setup();
    renderPage();

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Auftrag anfragen" }));

    await waitFor(() =>
      expect(screen.getByRole("alert")).toHaveTextContent(
        "Bitte prüfen Sie Ihre Eingaben.",
      ),
    );
  });
});
