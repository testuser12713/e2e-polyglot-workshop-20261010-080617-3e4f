import { afterEach, describe, expect, it, vi } from "vitest";
import { API_BASE_URL, apiFetch, resolveUrl } from "./client";
import { ApiError } from "./types";

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

describe("resolveUrl", () => {
  it("joins relative paths to the configured base URL", () => {
    expect(resolveUrl("/api/health")).toBe(`${API_BASE_URL}/api/health`);
  });

  it("leaves absolute URLs untouched", () => {
    expect(resolveUrl("https://example.org/x")).toBe("https://example.org/x");
  });
});

describe("apiFetch", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("parses a successful JSON body", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(jsonResponse({ status: "ok" }));

    const result = await apiFetch<{ status: string }>("/api/health");

    expect(result).toEqual({ status: "ok" });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe(`${API_BASE_URL}/api/health`);
  });

  it("throws an ApiError carrying the machine code from the error envelope", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      jsonResponse({ error: { code: "not_found", message: "weg" } }, 404),
    );

    await expect(apiFetch("/api/orders/status")).rejects.toMatchObject({
      name: "ApiError",
      code: "not_found",
      status: 404,
    });
  });

  it("throws an ApiError when the network request fails", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(
      new TypeError("Failed to fetch"),
    );

    await expect(apiFetch("/api/health")).rejects.toBeInstanceOf(ApiError);
  });
});
